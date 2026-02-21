# Chapter 19: When Concurrency Goes Wrong

Concurrency introduces failure modes that don't exist in sequential PHP code. Race conditions, deadlocks, and goroutine leaks are new territory for PHP developers.

## Race Conditions (New Territory for PHP Developers)

A race condition occurs when multiple goroutines access shared data, and at least one modifies it:

```go
// RACE CONDITION!
var counter int

func increment() {
    counter++  // Read-modify-write is not atomic
}

func main() {
    for i := 0; i < 1000; i++ {
        go increment()
    }
    time.Sleep(time.Second)
    fmt.Println(counter)  // Not 1000!
}
```

`counter++` is three operations:
1. Read current value
2. Add one
3. Write new value

Two goroutines can:
1. Both read 5
2. Both write 6
3. One increment is lost

### Why PHP Developers Don't See This

PHP's shared-nothing model means each request has its own memory:

```php
// Each request has its own $counter
$counter = 0;
$counter++;  // No race possible
```

Multiple PHP requests might race on database state, but not in-memory state.

### Fixing Race Conditions

**Option 1: Mutex**

```go
var (
    counter int
    mu      sync.Mutex
)

func increment() {
    mu.Lock()
    counter++
    mu.Unlock()
}
```

**Option 2: Atomic operations**

```go
var counter int64

func increment() {
    atomic.AddInt64(&counter, 1)
}
```

**Option 3: Channels (share by communicating)**

```go
func counter(increments <-chan struct{}) <-chan int {
    result := make(chan int)
    go func() {
        count := 0
        for range increments {
            count++
        }
        result <- count
        close(result)
    }()
    return result
}
```

## The Race Detector

Go has a built-in race detector:

```bash
go run -race main.go
go test -race ./...
```

Output for the counter example:

```
WARNING: DATA RACE
Read at 0x00c00001c0b8 by goroutine 7:
  main.increment()
      main.go:10 +0x3a

Previous write at 0x00c00001c0b8 by goroutine 6:
  main.increment()
      main.go:10 +0x50
```

### Using the Race Detector

- Run tests with `-race` in CI
- Test under realistic concurrency (race detector needs actual concurrent access)
- Races are non-deterministic—run tests multiple times
- The race detector slows execution 2-10x; don't use in production

### Common Race Patterns

**Map access:**
```go
// RACE!
var cache = make(map[string]string)

go func() { cache["a"] = "1" }()
go func() { _ = cache["b"] }()
```

Fix with `sync.Map` or mutex:
```go
var cache sync.Map
cache.Store("a", "1")
val, _ := cache.Load("a")
```

**Slice append:**
```go
// RACE!
var results []int

go func() { results = append(results, 1) }()
go func() { results = append(results, 2) }()
```

**Struct field access:**
```go
// RACE if fields accessed concurrently
type Stats struct {
    Count int
    Total int
}
```

## Deadlocks

A deadlock occurs when goroutines wait for each other forever:

```go
// DEADLOCK!
func main() {
    ch := make(chan int)  // Unbuffered
    ch <- 1  // Blocks forever—no receiver!
    fmt.Println(<-ch)
}
```

### Classic Deadlock: Two Mutexes

```go
var mu1, mu2 sync.Mutex

// Goroutine 1
go func() {
    mu1.Lock()
    time.Sleep(time.Millisecond)
    mu2.Lock()  // Waits for goroutine 2
    // ...
}()

// Goroutine 2
go func() {
    mu2.Lock()
    time.Sleep(time.Millisecond)
    mu1.Lock()  // Waits for goroutine 1
    // ...
}()

// DEADLOCK! Both wait forever
```

### Prevention Strategies

**Consistent lock ordering:**
```go
// Always lock mu1 before mu2
func operation() {
    mu1.Lock()
    mu2.Lock()
    // ...
    mu2.Unlock()
    mu1.Unlock()
}
```

**Use timeouts:**
```go
select {
case ch <- value:
    // Sent successfully
case <-time.After(5 * time.Second):
    // Timed out, handle gracefully
}
```

