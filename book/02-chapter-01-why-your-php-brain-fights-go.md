# Chapter 1: Why Your PHP Brain Fights Go

You've spent years—perhaps decades—mastering PHP. You know its quirks, its strengths, its idioms. You can look at a codebase and immediately sense what's wrong. You've internalised patterns so deeply that they feel like instinct.

Now you're learning Go, and something strange is happening: your expertise is working against you.

## The Curse of Expertise

When you're a beginner, everything is new. You have no expectations, no ingrained habits. You absorb information without resistance.

But when you're an expert learning a new language, you bring seventeen years of baggage. Every concept in Go gets filtered through your PHP lens. You see structs and think "classes without inheritance." You see error returns and think "exceptions that forgot how to throw." You see explicit imports and think "why isn't there autoloading?"

This filtering isn't conscious. It happens before you can stop it. And it's exactly what makes the transition so difficult.

### The Expertise Trap

In PHP, you've developed what cognitive scientists call "chunking"—the ability to see complex patterns as single units. When you look at a Symfony controller, you don't see individual lines of code; you see a coherent whole.

```php
#[Route('/users/{id}', methods: ['GET'])]
public function show(int $id, UserRepository $repo): Response
{
    $user = $repo->find($id);
    if (!$user) {
        throw new NotFoundHttpException();
    }
    return $this->json($user);
}
```

You don't consciously process the autowiring, the parameter conversion, the exception handling, the JSON serialisation. It's all one mental unit: "fetch user, return JSON."

In Go, that same operation looks like this:

```go
func (h *UserHandler) Show(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.PathValue("id"))
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }

    user, err := h.repo.Find(r.Context(), id)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    if err := json.NewEncoder(w).Encode(user); err != nil {
        http.Error(w, "encoding error", http.StatusInternalServerError)
        return
    }
}
```

Your PHP brain sees this and screams: "Why is this so verbose? Where's the magic? Why do I have to handle every error manually?"

But a Go developer sees something different: explicit, testable, and obvious code where nothing is hidden.

## Interpreted vs Compiled: More Than Just Speed

PHP and Go differ fundamentally in how they execute. PHP interprets your code at runtime; Go compiles it to machine code. This isn't just a performance detail—it shapes everything about how the languages work.

### PHP's Runtime Flexibility

In PHP, code is evaluated at runtime. This enables powerful features:

```php
// Dynamic method calls
$method = 'processOrder';
$service->$method($order);

// Runtime class discovery
$handlers = glob(__DIR__ . '/Handlers/*.php');
foreach ($handlers as $file) {
    require_once $file;
}

// Magic methods
public function __call($name, $args) {
    // Handle any method dynamically
}
```

This flexibility is incredibly powerful. It's what makes frameworks like Symfony possible—autowiring, event dispatching, and annotation processing all rely on runtime introspection.

### Go's Compile-Time Rigidity

Go resolves everything at compile time. There's no runtime class loading, no dynamic method discovery, no magic methods:

```go
// This won't compile - method must exist
method := "ProcessOrder"
service.method(order)  // Error: method is not a field

// No glob-and-load pattern
// All imports must be explicit and known at compile time

// No magic methods
// If a method doesn't exist, it doesn't exist
```

This seems limiting. But it means that if your Go code compiles, entire categories of errors are impossible:

- No "method not found" at runtime
- No typos in method names that only fail in production
- No missing dependencies discovered during a critical deployment

### The Safety Trade-off

PHP trusts you to get things right at runtime. Go forces you to prove correctness at compile time. Neither is wrong, but they require different mental approaches.

In PHP, you might write:

```php
$user = $repo->findOrFail($id);  // Might throw, might not
$user->activate();  // Hope $user has this method
```

In Go, you must be explicit:

```go
user, err := repo.Find(ctx, id)
if err != nil {
    return nil, err  // Handle the error now
}
user.Activate()  // Compiler guarantees this exists
```

## Dynamic vs Static: The Freedom You're Losing (and Gaining)

PHP's dynamic typing is one of its most defining features:

```php
function process($data) {
    if (is_array($data)) {
        return array_map(fn($x) => $x * 2, $data);
    }
    return $data * 2;
}

process(5);       // 10
process([1, 2]);  // [2, 4]
```

This flexibility is why PHP is so productive for rapid prototyping. You don't waste time declaring types—you just write code that works.

### What You're Losing

In Go, every value has a single type, known at compile time:

```go
// This isn't possible in Go
func process(data any) any {
    // You'd need type assertions and it would be ugly
}

// Instead, you write separate functions or use generics
func processInt(data int) int {
    return data * 2
}

func processSlice(data []int) []int {
    result := make([]int, len(data))
    for i, v := range data {
        result[i] = v * 2
    }
    return result
}
```

