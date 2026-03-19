# Appendix F: Exercise Solutions

This appendix contains solutions to selected exercises from each chapter. Try to complete the exercises on your own first before referring to these solutions.

## Part I: The Mental Shift

### Chapter 1: Why Your PHP Brain Fights Go

**Exercise 1.1: Compile-Time vs Runtime Errors**

```go
// PHP would run and fail at runtime:
// $result = "hello" + 5;

// Go catches at compile time:
// result := "hello" + 5  // compile error: mismatched types

// Solution: explicit conversion
result := "hello" + strconv.Itoa(5)  // "hello5"
```

**Exercise 1.2: Type Declaration Practice**

```go
// Declare variables with explicit types
var name string = "Alice"
var age int = 30
var balance float64 = 1234.56
var active bool = true

// Short declaration (type inference)
name := "Alice"
age := 30
balance := 1234.56
active := true
```

**Exercise 1.3: Understanding Zero Values**

```go
// All Go types have zero values
var s string    // ""
var i int       // 0
var f float64   // 0.0
var b bool      // false
var p *int      // nil
var sl []int    // nil (but len(sl) == 0)
var m map[string]int // nil

// PHP equivalent would be null/undefined
```

### Chapter 2: Philosophy Differences

**Exercise 2.1: Explicit Dependencies**

```go
// Instead of PHP's magical DI:
// $this->userService->doSomething();

// Go requires explicit passing:
type UserHandler struct {
    userService *UserService
    logger      *log.Logger
}

func NewUserHandler(us *UserService, l *log.Logger) *UserHandler {
    return &UserHandler{
        userService: us,
        logger:      l,
    }
}

func (h *UserHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // Dependencies are explicit
    h.logger.Printf("Handling request")
    h.userService.Process()
}
```

**Exercise 2.2: "A Little Copying Is Better Than a Little Dependency"**

```go
// Instead of importing a package for a simple function:

// Don't do this for simple cases:
// import "github.com/some/utils"
// utils.Min(a, b)

// Do this - copy the simple function:
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

### Chapter 3: The Type System Transition

**Exercise 3.1: Working with Interfaces**

```go
// Define a small interface
type Reader interface {
    Read(p []byte) (n int, err error)
}

// Any type with this method satisfies Reader
type FileReader struct {
    path string
}

func (fr *FileReader) Read(p []byte) (int, error) {
    // Implementation
    return len(p), nil
}

// Function accepts the interface
func process(r Reader) {
    buf := make([]byte, 1024)
    r.Read(buf)
}
```

**Exercise 3.2: Type Assertions**

```go
func handleValue(v interface{}) {
    // Type switch (preferred)
    switch val := v.(type) {
    case int:
        fmt.Printf("Integer: %d\n", val)
    case string:
        fmt.Printf("String: %s\n", val)
    case []byte:
        fmt.Printf("Bytes: %v\n", val)
    default:
        fmt.Printf("Unknown type: %T\n", val)
    }

    // Type assertion with check
    if s, ok := v.(string); ok {
        fmt.Printf("It's a string: %s\n", s)
    }
}
```

### Chapter 4: Error Handling

**Exercise 4.1: Error Wrapping**

```go
import (
    "errors"
    "fmt"
)

func fetchUser(id int) (*User, error) {
    user, err := db.Query(id)
    if err != nil {
        // Wrap with context
        return nil, fmt.Errorf("fetchUser(%d): %w", id, err)
    }
    return user, nil
}

func handleRequest(id int) error {
    user, err := fetchUser(id)
    if err != nil {
        // Check for specific error
        if errors.Is(err, sql.ErrNoRows) {
            return fmt.Errorf("user not found: %w", err)
        }
        return fmt.Errorf("handleRequest: %w", err)
    }
    // Use user...
    return nil
}
```

**Exercise 4.2: Custom Error Types**

```go
// Define custom error type
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// Usage
func validateUser(u *User) error {
    if u.Name == "" {
        return &ValidationError{
            Field:   "name",
            Message: "cannot be empty",
        }
    }
    return nil
}

// Checking error type
func handleError(err error) {
    var valErr *ValidationError
    if errors.As(err, &valErr) {
        fmt.Printf("Field %s is invalid\n", valErr.Field)
    }
}
```

## Part II: Structural Rewiring

### Chapter 5: From Classes to Structs

**Exercise 5.1: Constructor Pattern**

```go
// PHP: __construct()
// Go: New* factory function

