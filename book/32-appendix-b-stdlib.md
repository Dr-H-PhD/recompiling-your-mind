# Appendix B: Standard Library Essentials

Key Go standard library packages with Symfony component comparisons.

## net/http (HttpFoundation + HttpKernel)

```go
import "net/http"

// Server
http.HandleFunc("/", handler)
http.ListenAndServe(":8080", nil)

// Custom handler
type MyHandler struct{}
func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

// Request
r.Method           // GET, POST, etc.
r.URL.Path         // /users/123
r.URL.Query()      // Query parameters
r.Header           // Headers
r.Body             // Request body (io.ReadCloser)
r.Context()        // Request context
r.FormValue("key") // Form value

// Response
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write([]byte("body"))

// Client
client := &http.Client{Timeout: 10 * time.Second}
resp, err := client.Get("https://example.com")
req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
```

## encoding/json (Serializer)

```go
import "encoding/json"

// Struct tags
type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email,omitempty"`
    Password  string    `json:"-"`
    CreatedAt time.Time `json:"created_at"`
}

// Marshal (encode)
data, err := json.Marshal(user)
json.NewEncoder(w).Encode(user)

// Unmarshal (decode)
var user User
err := json.Unmarshal(data, &user)
err := json.NewDecoder(r.Body).Decode(&user)

// Raw JSON
var raw json.RawMessage
var generic map[string]interface{}
```

## database/sql (Doctrine DBAL)

```go
import (
    "database/sql"
    _ "github.com/lib/pq"
)

// Connect
db, err := sql.Open("postgres", dsn)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)

// Query multiple rows
rows, err := db.QueryContext(ctx, "SELECT id, name FROM users WHERE active = $1", true)
defer rows.Close()
for rows.Next() {
    var id int
    var name string
    rows.Scan(&id, &name)
}

// Query single row
var name string
err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = $1", id).Scan(&name)
if err == sql.ErrNoRows { /* not found */ }

// Execute
result, err := db.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", name)
id, _ := result.LastInsertId()
affected, _ := result.RowsAffected()

// Transactions
tx, err := db.BeginTx(ctx, nil)
defer tx.Rollback()
tx.ExecContext(ctx, "...")
tx.Commit()

// Prepared statements
stmt, err := db.PrepareContext(ctx, "SELECT * FROM users WHERE id = $1")
defer stmt.Close()
stmt.QueryRowContext(ctx, id)
```

## html/template (Twig)

```go
import "html/template"

// Parse template
t := template.Must(template.New("page").Parse(`
<!DOCTYPE html>
<html>
<body>
    <h1>{{.Title}}</h1>
    {{range .Items}}
        <p>{{.}}</p>
    {{end}}
    {{if .ShowFooter}}
        <footer>Footer</footer>
    {{end}}
</body>
</html>
`))

// Execute
t.Execute(w, map[string]interface{}{
    "Title":      "My Page",
    "Items":      []string{"a", "b", "c"},
    "ShowFooter": true,
})

// Custom functions
funcs := template.FuncMap{
    "upper": strings.ToUpper,
    "formatDate": func(t time.Time) string {
        return t.Format("2006-01-02")
    },
}
t := template.New("page").Funcs(funcs).Parse("{{.Name | upper}}")

// Parse files
t := template.Must(template.ParseFiles("base.html", "page.html"))
```

## log/slog (Monolog)

```go
import "log/slog"

// Setup
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
slog.SetDefault(logger)

// Logging
slog.Debug("Debug message", "key", "value")
slog.Info("Info message", "user_id", 123)
slog.Warn("Warning", "err", err)
slog.Error("Error occurred", "error", err)

// With context
logger := slog.With("request_id", requestID)
logger.Info("Processing")

// Groups
slog.Info("Request",
    slog.Group("request",
        slog.String("method", r.Method),
        slog.String("path", r.URL.Path),
    ),
)

// Levels
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})
```

## context (Request-scoped data)

```go
import "context"

// Create contexts
ctx := context.Background()
ctx := context.TODO()

// With timeout
ctx, cancel := context.WithTimeout(parent, 5*time.Second)
defer cancel()

// With deadline
ctx, cancel := context.WithDeadline(parent, time.Now().Add(30*time.Second))

// With cancellation
ctx, cancel := context.WithCancel(parent)
cancel() // Cancel when done

