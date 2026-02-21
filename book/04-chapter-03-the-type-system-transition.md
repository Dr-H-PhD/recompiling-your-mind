# Chapter 3: The Type System Transition

PHP's relationship with types has evolved dramatically. From PHP 4's complete absence of type hints, through PHP 7's scalar types, to PHP 8's union types and intersection types—the language has gradually embraced static typing while preserving dynamic flexibility.

Go, by contrast, was statically typed from day one. Every value has exactly one type, known at compile time, no exceptions.

This chapter explores how to transition your mental model from PHP's flexible typing to Go's strict typing.

## From `$anything` to Strict Types

In PHP, variables are vessels that can hold anything:

```php
$value = 42;
$value = "forty-two";
$value = ['forty', 'two'];
$value = new FortyTwo();

function process($input) {
    // $input could be anything
    // Your code must handle all possibilities
}
```

Even with modern PHP's type declarations, dynamic typing remains the default:

```php
declare(strict_types=1);

function processString(string $input): string
{
    return strtoupper($input);
}

// Without strict_types, this might work via coercion
// With strict_types, it fails at runtime
processString(42);
```

Note the key word: **runtime**. PHP discovers type errors when the code executes.

### Go's Compile-Time Certainty

In Go, every variable has exactly one type, forever:

```go
var value int = 42
value = "forty-two"  // Compile error: cannot use string as int

func process(input string) string {
    // input is always a string, guaranteed
    return strings.ToUpper(input)
}

process(42)  // Compile error: cannot use int as string
```

The Go compiler rejects invalid code before it ever runs. There's no `strict_types` to enable—strictness is the only mode.

### What You're Giving Up

PHP's dynamic typing enables powerful patterns:

```php
// Generic containers
$cache = [
    'user:1' => $userObject,
    'config' => ['debug' => true],
    'counter' => 42,
];

// Flexible function parameters
function dump(...$values): void
{
    foreach ($values as $value) {
        var_dump($value);  // Works with anything
    }
}

// Duck typing
function getLength($item): int
{
    return count($item);  // Works with arrays, Countable, etc.
}
```

Go requires explicit type definitions for each case:

```go
// Separate caches for different types
userCache := make(map[string]User)
configCache := make(map[string]map[string]bool)
counterCache := make(map[string]int)

// Or use interface{}/any (loses type safety)
cache := make(map[string]any)
cache["user:1"] = userObject
cache["config"] = map[string]bool{"debug": true}
cache["counter"] = 42
// But now you need type assertions to use values
```

This is the fundamental trade-off: flexibility versus safety.

## Type Inference: Go's Compromise

Go's designers understood that explicit typing everywhere is tedious. Their solution: **type inference** with the short declaration operator `:=`.

```go
// Explicit type
var name string = "Alice"
var age int = 30

// Inferred type (same result)
name := "Alice"  // inferred as string
age := 30        // inferred as int

// Works with complex types
users := []User{{Name: "Alice"}, {Name: "Bob"}}  // inferred as []User
config := map[string]int{"port": 8080}           // inferred as map[string]int
```

The type is still static and known at compile time—the compiler infers it from the right-hand side. This gives you PHP-like brevity with Go's compile-time safety.

### Where Inference Stops

Type inference has limits:

```go
// The compiler can't infer the type of an empty literal
var users []User        // Must specify type
users := []User{}       // Or use typed literal

// Function signatures are never inferred
func add(a int, b int) int {  // Must specify all types
    return a + b
}

// Interface variables need explicit types when empty
var reader io.Reader  // Must declare interface type
```

The pattern: Go infers types from values but requires explicit types for declarations without values.

## When You Miss `mixed` and When You Don't

PHP 8 introduced the `mixed` type to explicitly indicate "any type":

```php
function log(mixed $message): void
{
    file_put_contents('log.txt', print_r($message, true), FILE_APPEND);
}
```

Go's equivalent is `any` (alias for `interface{}`):

```go
func log(message any) {
    file, _ := os.OpenFile("log.txt", os.O_APPEND|os.O_WRONLY, 0644)
    defer file.Close()
    fmt.Fprintln(file, message)
}
```

Both work, but there's a crucial difference in how you use the value:

```php
// PHP: Use it directly
function processValue(mixed $value): string
{
    if (is_array($value)) {
        return implode(', ', $value);
    }
    return (string) $value;
}
```

```go
// Go: Must type-assert first
func processValue(value any) string {
    switch v := value.(type) {
    case []string:
        return strings.Join(v, ", ")
    case string:
        return v
    case fmt.Stringer:
        return v.String()
    default:
        return fmt.Sprint(value)
    }
}
```

In Go, `any` values are opaque until you assert their type. This is intentionally awkward—it discourages overuse of `any`.

### When You Actually Miss `mixed`

Legitimate uses of `any` in Go are rare:

1. **Serialisation**: `json.Unmarshal` into `map[string]any`
2. **Logging**: Print statements that accept anything
3. **Generic containers** (before Go 1.18 generics)

Most other uses signal design problems. If you reach for `any` often, you're probably fighting Go's type system instead of working with it.

## Generics: Go's Late Arrival vs PHP 8's Union Types

PHP 8's union and intersection types provide flexibility:

```php
function processId(int|string $id): User
{
    return $this->repo->find($id);
}

function setLogger(LoggerInterface&Countable $logger): void
{
    // $logger implements both interfaces
}
```

Go 1.18 introduced generics, which solve a different problem:

