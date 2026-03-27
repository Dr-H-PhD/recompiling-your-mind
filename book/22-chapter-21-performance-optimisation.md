# Chapter 21: Performance Optimisation

PHP performance tuning involves OPcache, database queries, and caching layers. Go performance tuning is more granular—memory allocations, escape analysis, and CPU profiling become important.

## Profiling: pprof (CPU, Memory, Goroutine)

Go has built-in profiling via `pprof`:

```go
import (
    "net/http"
    _ "net/http/pprof"
)

func main() {
    go func() {
        http.ListenAndServe(":6060", nil)
    }()
    // Application code
}
```

### CPU Profiling

```bash
# 30-second CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Interactive commands
(pprof) top10        # Top functions by CPU
(pprof) list funcName  # Source with CPU annotations
(pprof) web          # Visualise in browser
```

### Memory Profiling

```bash
go tool pprof http://localhost:6060/debug/pprof/heap

(pprof) top          # Top memory allocators
(pprof) list funcName  # Source with allocation annotations
```

### Goroutine Profiling

```bash
go tool pprof http://localhost:6060/debug/pprof/goroutine

(pprof) top          # Where goroutines are stuck
```

### Command-Line Profiling

```go
import "runtime/pprof"

func main() {
    f, _ := os.Create("cpu.prof")
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()

    // Run your code

    // Memory profile
    f2, _ := os.Create("mem.prof")
    pprof.WriteHeapProfile(f2)
}
```

Then analyse:
```bash
go tool pprof cpu.prof
```

## Benchmarking Methodology

Go benchmarks are built-in:

```go
func BenchmarkProcess(b *testing.B) {
    input := generateInput()
    b.ResetTimer()  // Don't count setup

    for i := 0; i < b.N; i++ {
        Process(input)
    }
}
```

Run:
```bash
go test -bench=. -benchmem

# Output:
# BenchmarkProcess-8   1000000   1234 ns/op   256 B/op   2 allocs/op
```

### Comparing Benchmarks

```bash
# Save baseline
go test -bench=. -count=5 > old.txt

# Make changes, re-run
go test -bench=. -count=5 > new.txt

# Compare
benchstat old.txt new.txt
```

### Avoiding Benchmark Pitfalls

```go
// BAD: Compiler might optimise away
func BenchmarkBad(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _ = compute()  // Result unused, might be eliminated
    }
}

// GOOD: Use result
var result int

func BenchmarkGood(b *testing.B) {
    var r int
    for i := 0; i < b.N; i++ {
        r = compute()
    }
    result = r  // Prevent elimination
}
```

## Memory Allocation Patterns

### Allocation Costs

Each allocation has overhead:
- Memory allocation
- Garbage collection tracking
- Potential GC pause contribution

### Reducing Allocations

**Pre-allocate slices:**
```go
// BAD: Multiple allocations as slice grows
var items []Item
for _, v := range data {
    items = append(items, transform(v))
}

// GOOD: Single allocation
items := make([]Item, 0, len(data))
for _, v := range data {
    items = append(items, transform(v))
}
```

**Reuse buffers:**
```go
// BAD: New buffer each call
func process(data []byte) []byte {
    buf := new(bytes.Buffer)
    buf.Write(data)
    return buf.Bytes()
}

// GOOD: Reuse buffer
func (p *Processor) process(data []byte) []byte {
    p.buf.Reset()
    p.buf.Write(data)
    return p.buf.Bytes()
}
```

**Use `strings.Builder`:**
```go
// BAD: String concatenation allocates
s := ""
for _, part := range parts {
    s += part
}

// GOOD: Builder
var b strings.Builder
for _, part := range parts {
    b.WriteString(part)
}
s := b.String()
```

## Escape Analysis Awareness

Go's compiler decides whether variables live on stack (fast) or heap (slower, GC-tracked).

### Viewing Escape Analysis

```bash
go build -gcflags="-m" .

# Output:
# ./main.go:10:6: moved to heap: x
# ./main.go:15:9: &User{} escapes to heap
```

### Common Escape Causes

**Returning pointers:**
```go
// Escapes to heap
func newUser() *User {
    u := User{Name: "Alice"}
    return &u  // Address taken, escapes
}

// Stays on stack
func newUser() User {
    u := User{Name: "Alice"}
    return u  // Value copy, no escape
}
```

