// Package main demonstrates Go channel patterns.
// For PHP developers: This is completely new territory - PHP has no equivalent.
// Channels enable safe communication between goroutines without shared memory.
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	fmt.Println("=== Channel Patterns ===\n")

	// Run all examples
	unbufferedExample()
	bufferedExample()
	fanOutExample()
	fanInExample()
	pipelineExample()
	selectExample()
	timeoutExample()
	cancelExample()
}

// unbufferedExample demonstrates synchronous channel communication.
// Both sender and receiver must be ready at the same time.
func unbufferedExample() {
	fmt.Println("1. Unbuffered Channel (synchronous)")
	fmt.Println("-----------------------------------")

	// Create unbuffered channel
	// PHP: No equivalent - this is synchronisation
	ch := make(chan string)

	// Start receiver goroutine
	go func() {
		// This blocks until a value is sent
		msg := <-ch
		fmt.Printf("   Received: %s\n", msg)
	}()

	// Send blocks until receiver is ready
	ch <- "Hello from sender"

	time.Sleep(100 * time.Millisecond)
	fmt.Println()
}

// bufferedExample demonstrates asynchronous channel communication.
// Sender can send up to buffer size without blocking.
func bufferedExample() {
	fmt.Println("2. Buffered Channel (asynchronous)")
	fmt.Println("-----------------------------------")

	// Create buffered channel with capacity 3
	ch := make(chan int, 3)

	// These don't block - buffer has space
	ch <- 1
	ch <- 2
	ch <- 3
	fmt.Println("   Sent 3 values without blocking")

	// ch <- 4 would block here (buffer full)

	// Receive values
	fmt.Printf("   Received: %d, %d, %d\n", <-ch, <-ch, <-ch)
	fmt.Println()
}

// fanOutExample demonstrates distributing work to multiple workers.
// One producer, multiple consumers.
func fanOutExample() {
	fmt.Println("3. Fan-Out (one to many)")
	fmt.Println("------------------------")

	jobs := make(chan int, 10)
	var wg sync.WaitGroup

	// Start 3 workers
	for w := 1; w <= 3; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				fmt.Printf("   Worker %d processing job %d\n", workerID, job)
				time.Sleep(50 * time.Millisecond)
			}
		}(w)
	}

	// Send jobs
	for j := 1; j <= 6; j++ {
		jobs <- j
	}
	close(jobs) // Signal no more jobs

	wg.Wait()
	fmt.Println()
}

// fanInExample demonstrates collecting results from multiple sources.
// Multiple producers, one consumer.
func fanInExample() {
	fmt.Println("4. Fan-In (many to one)")
	fmt.Println("-----------------------")

	// Merge function combines multiple channels into one
	merge := func(channels ...<-chan string) <-chan string {
		out := make(chan string)
		var wg sync.WaitGroup

		for _, ch := range channels {
			wg.Add(1)
			go func(c <-chan string) {
				defer wg.Done()
				for v := range c {
					out <- v
				}
			}(ch)
		}

		go func() {
			wg.Wait()
			close(out)
		}()

		return out
	}

	// Create producer channels
	producer := func(name string, count int) <-chan string {
		ch := make(chan string)
		go func() {
			defer close(ch)
			for i := 1; i <= count; i++ {
				ch <- fmt.Sprintf("%s-%d", name, i)
				time.Sleep(30 * time.Millisecond)
			}
		}()
		return ch
	}

	// Merge outputs from multiple producers
	merged := merge(producer("A", 3), producer("B", 3))

	for msg := range merged {
		fmt.Printf("   Received: %s\n", msg)
	}
	fmt.Println()
}

// pipelineExample demonstrates chained processing stages.
// Each stage is a goroutine connected by channels.
func pipelineExample() {
	fmt.Println("5. Pipeline (chained stages)")
	fmt.Println("-----------------------------")

	// Stage 1: Generate numbers
	generator := func(nums ...int) <-chan int {
		out := make(chan int)
		go func() {
			defer close(out)
			for _, n := range nums {
				out <- n
			}
		}()
		return out
	}

	// Stage 2: Square numbers
	square := func(in <-chan int) <-chan int {
		out := make(chan int)
		go func() {
			defer close(out)
			for n := range in {
				out <- n * n
			}
		}()
		return out
	}

	// Stage 3: Double numbers
	double := func(in <-chan int) <-chan int {
		out := make(chan int)
		go func() {
			defer close(out)
			for n := range in {
				out <- n * 2
			}
		}()
		return out
	}

	// Connect the pipeline: generate -> square -> double
	pipeline := double(square(generator(1, 2, 3, 4, 5)))

	fmt.Print("   Results: ")
	for result := range pipeline {
		fmt.Printf("%d ", result)
	}
	fmt.Println("\n")
}

// selectExample demonstrates waiting on multiple channels.
// Similar to Unix select() but for channels.
func selectExample() {
	fmt.Println("6. Select (multiplexing)")
	fmt.Println("------------------------")

	ch1 := make(chan string)
	ch2 := make(chan string)

	go func() {
		time.Sleep(50 * time.Millisecond)
		ch1 <- "from channel 1"
	}()

	go func() {
		time.Sleep(30 * time.Millisecond)
		ch2 <- "from channel 2"
	}()

	// Wait for first available message
	for i := 0; i < 2; i++ {
		select {
		case msg := <-ch1:
			fmt.Printf("   Received %s\n", msg)
		case msg := <-ch2:
			fmt.Printf("   Received %s\n", msg)
		}
	}
	fmt.Println()
}

// timeoutExample demonstrates implementing timeouts.
func timeoutExample() {
	fmt.Println("7. Timeout Pattern")
	fmt.Println("------------------")

	slowOperation := make(chan string)

	go func() {
		time.Sleep(200 * time.Millisecond) // Simulate slow work
		slowOperation <- "completed"
	}()

	select {
	case result := <-slowOperation:
		fmt.Printf("   Result: %s\n", result)
	case <-time.After(100 * time.Millisecond):
		fmt.Println("   Operation timed out!")
	}
	fmt.Println()
}

// cancelExample demonstrates cancellation via done channel.
func cancelExample() {
	fmt.Println("8. Cancellation Pattern")
	fmt.Println("-----------------------")

	done := make(chan struct{})
	work := make(chan int)

	// Worker that respects cancellation
	go func() {
		defer close(work)
		for i := 1; ; i++ {
			select {
			case <-done:
				fmt.Println("   Worker cancelled")
				return
			case work <- i:
				time.Sleep(30 * time.Millisecond)
			}
		}
	}()

	// Receive some values then cancel
	for i := 0; i < 3; i++ {
		fmt.Printf("   Received: %d\n", <-work)
	}

	close(done) // Signal cancellation
	time.Sleep(50 * time.Millisecond)
	fmt.Println()
}