// With value
ctx := context.WithValue(parent, "user_id", 123)
userID := ctx.Value("user_id").(int)

// Check cancellation
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue
}

// In functions
func doWork(ctx context.Context) error {
    if ctx.Err() != nil {
        return ctx.Err()
    }
    // Work...
}
```

## time (DateTime)

```go
import "time"

// Current time
now := time.Now()
now.UTC()

// Create time
t := time.Date(2024, time.January, 15, 10, 30, 0, 0, time.UTC)

// Parse
t, err := time.Parse("2006-01-02", "2024-01-15")
t, err := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")

// Format
s := t.Format("2006-01-02 15:04:05")
s := t.Format(time.RFC3339)

// Duration
d := 5 * time.Second
d := time.Hour
d := time.Since(start)
d := time.Until(deadline)

// Arithmetic
tomorrow := now.Add(24 * time.Hour)
yesterday := now.Add(-24 * time.Hour)
diff := t2.Sub(t1)

// Sleep
time.Sleep(time.Second)

// Ticker
ticker := time.NewTicker(time.Second)
for t := range ticker.C {
    // Every second
}
ticker.Stop()

// Timer
timer := time.NewTimer(5 * time.Second)
<-timer.C // After 5 seconds
```

## sync (Concurrency primitives)

```go
import "sync"

// Mutex
var mu sync.Mutex
mu.Lock()
defer mu.Unlock()

// RWMutex (read-write)
var rw sync.RWMutex
rw.RLock()  // Multiple readers OK
rw.RUnlock()
rw.Lock()   // Exclusive write
rw.Unlock()

// WaitGroup
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    // Work
}()
wg.Wait()

// Once (singleton)
var once sync.Once
once.Do(func() {
    // Runs exactly once
})

// Pool (object reuse)
var pool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}
buf := pool.Get().(*bytes.Buffer)
defer pool.Put(buf)

// Map (concurrent-safe)
var m sync.Map
m.Store("key", "value")
v, ok := m.Load("key")
m.Delete("key")
m.Range(func(k, v interface{}) bool {
    return true // Continue iteration
})
```

## os (Environment, Files)

```go
import "os"

// Environment
val := os.Getenv("KEY")
os.Setenv("KEY", "value")
os.Unsetenv("KEY")
os.Environ() // All env vars

// Files
f, err := os.Open("file.txt")       // Read
f, err := os.Create("file.txt")     // Write (create/truncate)
f, err := os.OpenFile("file.txt", os.O_APPEND|os.O_WRONLY, 0644)
defer f.Close()

data, err := os.ReadFile("file.txt")
err := os.WriteFile("file.txt", data, 0644)

// Directories
err := os.Mkdir("dir", 0755)
err := os.MkdirAll("path/to/dir", 0755)
err := os.Remove("file.txt")
err := os.RemoveAll("dir")
entries, err := os.ReadDir("dir")

// Process
os.Exit(1)
os.Getpid()
os.Args // Command line arguments
```

## io (Readers/Writers)

```go
import "io"

// Read
data, err := io.ReadAll(r)
n, err := io.Copy(dst, src)
n, err := io.CopyN(dst, src, 100)

// Limited read
lr := io.LimitReader(r, 1024)

// Multi-reader
r := io.MultiReader(r1, r2, r3)

// Multi-writer
w := io.MultiWriter(w1, w2)

// Pipe
pr, pw := io.Pipe()
go func() {
    pw.Write(data)
    pw.Close()
}()
io.ReadAll(pr)

// NopCloser (add Close() to Reader)
rc := io.NopCloser(r)
```

## fmt (Formatting)

```go
import "fmt"

// Print
fmt.Print("no newline")
fmt.Println("with newline")
fmt.Printf("formatted: %s %d\n", s, n)

// Sprint (return string)
s := fmt.Sprint(value)
s := fmt.Sprintf("format: %v", value)

// Fprint (write to io.Writer)
fmt.Fprint(w, "to writer")
fmt.Fprintf(w, "format: %v", value)

// Scan (read input)
var s string
var n int
fmt.Scan(&s, &n)
fmt.Scanf("%s %d", &s, &n)

// Format verbs
%v   // Default format
%+v  // With field names (structs)
%#v  // Go syntax
%T   // Type
%s   // String
%d   // Integer
%f   // Float
%t   // Boolean
%p   // Pointer
%w   // Error wrapping (Errorf only)
```