```go
// Generic function: works with any ordered type
func Min[T constraints.Ordered](a, b T) T {
    if a < b {
        return a
    }
    return b
}

// Usage
minInt := Min(3, 5)       // T inferred as int
minStr := Min("a", "b")   // T inferred as string
```

### Key Differences

**PHP union types** let a parameter accept multiple unrelated types. The function handles each type differently:

```php
function format(int|float|string $value): string
{
    if (is_string($value)) return $value;
    return number_format($value, 2);
}
```

**Go generics** constrain a type parameter to satisfy requirements, then treat all valid types uniformly:

```go
// T must be ordered (comparable with <)
func Sort[T constraints.Ordered](slice []T) {
    // Sorting logic that works identically for all ordered types
}
```

Go doesn't have union types. If you need `int | string`, you use:

1. **Separate functions**: `ProcessInt`, `ProcessString`
2. **Interface**: Define a common interface both types satisfy
3. **`any` with type switch**: As a last resort

```go
// Approach 1: Separate functions (clearest)
func ProcessInt(id int) User { ... }
func ProcessString(id string) User { ... }

// Approach 2: Interface (when behaviour is shared)
type Identifier interface {
    String() string
}

func Process(id Identifier) User { ... }
```

## Type Assertions vs PHP's `instanceof`

PHP's type checking is intuitive:

```php
if ($value instanceof User) {
    echo $value->getName();
}

if (is_string($value)) {
    echo strtoupper($value);
}
```

Go uses type assertions:

```go
// Simple assertion (panics if wrong type)
user := value.(User)
fmt.Println(user.Name)

// Safe assertion (checks first)
if user, ok := value.(User); ok {
    fmt.Println(user.Name)
}

// Type switch (for multiple possibilities)
switch v := value.(type) {
case User:
    fmt.Println(v.Name)
case string:
    fmt.Println(strings.ToUpper(v))
case int:
    fmt.Println(v * 2)
default:
    fmt.Println("unknown type")
}
```

The two-value form (`value, ok := x.(T)`) is idiomatic Go—it never panics and lets you handle the "wrong type" case gracefully.

### The Empty Interface Dance

When working with `any`/`interface{}`, you'll often need multiple assertions:

```go
func extractName(data any) string {
    // Is it a map?
    if m, ok := data.(map[string]any); ok {
        // Is the "name" key a string?
        if name, ok := m["name"].(string); ok {
            return name
        }
    }
    // Is it a struct with Name field? (can't do this directly)
    // You'd need reflection or an interface
    return ""
}
```

This verbosity is intentional—it's showing you how much type information you've lost. In Go, you're better off designing types that don't require such assertions.

## Symfony's Type-Hinted DI vs Go's Explicit Wiring

Let's compare how type systems interact with dependency injection.

### Symfony: Types as Configuration

```php
class OrderService
{
    public function __construct(
        private OrderRepository $repository,
        private MailerInterface $mailer,
    ) {}
}
```

Symfony's container uses type hints as configuration:
- `OrderRepository` is a concrete class → inject it directly
- `MailerInterface` is an interface → find a matching service

The wiring is implicit, driven by types.

### Go: Types as Constraints Only

```go
type OrderService struct {
    repository OrderRepository  // Interface
    mailer     Mailer           // Interface
}

func NewOrderService(repo OrderRepository, mailer Mailer) *OrderService {
    return &OrderService{
        repository: repo,
        mailer:     mailer,
    }
}

// Wiring is explicit
func main() {
    repo := NewSQLOrderRepository(db)
    mailer := NewSMTPMailer(config)
    service := NewOrderService(repo, mailer)
}
```

Go's types constrain what can be passed but don't configure how to find it. You write the wiring code explicitly.

This might seem like a step backward, but consider:

- **Clarity**: Every dependency is visible in `main.go`
- **Testability**: Swap dependencies by passing different implementations
- **No surprises**: No container magic to debug

## Summary

- **Static typing** catches errors at compile time, not runtime
- **Type inference** (`:=`) provides convenience without sacrificing safety
- **Generics** solve different problems than PHP's union types
- **Type assertions** replace `instanceof` but require more explicit handling
- **Explicit wiring** replaces type-driven dependency injection

---

## Exercises

1. **Type Conversion Audit**: Take PHP code that relies on type coercion (e.g., concatenating int with string). Rewrite it in Go with explicit conversions. How many hidden conversions become visible?

2. **Union Type Refactor**: Find PHP code using union types (`int|string`). Design the Go equivalent using either separate functions, interfaces, or generics. Compare the approaches.

3. **Generic Implementation**: Implement a generic `Stack[T]` in Go with `Push`, `Pop`, and `Peek` methods. Then implement the same in PHP using union types or mixed. Which is more type-safe?

4. **Type Assertion Chains**: Write Go code that parses a JSON object into `map[string]any` and extracts deeply nested values safely. Count the type assertions needed. Consider how you'd redesign with defined struct types.

5. **Interface Discovery**: Take a PHP class that implements multiple interfaces. Convert it to Go. How does implicit interface satisfaction change the design?

6. **Inference Limits**: Write Go code that uses `:=` extensively, then convert to explicit `var` declarations. Do the explicit types reveal any surprises about what types were actually inferred?

7. **Container Replacement**: Take a Symfony service with autowired dependencies. Write equivalent Go code with manual wiring. Measure lines of code versus clarity of dependency flow.

8. **Type Safety Comparison**: Create a scenario where PHP's dynamic typing would allow a bug that Go's static typing prevents. Then create the opposite—a scenario where Go's strictness creates more verbose code for an obviously safe operation.