type User struct {
    id        int64
    name      string
    email     string
    createdAt time.Time
}

// Constructor function
func NewUser(name, email string) *User {
    return &User{
        name:      name,
        email:     email,
        createdAt: time.Now(),
    }
}

// Constructor with validation
func NewUserWithValidation(name, email string) (*User, error) {
    if name == "" {
        return nil, errors.New("name is required")
    }
    if !strings.Contains(email, "@") {
        return nil, errors.New("invalid email")
    }
    return NewUser(name, email), nil
}
```

**Exercise 5.2: Value vs Pointer Receivers**

```go
type Counter struct {
    value int
}

// Value receiver - doesn't modify original
func (c Counter) Value() int {
    return c.value
}

// Pointer receiver - modifies original
func (c *Counter) Increment() {
    c.value++
}

// Usage
func main() {
    c := Counter{value: 0}
    fmt.Println(c.Value()) // 0
    c.Increment()
    fmt.Println(c.Value()) // 1
}
```

### Chapter 6: Composition Over Inheritance

**Exercise 6.1: Embedding for Reuse**

```go
// PHP would use inheritance:
// class AdminUser extends User { ... }

// Go uses embedding:
type User struct {
    ID    int64
    Name  string
    Email string
}

func (u *User) String() string {
    return fmt.Sprintf("%s <%s>", u.Name, u.Email)
}

type AdminUser struct {
    User  // Embedded - AdminUser "has" User
    Roles []string
}

func (a *AdminUser) HasRole(role string) bool {
    for _, r := range a.Roles {
        if r == role {
            return true
        }
    }
    return false
}

// AdminUser can use User's methods
func main() {
    admin := AdminUser{
        User:  User{ID: 1, Name: "Alice", Email: "alice@example.com"},
        Roles: []string{"admin", "moderator"},
    }

    // Promoted method from User
    fmt.Println(admin.String()) // Alice <alice@example.com>

    // AdminUser's own method
    fmt.Println(admin.HasRole("admin")) // true
}
```

### Chapter 7: Interfaces

**Exercise 7.1: Small Interface Design**

```go
// Bad: Large interface
type UserManager interface {
    Create(u *User) error
    Update(u *User) error
    Delete(id int64) error
    Find(id int64) (*User, error)
    FindAll() ([]*User, error)
    FindByEmail(email string) (*User, error)
}

// Good: Small, focused interfaces
type UserCreator interface {
    Create(u *User) error
}

type UserFinder interface {
    Find(id int64) (*User, error)
}

type UserUpdater interface {
    Update(u *User) error
}

// Compose when needed
type UserStore interface {
    UserCreator
    UserFinder
    UserUpdater
}
```

**Exercise 7.2: Accept Interfaces, Return Structs**

```go
// Accept interface
func ProcessData(r io.Reader) error {
    data, err := io.ReadAll(r)
    if err != nil {
        return err
    }
    // Process data...
    return nil
}

// Return concrete type
func NewFileReader(path string) *os.File {
    f, _ := os.Open(path)
    return f
}

// Usage - any Reader works
func main() {
    // File
    f, _ := os.Open("data.txt")
    ProcessData(f)

    // String
    ProcessData(strings.NewReader("hello"))

    // Network
    resp, _ := http.Get("https://example.com")
    ProcessData(resp.Body)
}
```

## Part III: Practical Patterns

### Chapter 10: Web Development

**Exercise 10.1: Middleware Chain**

```go
// Middleware type
type Middleware func(http.Handler) http.Handler

// Chain middleware
func Chain(h http.Handler, middleware ...Middleware) http.Handler {
    for i := len(middleware) - 1; i >= 0; i-- {
        h = middleware[i](h)
    }
    return h
}

// Example middleware
func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
    })
}

func Auth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Usage
handler := Chain(myHandler, Logger, Auth)
```

### Chapter 11: Database Access

**Exercise 11.1: Transaction Pattern**

```go
func WithTransaction(db *sql.DB, fn func(*sql.Tx) error) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)
        }
    }()

    if err := fn(tx); err != nil {
        tx.Rollback()
        return err
    }

    return tx.Commit()
}