**Use context with deadline:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

select {
case result := <-doWork(ctx):
    return result, nil
case <-ctx.Done():
    return nil, ctx.Err()
}
```

## Channel Leaks

A channel leak occurs when a goroutine is blocked on a channel forever:

```go
// LEAK!
func process(items []int) <-chan int {
    results := make(chan int)

    go func() {
        for _, item := range items {
            if item < 0 {
                return  // Exits without closing channel
            }
            results <- item * 2
        }
        close(results)
    }()

    return results
}

// Caller waits forever if early return
for result := range process([]int{1, 2, -1, 4}) {
    fmt.Println(result)
}
```

### Preventing Leaks

**Always close channels:**
```go
go func() {
    defer close(results)  // Always closes
    for _, item := range items {
        if item < 0 {
            return
        }
        results <- item * 2
    }
}()
```

**Use context for cancellation:**
```go
func process(ctx context.Context, items []int) <-chan int {
    results := make(chan int)

    go func() {
        defer close(results)
        for _, item := range items {
            select {
            case <-ctx.Done():
                return
            case results <- item * 2:
            }
        }
    }()

    return results
}
```

## Debugging Concurrent Code

### Goroutine Dumps

```go
import "runtime/pprof"

// Dump all goroutine stacks
pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
```

Or via HTTP:
```go
import _ "net/http/pprof"

go func() {
    http.ListenAndServe(":6060", nil)
}()
```

Then: `curl http://localhost:6060/debug/pprof/goroutine?debug=1`

### Counting Goroutines

```go
fmt.Println("Goroutines:", runtime.NumGoroutine())
```

Growing goroutine count suggests leaks.

### Logging with Goroutine ID

```go
func getGoroutineID() uint64 {
    b := make([]byte, 64)
    runtime.Stack(b, false)
    var id uint64
    fmt.Sscanf(string(b), "goroutine %d", &id)
    return id
}

log.Printf("[goroutine %d] processing item", getGoroutineID())
```

## Common Mistakes from PHP Developers

### 1. Forgetting Goroutines Outlive Function Calls

```go
func handler(w http.ResponseWriter, r *http.Request) {
    go sendEmail(email)  // Goroutine continues after handler returns
    w.Write([]byte("OK"))
}

// If server shuts down, email might not send
```

### 2. Closing Channels from Wrong Side

```go
// WRONG: Receiver closing sender's channel
go func() {
    for val := range ch {
        if val < 0 {
            close(ch)  // Sender will panic!
        }
    }
}()
```

### 3. Assuming Channel Order

```go
ch := make(chan int, 10)
for i := 0; i < 10; i++ {
    go func(n int) {
        ch <- n
    }(i)
}

// Order is NOT guaranteed!
for i := 0; i < 10; i++ {
    fmt.Println(<-ch)  // Could be any order
}
```

### 4. Not Waiting for Goroutines

```go
func main() {
    go doWork()
    // Program exits before goroutine completes!
}
```

## Summary

- **Race conditions** occur when goroutines share mutable data
- **The race detector** (`-race`) finds races automatically
- **Deadlocks** happen when goroutines wait for each other
- **Channel leaks** leave goroutines blocked forever
- **Common mistakes** include assuming order and not waiting

---

## Exercises

1. **Create a Race**: Write code with an intentional race condition. Verify the race detector finds it.

2. **Fix the Race**: Fix the race using mutex, atomic, and channel approaches. Benchmark each.

3. **Deadlock Scenario**: Create a deadlock with two mutexes. Then fix it with consistent ordering.

4. **Channel Leak**: Create a goroutine leak. Use `runtime.NumGoroutine()` to detect it.

5. **Race Detector CI**: Add race detection to a test suite. Simulate running in CI.

6. **Debug with pprof**: Set up pprof HTTP endpoint. Analyse goroutine dumps.

7. **Timeout Prevention**: Take blocking code and add timeouts to prevent hangs.

8. **Leak Prevention Pattern**: Implement a worker pattern that guarantees no goroutine leaks even on errors.
