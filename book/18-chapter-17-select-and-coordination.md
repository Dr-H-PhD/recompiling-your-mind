# Chapter 17: Select and Coordination

Real concurrent programs coordinate multiple channels, handle timeouts, and propagate cancellation. This chapter covers the tools for sophisticated coordination.

## Select Statements

The `select` statement waits on multiple channel operations:

```go
select {
case msg := <-ch1:
    fmt.Println("Received from ch1:", msg)
case msg := <-ch2:
    fmt.Println("Received from ch2:", msg)
case ch3 <- value:
    fmt.Println("Sent to ch3")
}
```

`select` blocks until one case can proceed, then executes that case. If multiple cases are ready, one is chosen at random.

### Non-Blocking Operations

Use `default` for non-blocking:

```go
select {
case msg := <-ch:
    process(msg)
default:
    // Channel wasn't ready, do something else
}
```

### Infinite Select Loop

```go
for {
    select {
    case msg := <-input:
        process(msg)
    case <-done:
        return
    }
}
```

This is the foundation for event loops in Go.

## Timeouts and Deadlines

PHP might use cURL timeouts:

```php
$client = new HttpClient(['timeout' => 5.0]);
```

Go uses `time.After` or context:

### Timeout with Select

```go
select {
case result := <-doWork():
    return result
case <-time.After(5 * time.Second):
    return nil, errors.New("operation timed out")
}
```

### Ticker for Periodic Work

```go
ticker := time.NewTicker(1 * time.Second)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        doPeriodicWork()
    case <-done:
        return
    }
}
```

## Context Package Deep Dive

The `context` package is Go's standard for cancellation, deadlines, and request-scoped values.

### Why Context?

Imagine an HTTP handler that:
1. Queries the database
2. Calls an external API
3. Processes results

If the client disconnects, all this work should stop. Context propagates cancellation signals through the call tree.

### Creating Contexts

```go
// Background context (root, never cancelled)
ctx := context.Background()

// With timeout
ctx, cancel := context.WithTimeout(parent, 5*time.Second)
defer cancel()  // Always call cancel to release resources

// With deadline
deadline := time.Now().Add(30 * time.Second)
ctx, cancel := context.WithDeadline(parent, deadline)
defer cancel()

// Manually cancellable
ctx, cancel := context.WithCancel(parent)
// Call cancel() when you want to cancel
```

### Using Context

```go
func fetchData(ctx context.Context) (*Data, error) {
    // Check if already cancelled
    if ctx.Err() != nil {
        return nil, ctx.Err()
    }

    // Pass context to operations
    resp, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    // ...

    rows, err := db.QueryContext(ctx, "SELECT ...")
    // ...
}
```

### Context in Select

```go
func worker(ctx context.Context, jobs <-chan Job) {
    for {
        select {
        case <-ctx.Done():
            log.Println("Worker cancelled:", ctx.Err())
            return
        case job := <-jobs:
            processJob(ctx, job)
        }
    }
}
```

### HTTP Handler Context

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()  // Cancelled if client disconnects

    result, err := fetchData(ctx)
    if err == context.Canceled {
        // Client disconnected, stop work
        return
    }
    // ...
}
```

## Cancellation Propagation

Cancellation flows down the context tree:

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())

    go worker1(ctx)
    go worker2(ctx)
    go worker3(ctx)

    time.Sleep(5 * time.Second)
    cancel()  // All workers receive cancellation
}

func worker1(ctx context.Context) {
    <-ctx.Done()  // Unblocks when cancel() called
    fmt.Println("worker1 stopping")
}
```

### Nested Contexts

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Request context (cancelled on disconnect)
    ctx := r.Context()

    // Add timeout for this specific operation
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // Cancelled if: client disconnects OR timeout
    result, err := fetchData(ctx)
}
```

## WaitGroups

`sync.WaitGroup` waits for a collection of goroutines to finish:

```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        doWork(id)
    }(i)
}

wg.Wait()  // Block until all Done() called
fmt.Println("All workers finished")
```

### WaitGroup Rules

- Call `Add` before starting the goroutine
- Call `Done` when the goroutine completes (usually `defer`)
- `Wait` blocks until counter reaches zero

### Combining WaitGroup with Context

```go
func processAll(ctx context.Context, items []Item) error {
    var wg sync.WaitGroup
    errs := make(chan error, len(items))

    for _, item := range items {
        wg.Add(1)
        item := item
        go func() {
            defer wg.Done()
            if err := process(ctx, item); err != nil {
                errs <- err
            }
        }()
    }

    // Wait in separate goroutine
    go func() {
        wg.Wait()
        close(errs)
    }()

    // Check for first error
    for err := range errs {
        if err != nil {
            return err
        }
    }
    return nil
}
```

## errgroup for Error Handling

The `golang.org/x/sync/errgroup` package combines WaitGroup with error handling:

```go
import "golang.org/x/sync/errgroup"

func fetchAll(ctx context.Context, urls []string) ([]Result, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]Result, len(urls))

    for i, url := range urls {
        i, url := i, url
        g.Go(func() error {
            result, err := fetch(ctx, url)
            if err != nil {
                return err  // Cancels context, stops other goroutines
            }
            results[i] = result
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

Key features:
- First error cancels the context
- `Wait` returns the first error
- All goroutines share the cancellable context

## Summary

- **Select** waits on multiple channels simultaneously
- **Timeouts** use `time.After` or context deadlines
- **Context** propagates cancellation and deadlines
- **WaitGroups** wait for goroutine completion
- **errgroup** combines waiting with error handling

---

## Exercises

1. **Multi-Channel Select**: Create two goroutines sending on different channels. Use select to receive from whichever is ready first.

2. **Timeout Implementation**: Write a function that returns an error if an operation takes longer than a specified duration.

3. **Context Cancellation**: Create a worker that respects context cancellation. Verify it stops when context is cancelled.

4. **Deadline Propagation**: Implement a chain of three functions, each adding context. Verify deadline propagates through all.

5. **WaitGroup Coordination**: Start N workers, wait for all to complete, then aggregate their results.

6. **errgroup Usage**: Fetch data from 5 URLs using errgroup. Handle the first error appropriately.

7. **Graceful HTTP Server**: Implement an HTTP server that gracefully shuts down on SIGINT, waiting for active requests.

8. **Heartbeat Pattern**: Implement a worker that sends periodic heartbeats on a channel while processing work.