// Usage
err := WithTransaction(db, func(tx *sql.Tx) error {
    _, err := tx.Exec("INSERT INTO users (name) VALUES (?)", "Alice")
    if err != nil {
        return err
    }
    _, err = tx.Exec("INSERT INTO audit_log (action) VALUES (?)", "created user")
    return err
})
```

## Part IV: Concurrency

### Chapter 15: Introducing Concurrency

**Exercise 15.1: Basic Goroutines**

```go
func main() {
    // Launch goroutine
    go func() {
        fmt.Println("Hello from goroutine")
    }()

    // Main continues immediately
    fmt.Println("Hello from main")

    // Wait for goroutine (simple but not recommended)
    time.Sleep(100 * time.Millisecond)
}

// Better with WaitGroup
func main() {
    var wg sync.WaitGroup

    wg.Add(1)
    go func() {
        defer wg.Done()
        fmt.Println("Hello from goroutine")
    }()

    wg.Wait()
    fmt.Println("All goroutines done")
}
```

### Chapter 16: Channels

**Exercise 16.1: Producer-Consumer**

```go
func producer(ch chan<- int) {
    for i := 0; i < 10; i++ {
        ch <- i
    }
    close(ch)
}

func consumer(ch <-chan int, done chan<- bool) {
    for v := range ch {
        fmt.Printf("Received: %d\n", v)
    }
    done <- true
}

func main() {
    ch := make(chan int, 5)
    done := make(chan bool)

    go producer(ch)
    go consumer(ch, done)

    <-done
}
```

### Chapter 17: Select and Coordination

**Exercise 17.1: Timeout with Context**

```go
func fetchWithTimeout(url string, timeout time.Duration) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    return io.ReadAll(resp.Body)
}
```

### Chapter 18: Concurrency Patterns

**Exercise 18.1: Worker Pool**

```go
func worker(id int, jobs <-chan int, results chan<- int) {
    for j := range jobs {
        fmt.Printf("Worker %d processing job %d\n", id, j)
        time.Sleep(time.Second)
        results <- j * 2
    }
}

func main() {
    jobs := make(chan int, 100)
    results := make(chan int, 100)

    // Start 3 workers
    for w := 1; w <= 3; w++ {
        go worker(w, jobs, results)
    }

    // Send 9 jobs
    for j := 1; j <= 9; j++ {
        jobs <- j
    }
    close(jobs)

    // Collect results
    for a := 1; a <= 9; a++ {
        <-results
    }
}
```

## Part V: Advanced Topics

### Chapter 20: Reflection and Code Generation

**Exercise 20.1: Simple Reflection**

```go
import "reflect"

func inspectStruct(v interface{}) {
    t := reflect.TypeOf(v)
    val := reflect.ValueOf(v)

    if t.Kind() == reflect.Ptr {
        t = t.Elem()
        val = val.Elem()
    }

    fmt.Printf("Type: %s\n", t.Name())

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        value := val.Field(i)
        fmt.Printf("  %s: %v (%s)\n", field.Name, value, field.Type)
    }
}
```

### Chapter 21: Performance

**Exercise 21.1: Benchmarking**

```go
// In *_test.go file:
func BenchmarkStringConcat(b *testing.B) {
    for i := 0; i < b.N; i++ {
        s := ""
        for j := 0; j < 100; j++ {
            s += "a"
        }
    }
}

func BenchmarkStringBuilder(b *testing.B) {
    for i := 0; i < b.N; i++ {
        var sb strings.Builder
        for j := 0; j < 100; j++ {
            sb.WriteString("a")
        }
        _ = sb.String()
    }
}

// Run: go test -bench=. -benchmem
```

## Part VI: Deployment and Migration

### Chapter 23: Building and Deploying

**Exercise 23.1: Multi-Stage Dockerfile**

```dockerfile
# Stage 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
RUN adduser -D appuser
USER appuser
EXPOSE 8080
CMD ["./server"]
```

### Chapter 25: Migration Strategies

**Exercise 25.1: Feature Flag Pattern**

```go
type FeatureFlags struct {
    flags map[string]bool
    mu    sync.RWMutex
}

func (f *FeatureFlags) IsEnabled(name string) bool {
    f.mu.RLock()
    defer f.mu.RUnlock()
    return f.flags[name]
}

func (f *FeatureFlags) Enable(name string) {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.flags[name] = true
}

// Usage
var features = &FeatureFlags{flags: make(map[string]bool)}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    if features.IsEnabled("new_algorithm") {
        // New Go implementation
        newAlgorithm(w, r)
    } else {
        // Proxy to PHP
        proxyToPHP(w, r)
    }
}
```

---

*Note: These are selected solutions. For complete solutions and additional exercises, see the companion code repository.*
