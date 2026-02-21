# Appendix C: Common Pitfalls

Mistakes PHP developers commonly make when learning Go.

## 1. Forgetting to Handle Errors

**Wrong:**
```go
result, _ := doSomething()  // Ignoring error!
```

**Right:**
```go
result, err := doSomething()
if err != nil {
    return nil, fmt.Errorf("doing something: %w", err)
}
```

**Why:** Go doesn't have exceptions. Ignored errors cause silent failures.

---

## 2. Nil Pointer Dereference

**Wrong:**
```go
func getName(u *User) string {
    return u.Name  // Panics if u is nil!
}
```

**Right:**
```go
func getName(u *User) string {
    if u == nil {
        return ""
    }
    return u.Name
}
```

**Why:** Unlike PHP's null-safe operator, Go panics on nil pointer access.

---

## 3. Modifying Slice While Iterating

**Wrong:**
```go
for i, v := range items {
    if shouldRemove(v) {
        items = append(items[:i], items[i+1:]...)  // Dangerous!
    }
}
```

**Right:**
```go
result := items[:0]
for _, v := range items {
    if !shouldRemove(v) {
        result = append(result, v)
    }
}
items = result
```

**Why:** Range iterates over a copy of the slice header; modifying during iteration causes skips or panics.

---

## 4. Goroutine Loop Variable Capture

**Wrong:**
```go
for _, item := range items {
    go func() {
        process(item)  // All goroutines see the same (last) item!
    }()
}
```

**Right (Go < 1.22):**
```go
for _, item := range items {
    item := item  // Shadow the variable
    go func() {
        process(item)
    }()
}
```

**Right (Go 1.22+):**
```go
for _, item := range items {
    go func() {
        process(item)  // Fixed in Go 1.22
    }()
}
```

**Why:** Before Go 1.22, the loop variable was reused; goroutines captured its address.

---

## 5. Using Defer in a Loop

**Wrong:**
```go
for _, file := range files {
    f, _ := os.Open(file)
    defer f.Close()  // All files stay open until function returns!
}
```

**Right:**
```go
for _, file := range files {
    func() {
        f, _ := os.Open(file)
        defer f.Close()
        // Process file
    }()
}
```

**Why:** Defer runs when the function returns, not when the loop iteration ends.

---

## 6. Expecting Maps to Be Ordered

**Wrong:**
```go
m := map[string]int{"a": 1, "b": 2, "c": 3}
for k, v := range m {
    fmt.Println(k, v)  // Order is random!
}
```

**Right:**
```go
keys := make([]string, 0, len(m))
for k := range m {
    keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
    fmt.Println(k, m[k])
}
```

**Why:** Go maps are unordered by design. PHP arrays maintain insertion order.

---

## 7. Returning Interface When Concrete Would Work

**Wrong:**
```go
func NewService() ServiceInterface {
    return &service{}  // Loses concrete type info
}
```

**Right:**
```go
func NewService() *Service {
    return &Service{}  // Return concrete type
}
```

**Why:** Return concrete types; accept interfaces. Callers can store in interface variables if needed.

---

## 8. Forgetting that Strings Are Immutable

**Wrong:**
```go
s := "hello"
s[0] = 'H'  // Compile error!
```

**Right:**
```go
s := "hello"
b := []byte(s)
b[0] = 'H'
s = string(b)
```

**Why:** Go strings are immutable byte sequences. Use `[]byte` or `strings.Builder` for modification.

---

## 9. Not Understanding Zero Values

**Surprise:**
```go
var s string   // "" not nil
var n int      // 0
var b bool     // false
var slice []int // nil (but usable with append!)
var m map[string]int // nil (NOT usable—must make())
```

**Right:**
```go
m := make(map[string]int)  // Initialize before use
```

**Why:** Zero values are useful but nil maps panic on write. Nil slices are safe to append.

---

## 10. Comparing Slices Directly

**Wrong:**
```go
if a == b {  // Compile error for slices!
}
```

**Right:**
```go
if slices.Equal(a, b) {  // Go 1.21+
}
// Or manual comparison
```

**Why:** Slices are reference types; use `slices.Equal` or loop comparison.

---

## 11. Modifying a Map While Reading

**Wrong (concurrent):**
```go
var m = make(map[string]int)

go func() {
    for k := range m {
        fmt.Println(k)
    }
}()

go func() {
    m["key"] = 1  // Race condition!
}()
```

**Right:**
```go
var m sync.Map
// Or: protect with mutex
```

**Why:** Go maps are not concurrency-safe. Use `sync.Map` or mutex.

---

## 12. Assuming Printf Arguments Are Evaluated Lazily

**Wrong:**
```go
slog.Debug("expensive", "data", computeExpensiveData())  // Always computed!
```

**Right:**
```go
if slog.Default().Enabled(ctx, slog.LevelDebug) {
    slog.Debug("expensive", "data", computeExpensiveData())
}
```