You're losing the ability to write "it works on anything" functions easily. You're losing the convenience of not thinking about types until you need to.

### What You're Gaining

But you're gaining something valuable: certainty.

In PHP, this code compiles and runs:

```php
function calculateTotal(array $items): float
{
    return array_sum(array_column($items, 'price'));
}

// Called with wrong data
calculateTotal(['not', 'items']);  // Returns 0, silently wrong
```

In Go, type mismatches are caught at compile time:

```go
type Item struct {
    Price float64
}

func calculateTotal(items []Item) float64 {
    var total float64
    for _, item := range items {
        total += item.Price
    }
    return total
}

// Called with wrong data
calculateTotal([]string{"not", "items"})  // Won't compile
```

The Go compiler acts as a proofreader that catches entire categories of errors before your code ever runs.

## "It Just Works" vs "Prove It Works"

PHP culture values pragmatism. Get it working, ship it, iterate. This approach built the modern web.

```php
// Symfony's magic - it just works
#[Required]
public function setLogger(LoggerInterface $logger): void
{
    $this->logger = $logger;
}
```

How does `#[Required]` work? How does Symfony know to call this method? How does it find the LoggerInterface implementation? You don't need to know. It just works.

Go culture values explicitness. Show your work. Make everything visible.

```go
// Go's explicitness - prove it works
func NewService(logger *slog.Logger) *Service {
    return &Service{logger: logger}
}

// Caller
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
service := NewService(logger)
```

Nothing is hidden. Every dependency is explicitly passed. There's no container, no autowiring, no magic.

### The Debugging Difference

When Symfony's autowiring breaks, you're debugging framework internals:

```
Could not autowire service "App\Service\OrderService":
argument "$repository" of method "__construct()" references
interface "App\Repository\OrderRepositoryInterface" but no
such service exists.
```

When Go code fails, you're debugging your code:

```
./main.go:15:23: cannot use repo (variable of type *OrderRepository)
as OrderRepositoryInterface value in argument to NewOrderService:
*OrderRepository does not implement OrderRepositoryInterface
(missing method FindByUser)
```

Both errors tell you what's wrong. But Go's error points directly at your code and the specific missing method.

## The Discomfort Is the Learning

If Go feels awkward, that's not a sign that something is wrong with Go or with you. It's a sign that learning is happening.

Your PHP mental models are deeply ingrained. They took years to build. Replacing them with Go mental models takes time and deliberate practice.

Every time you feel the urge to:
- Create a base class (Go has no inheritance)
- Throw an exception (Go uses error returns)
- Use a magic method (Go has no magic)
- Let the framework handle it (Go uses explicit wiring)

...you're feeling the boundary between your old mental model and the new one. That friction is productive.

### Embracing the Beginner's Mind

The fastest path through this transition is to temporarily let go of your expertise. Approach Go as if PHP didn't exist. Accept that things will feel verbose, explicit, and perhaps even primitive.

Then watch as the patterns start making sense. As the verbosity reveals clarity. As the explicitness enables confidence.

The goal isn't to forget PHP. It's to add Go's mental models alongside your existing ones, and to know when to apply which.

## Summary

- **Expertise is a double-edged sword**: Your PHP knowledge filters how you see Go, often unhelpfully
- **Interpreted vs compiled** changes everything about how you think about code correctness
- **Dynamic vs static typing** trades flexibility for certainty
- **Explicitness vs magic** trades convenience for clarity
- **The discomfort is productive**: It means your mental models are being rewired

---

## Exercises

1. **Error Archaeology**: Take a PHP project you know well. Find three places where errors could occur at runtime but wouldn't be caught by static analysis. How would Go's type system prevent each?

2. **Magic Inventory**: List all the "magic" features your favourite Symfony application uses (autowiring, annotations, event listeners, etc.). For each, describe what would need to be explicit in Go.

3. **Expertise Audit**: Write down five PHP patterns that feel "obvious" to you. For each, explain what assumptions underlie the pattern. Which assumptions don't hold in Go?

4. **Compile-Time Proof**: Take a simple PHP function and rewrite it in Go. Identify all the checks that move from runtime to compile time.

5. **Verbosity Analysis**: Compare equivalent operations in PHP and Go (e.g., HTTP handler, JSON processing). Count the lines of code. Then count the explicit decisions made in each version. What's the ratio?

6. **Dynamic Challenge**: Write PHP code that uses dynamic typing heavily (e.g., a function that accepts mixed input types). Consider how you would restructure this for Go's type system.

7. **Framework Dependency Map**: Draw a diagram showing everything Symfony does implicitly when handling a single HTTP request. How many of these steps would be explicit in Go?

8. **Beginner's Mind Exercise**: Explain a Go concept (channels, goroutines, or interfaces) to yourself as if you'd never programmed before. Notice where PHP concepts intrude on your explanation.
