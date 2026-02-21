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
