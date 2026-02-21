// Package main demonstrates the worker pool pattern in Go.
// For PHP developers: Similar to Symfony Messenger workers, but in-process.
// PHP workers are separate processes; Go workers are goroutines sharing memory.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// Job represents a unit of work.
// PHP equivalent: Symfony Messenger message class
type Job struct {
	ID       int
	Payload  string
	Priority int
}

// Result represents the outcome of processing a job.
// PHP equivalent: Return value or event from handler
type Result struct {
	JobID    int
	Success  bool
	Output   string
	Duration time.Duration
}

// WorkerPool manages a pool of workers processing jobs.
// PHP equivalent: Messenger transport with multiple workers
type WorkerPool struct {
	numWorkers int
	jobs       chan Job
	results    chan Result
	wg         sync.WaitGroup
}

// NewWorkerPool creates a worker pool with the specified number of workers.
// PHP equivalent: bin/console messenger:consume --limit=N
func NewWorkerPool(numWorkers, jobQueueSize int) *WorkerPool {
	return &WorkerPool{
		numWorkers: numWorkers,
		jobs:       make(chan Job, jobQueueSize),
		results:    make(chan Result, jobQueueSize),
	}
}

// Start launches all workers.
// PHP equivalent: Starting multiple messenger:consume processes
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 1; i <= wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}
	log.Printf("Started %d workers", wp.numWorkers)
}

// worker processes jobs from the queue.
// PHP equivalent: MessageHandler invoked by Messenger
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d shutting down", id)
			return

		case job, ok := <-wp.jobs:
			if !ok {
				log.Printf("Worker %d: job channel closed", id)
				return
			}

			start := time.Now()
			log.Printf("Worker %d: processing job %d", id, job.ID)

			// Simulate work
			// PHP equivalent: MessageHandler::__invoke()
			output, success := processJob(job)

			wp.results <- Result{
				JobID:    job.ID,
				Success:  success,
				Output:   output,
				Duration: time.Since(start),
			}
		}
	}
}

// Submit adds a job to the queue.
// PHP equivalent: $bus->dispatch(new Message())
func (wp *WorkerPool) Submit(job Job) {
	wp.jobs <- job
}

// Results returns the results channel for reading.
func (wp *WorkerPool) Results() <-chan Result {
	return wp.results
}

// Close signals workers to stop and waits for completion.
// PHP equivalent: Graceful shutdown with SIGTERM
func (wp *WorkerPool) Close() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
}

// processJob simulates job processing.
func processJob(job Job) (string, bool) {
	// Simulate variable processing time
	duration := time.Duration(50+rand.Intn(150)) * time.Millisecond
	time.Sleep(duration)

	// Simulate occasional failures
	if rand.Float32() < 0.1 {
		return "random failure", false
	}

	return fmt.Sprintf("Processed: %s", job.Payload), true
}

// --- Rate-limited worker pool ---

// RateLimitedPool limits the rate of job processing.
// PHP equivalent: Rate limiting with Redis + Symfony RateLimiter
type RateLimitedPool struct {
	*WorkerPool
	ticker *time.Ticker
}

// NewRateLimitedPool creates a pool that processes at most rate jobs per second.
func NewRateLimitedPool(numWorkers, queueSize int, rate time.Duration) *RateLimitedPool {
	return &RateLimitedPool{
		WorkerPool: NewWorkerPool(numWorkers, queueSize),
		ticker:     time.NewTicker(rate),
	}
}

// SubmitRateLimited submits a job with rate limiting.
func (rp *RateLimitedPool) SubmitRateLimited(job Job) {
	<-rp.ticker.C // Wait for next tick
	rp.Submit(job)
}

// --- Semaphore for bounded concurrency ---

// Semaphore limits concurrent operations.
// PHP equivalent: Using a Redis-based semaphore or database locks
type Semaphore struct {
	sem chan struct{}
}

// NewSemaphore creates a semaphore with the given capacity.
func NewSemaphore(max int) *Semaphore {
	return &Semaphore{sem: make(chan struct{}, max)}
}

// Acquire blocks until a slot is available.
func (s *Semaphore) Acquire() {
	s.sem <- struct{}{}
}

// Release frees a slot.
func (s *Semaphore) Release() {
	<-s.sem
}

// --- Main demonstration ---

func main() {
	log.Println("=== Worker Pool Pattern Demo ===")

	// Create context with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create and start worker pool
	pool := NewWorkerPool(3, 100)
	pool.Start(ctx)

	// Collect results in background
	var results []Result
	var resultsMu sync.Mutex
	done := make(chan struct{})

	go func() {
		for result := range pool.Results() {
			resultsMu.Lock()
			results = append(results, result)
			resultsMu.Unlock()

			status := "✓"
			if !result.Success {
				status = "✗"
			}
			log.Printf("Result [%s] Job %d: %s (%v)", status, result.JobID, result.Output, result.Duration)
		}
		close(done)
	}()

	// Submit jobs
	log.Println("\nSubmitting 10 jobs...")
	for i := 1; i <= 10; i++ {
		pool.Submit(Job{
			ID:      i,
			Payload: fmt.Sprintf("Task-%d", i),
		})
	}

	// Close pool and wait for results
	pool.Close()
	<-done

	// Summary
	resultsMu.Lock()
	successful := 0
	var totalDuration time.Duration
	for _, r := range results {
		if r.Success {
			successful++
		}
		totalDuration += r.Duration
	}
	resultsMu.Unlock()

	log.Printf("\n=== Summary ===")
	log.Printf("Total jobs: %d", len(results))
	log.Printf("Successful: %d", successful)
	log.Printf("Failed: %d", len(results)-successful)
	log.Printf("Total processing time: %v", totalDuration)

	// Demonstrate semaphore
	log.Println("\n=== Semaphore Demo ===")
	sem := NewSemaphore(2)
	var wg sync.WaitGroup

	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sem.Acquire()
			log.Printf("Task %d acquired semaphore", id)
			time.Sleep(100 * time.Millisecond)
			log.Printf("Task %d releasing semaphore", id)
			sem.Release()
		}(i)
	}

	wg.Wait()
	log.Println("All tasks complete")
}