**Why:** Go evaluates all arguments before the function call. Unlike PHP's short-circuit evaluation.

---

## 13. Forgetting Context Cancellation

**Wrong:**
```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
// Forgot cancel()! Resources leak.
```

**Right:**
```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()  // Always call cancel
```

**Why:** Cancel releases resources associated with the context.

---

## 14. Shadowing Variables Accidentally

**Surprise:**
```go
err := doFirst()
if err != nil {
    return err
}

result, err := doSecond()  // This is the same err
if err != nil {
    return err
}

result, err := doThird()  // Compile error if err not used!
```

**Watch for:**
```go
x := 1
if true {
    x := 2  // New x! Shadows outer x
}
fmt.Println(x)  // Still 1
```

**Why:** `:=` creates new variables; watch for unintentional shadowing.

---

## 15. Expecting Short-Circuit Evaluation in Custom Types

**Wrong:**
```go
type MyBool bool

func (b MyBool) And(other MyBool) MyBool {
    return b && other  // Both sides always evaluated
}

a.And(expensiveOperation())  // Always runs!
```

**Right:**
```go
if a && expensiveOperation() {  // Built-in && short-circuits
}
```

**Why:** Only built-in `&&` and `||` short-circuit. Method calls always evaluate arguments first.

---

## 16. Using Append Without Assigning

**Wrong:**
```go
items := []int{1, 2, 3}
append(items, 4)  // Result discarded!
```

**Right:**
```go
items = append(items, 4)  // Must assign
```

**Why:** `append` may return a new slice; always assign the result.

---

## 17. Passing Structs by Value When You Want Mutation

**Wrong:**
```go
func updateUser(u User) {
    u.Name = "Updated"  // Modifies copy!
}
```

**Right:**
```go
func updateUser(u *User) {
    u.Name = "Updated"  // Modifies original
}
```

**Why:** Go passes by value. Structs are copied unless you use pointers.

---

## 18. Assuming HTTP Client Reuse

**Wrong:**
```go
func fetch(url string) {
    client := &http.Client{}  // New client each call!
    client.Get(url)
}
```

**Right:**
```go
var client = &http.Client{
    Timeout: 10 * time.Second,
}

func fetch(url string) {
    client.Get(url)  // Reuse client
}
```

**Why:** Creating clients is expensive; reuse them for connection pooling.

---

## 19. Not Closing HTTP Response Bodies

**Wrong:**
```go
resp, _ := http.Get(url)
body, _ := io.ReadAll(resp.Body)
// Body never closed! Connection leak.
```

**Right:**
```go
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
```

**Why:** Unclosed bodies prevent connection reuse and cause resource leaks.

---

## 20. Expecting JSON Numbers to Be int

**Surprise:**
```go
var data map[string]interface{}
json.Unmarshal([]byte(`{"count": 42}`), &data)
count := data["count"].(int)  // Panic! It's float64
```

**Right:**
```go
count := data["count"].(float64)
// Or use a typed struct
```

**Why:** JSON numbers unmarshal to `float64` by default in Go.

---

## 21. Slice Capacity Surprises

**Wrong:**
```go
a := []int{1, 2, 3, 4, 5}
b := a[1:3]           // b = [2, 3], shares backing array
b = append(b, 100)    // Overwrites a[3]!
fmt.Println(a)        // [1 2 3 100 5]
```

**Right:**
```go
b := append([]int{}, a[1:3]...)  // Create independent copy
// Or
b := make([]int, 2)
copy(b, a[1:3])
```

**Why:** Slices share underlying arrays until capacity forces reallocation.

---

## 22. Goroutine Leaks

**Wrong:**
```go
func fetch(url string) <-chan string {
    ch := make(chan string)
    go func() {
        resp, _ := http.Get(url)
        body, _ := io.ReadAll(resp.Body)
        ch <- string(body)  // Blocks forever if nobody receives!
    }()
    return ch
}

// If caller doesn't read: goroutine leaks
```

**Right:**
```go
func fetch(ctx context.Context, url string) <-chan string {
    ch := make(chan string, 1)  // Buffered: won't block
    go func() {
        defer close(ch)
        req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return
        }
        defer resp.Body.Close()
        body, _ := io.ReadAll(resp.Body)
        select {
        case ch <- string(body):
        case <-ctx.Done():
        }
    }()
    return ch
}
```

**Why:** Unbuffered channels block; always provide cancellation paths.

---

## 23. Nil Interface vs Nil Concrete Type

**Surprise:**
```go
func getUser() *User {
    return nil
}

func process(u interface{}) {
    if u == nil {
        fmt.Println("nil")
    } else {
        fmt.Println("not nil")  // This prints!
    }
}

process(getUser())  // Prints "not nil"!
```

**Why:** An interface holding a nil pointer is not itself nil. The interface value is `(*User, nil)`, which is not equal to `nil`.