**Interface conversions:**
```go
func process(v any) { }

func main() {
    x := 42
    process(x)  // x escapes (boxed in interface)
}
```

**Closures capturing variables:**
```go
func createCounter() func() int {
    count := 0
    return func() int {
        count++  // count escapes (captured by closure)
        return count
    }
}
```

### When to Care

For hot paths (millions of calls), reducing allocations matters. For most code, clarity beats micro-optimisation.

> **Programmer:** PHP performance tuning revolves around OPcache, preloading, and reducing database queries -- the language itself is interpreted, so there is always a VM layer between your code and the CPU. Go compiles directly to native machine code with no VM overhead, which means performance bottlenecks shift to memory allocations and CPU cache behaviour rather than interpreter overhead. The `go test -bench=. -benchmem` command is your primary weapon: it reports nanoseconds per operation, bytes allocated, and number of allocations, giving you concrete data to optimise against. Use `-gcflags='-m'` to see escape analysis decisions -- variables that escape to the heap require garbage collection, whilst stack-allocated variables are freed automatically when the function returns. In production Go services handling thousands of requests per second, reducing a hot-path function from two allocations to zero can measurably improve P99 latency.

## Pool Patterns for Allocation Reduction

`sync.Pool` provides object reuse:

```go
var bufferPool = sync.Pool{
    New: func() any {
        return new(bytes.Buffer)
    },
}

func process(data []byte) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    buf.Write(data)
    // Use buffer
}
```

### Pool Caveats

- Pool may be cleared between GC cycles
- Not for long-lived objects
- Best for frequently allocated short-lived objects
- Thread-safe

### Common Pool Use Cases

- Byte buffers
- Temporary slices
- JSON encoder/decoder buffers
- HTTP request/response objects

## Comparing to Blackfire/Xdebug Profiling

PHP profiling:
```php
// Xdebug profiler output: cachegrind files
// Blackfire: Timeline visualisation
```

Go profiling is similar in concept but:
- Built into the language (no extensions)
- Works in production (low overhead)
- Includes memory and goroutine profiling
- `pprof` output is analysable offline

### Go Profiling Workflow

1. **Identify**: Which endpoint or function is slow?
2. **Profile**: Run pprof on that code path
3. **Analyse**: Find hot spots (CPU) or allocation sources (memory)
4. **Optimise**: Fix the bottleneck
5. **Benchmark**: Verify improvement
6. **Repeat**: Profile again

> **Programmer:** Unlike Xdebug and Blackfire, which require PHP extensions and impose significant overhead, Go's `pprof` is built into the standard library and designed for production use. Import `net/http/pprof` with a blank identifier and you get live CPU, memory, and goroutine profiling on a debug HTTP endpoint with minimal performance impact. The `benchstat` tool compares benchmark runs statistically, telling you whether a change is a genuine improvement or noise -- this replaces the guesswork of "run it a few times and eyeball the results." Use `sync.Pool` for objects that are allocated and discarded frequently (byte buffers, temporary slices, encoder states): it recycles them across goroutines, dramatically reducing GC pressure. In production, always profile before optimising -- the Go compiler's inlining, escape analysis, and register allocation often make intuitive "optimisations" counterproductive.

## Summary

- **pprof** profiles CPU, memory, and goroutines
- **Benchmarks** measure and compare performance
- **Allocation patterns** affect GC and speed
- **Escape analysis** determines stack vs heap
- **sync.Pool** reuses frequently allocated objects
- **Profile before optimising**—intuition is often wrong

---

## Exercises

1. **CPU Profile**: Profile an application. Find the hottest function. Optimise it.

2. **Memory Profile**: Profile memory. Find the biggest allocator. Reduce allocations.

3. **Benchmark Comparison**: Write a function two ways. Benchmark both. Use benchstat to compare.

4. **Escape Analysis**: Write code that escapes to heap. Modify it to stay on stack. Verify with `-gcflags="-m"`.

5. **sync.Pool**: Add pooling to a frequently allocated object. Benchmark before and after.

6. **Allocation Audit**: Use `-benchmem` to find allocations in a function. Reduce to zero allocations.

7. **Goroutine Leak Detection**: Profile goroutines. Find where they're blocked. Fix the leak.

8. **Production Profiling**: Set up pprof in an HTTP server. Profile under load.