**Right:**
```go
func process(u interface{}) {
    if u == nil || reflect.ValueOf(u).IsNil() {
        fmt.Println("nil")
    }
}
// Or return interface from function:
func getUser() interface{} {
    return nil
}
```

---

## 24. Embedding Pointer vs Value

**Subtle:**
```go
type Base struct {
    Name string
}

type Derived struct {
    Base  // Value embedding
}

type DerivedPtr struct {
    *Base  // Pointer embedding - can be nil!
}

d := DerivedPtr{}
fmt.Println(d.Name)  // Panic! Base is nil
```

**Why:** Pointer embedding can lead to nil panics; value embedding is safer.

---

## 25. Race Conditions in Tests

**Wrong:**
```go
func TestConcurrent(t *testing.T) {
    count := 0
    var wg sync.WaitGroup

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            count++  // Race condition!
        }()
    }

    wg.Wait()
    if count != 100 {
        t.Errorf("got %d, want 100", count)
    }
}
```

**Right:**
```go
func TestConcurrent(t *testing.T) {
    var count atomic.Int64
    var wg sync.WaitGroup

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            count.Add(1)  // Atomic operation
        }()
    }

    wg.Wait()
    if count.Load() != 100 {
        t.Errorf("got %d, want 100", count.Load())
    }
}
```

**Why:** Always run tests with `-race` flag to detect races.

---

# Idiomatic Go Patterns

## Writing Clean Go Code

### 1. Accept Interfaces, Return Structs

```go
// Good: Accept interface
func Process(r io.Reader) error {
    // Works with files, HTTP bodies, strings, etc.
}

// Good: Return concrete type
func NewService(db *sql.DB) *Service {
    return &Service{db: db}
}
```

### 2. Error Handling Patterns

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("processing user %d: %w", userID, err)
}

// Sentinel errors for comparison
var ErrNotFound = errors.New("not found")

if errors.Is(err, ErrNotFound) {
    // Handle not found
}

// Custom error types for data
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
```

### 3. Functional Options Pattern

```go
type Server struct {
    host    string
    port    int
    timeout time.Duration
}

type Option func(*Server)

func WithHost(host string) Option {
    return func(s *Server) { s.host = host }
}

func WithPort(port int) Option {
    return func(s *Server) { s.port = port }
}

func WithTimeout(d time.Duration) Option {
    return func(s *Server) { s.timeout = d }
}

func NewServer(opts ...Option) *Server {
    s := &Server{
        host:    "localhost",
        port:    8080,
        timeout: 30 * time.Second,
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// Usage
server := NewServer(
    WithHost("0.0.0.0"),
    WithPort(3000),
)
```

### 4. Table-Driven Tests

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positives", 1, 2, 3},
        {"negatives", -1, -2, -3},
        {"mixed", -1, 2, 1},
        {"zeros", 0, 0, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.expected {
                t.Errorf("Add(%d, %d) = %d, want %d",
                    tt.a, tt.b, got, tt.expected)
            }
        })
    }
}
```

### 5. Constructor Pattern

```go
type User struct {
    id    int64
    name  string
    email string
}

// NewUser validates and creates a User
func NewUser(name, email string) (*User, error) {
    if name == "" {
        return nil, errors.New("name is required")
    }
    if !strings.Contains(email, "@") {
        return nil, errors.New("invalid email")
    }
    return &User{
        name:  name,
        email: email,
    }, nil
}
```

## Performance Tips

### 1. Pre-allocate Slices

```go
// Slow: grows multiple times
var result []int
for i := 0; i < n; i++ {
    result = append(result, i)
}

// Fast: single allocation
result := make([]int, 0, n)
for i := 0; i < n; i++ {
    result = append(result, i)
}
```

### 2. Use strings.Builder

```go
// Slow: creates many intermediate strings
var s string
for i := 0; i < 1000; i++ {
    s += fmt.Sprintf("%d,", i)
}

// Fast: efficient concatenation
var b strings.Builder
for i := 0; i < 1000; i++ {
    fmt.Fprintf(&b, "%d,", i)
}
s := b.String()
```

### 3. Sync.Pool for Reusable Objects

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func process(data []byte) string {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    // Use buffer...
    return buf.String()
}
```

### 4. Avoid Allocations in Hot Paths

```go
// Allocation per call
func formatTime(t time.Time) string {
    return t.Format("2006-01-02 15:04:05")
}

// Reuse format constant
const timeFormat = "2006-01-02 15:04:05"

func formatTime(t time.Time) string {
    return t.Format(timeFormat)
}
```

## Code Quality Tools

```bash
# Format code
go fmt ./...

# Vet for common mistakes
go vet ./...

# Run tests with race detector
go test -race ./...

# Check test coverage
go test -cover ./...

# Comprehensive linting
golangci-lint run

# Check for vulnerabilities
govulncheck ./...
```
