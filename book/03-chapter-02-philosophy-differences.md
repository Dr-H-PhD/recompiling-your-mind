# Chapter 2: Philosophy Differences

PHP and Go emerged from different eras, different problems, and different worldviews. Understanding these philosophical differences is key to making the mental transition.

## PHP: "Get It Done, Fix It Later"

PHP was created in 1994 to make web development accessible. Rasmus Lerdorf famously didn't intend to create a programming language—he just wanted to track visits to his online resume.

This origin story matters. PHP was always about pragmatism, accessibility, and getting things working. The language evolved to solve immediate problems, often at the expense of long-term consistency.

### The Pragmatist's Toolkit

PHP's philosophy can be summarised as: "Make the common case easy."

```php
// Reading a file? One line.
$content = file_get_contents('data.json');

// JSON decode? One line.
$data = json_decode($content, true);

// Database query? A few lines.
$users = $pdo->query("SELECT * FROM users")->fetchAll();
```

No setup, no boilerplate, no ceremony. Just results.

This philosophy made PHP the language of the web. When you needed a website quickly, PHP delivered. When something broke, you fixed it in production. When the code got messy, you refactored later (or didn't).

### Symfony's Mature Pragmatism

Symfony brought discipline to PHP without abandoning pragmatism. It introduced:

- Conventions that reduce decisions
- Dependency injection for testability
- A component ecosystem for flexibility

But Symfony still embraces PHP's core philosophy. Magic methods, annotations, and autowiring all prioritise developer convenience over explicitness:

```php
#[Route('/api/users')]
class UserController extends AbstractController
{
    public function __construct(
        private UserRepository $users,  // Autowired
        private LoggerInterface $logger  // Autowired
    ) {}

    #[Route('/{id}', methods: ['GET'])]
    public function show(User $user): Response  // ParamConverter magic
    {
        return $this->json($user);  // Serializer magic
    }
}
```

How many implicit operations happen in this code? The route is parsed from annotations. The constructor parameters are autowired. The `$user` parameter is hydrated from the database via ParamConverter. The response is serialised by the Symfony Serializer.

None of this is visible. It just works.

## Go: "Do It Right, Do It Once"

Go was created in 2007 at Google to solve Google's problems: massive codebases, thousands of engineers, slow compile times, and dependency hell.

The creators—Rob Pike, Ken Thompson, and Robert Griesemer—had decades of experience with large-scale systems. They'd seen what happens when languages accumulate features: complexity compounds, codebases become unmaintainable, and build times grow without bound.

Go's philosophy is ruthlessly minimalist: include only what's essential, and make everything explicit.

### The Minimalist's Manifesto

Go's design principles:

- **One way to do things**: Less choice means less cognitive load
- **Explicit over implicit**: No hidden behaviour
- **Simplicity over expressiveness**: Readability trumps writability
- **Composition over inheritance**: Flat hierarchies
- **Fast compilation**: Measured in seconds, not minutes

This philosophy produces code that looks different from PHP:

```go
// Reading a file
content, err := os.ReadFile("data.json")
if err != nil {
    return nil, fmt.Errorf("reading data: %w", err)
}

// JSON decode
var data map[string]interface{}
if err := json.Unmarshal(content, &data); err != nil {
    return nil, fmt.Errorf("parsing JSON: %w", err)
}

// Database query
rows, err := db.QueryContext(ctx, "SELECT * FROM users")
if err != nil {
    return nil, fmt.Errorf("querying users: %w", err)
}
defer rows.Close()

var users []User
for rows.Next() {
    var u User
    if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
        return nil, fmt.Errorf("scanning row: %w", err)
    }
    users = append(users, u)
}
```

More lines of code? Absolutely. But also:
- Every error is handled explicitly
- Every resource cleanup is visible (`defer rows.Close()`)
- No magic—you can trace exactly what happens

## Explicit Over Implicit (No Magic)

PHP culture embraces "magic"—behaviour that happens without explicit code. Symfony takes this further with:

- **Autowiring**: Dependencies appear without configuration
- **ParamConverters**: Request parameters become objects
- **Event listeners**: Code runs without being called
- **Annotations**: Metadata drives behaviour

```php
// How does this get called? Magic.
#[AsEventListener]
class OrderCreatedListener
{
    public function __invoke(OrderCreatedEvent $event): void
    {
        // This runs when OrderCreatedEvent is dispatched
        // But you can't tell from looking at the code
    }
}
```

Go rejects magic entirely. If something happens, the code shows it happening:

```go
// No magic - explicit subscription
type OrderService struct {
    listeners []func(Order)
}

func (s *OrderService) OnOrderCreated(fn func(Order)) {
    s.listeners = append(s.listeners, fn)
}

func (s *OrderService) CreateOrder(o Order) error {
    // ... create order ...

    // Explicit notification
    for _, listener := range s.listeners {
        listener(o)
    }
    return nil
}

// Wiring is explicit
service := &OrderService{}
service.OnOrderCreated(func(o Order) {
    log.Printf("Order created: %s", o.ID)
})
```

The PHP version is more concise. The Go version is more traceable. Neither is objectively better—they reflect different values.

## Simplicity Over Expressiveness

PHP provides many ways to express the same idea:

```php
// All valid ways to iterate
foreach ($items as $item) { ... }
array_map(fn($item) => ..., $items);
array_walk($items, function($item) { ... });
for ($i = 0; $i < count($items); $i++) { ... }
```

Go provides one way:

```go
// The only way to iterate a slice
for i, item := range items {
    // ...
}
```

PHP provides many ways to declare functions:

```php
function named($x) { return $x * 2; }
$lambda = function($x) { return $x * 2; };
$arrow = fn($x) => $x * 2;
$method = [$object, 'method'];
```

Go provides two, and they're clearly distinct:

```go
// Function declaration
func double(x int) int { return x * 2 }

// Function literal (closure)
double := func(x int) int { return x * 2 }
```

This limitation is intentional. When there's only one way to do something, code becomes consistent across teams, projects, and companies. Any Go code you read uses the same patterns.

## "A Little Copying Is Better Than a Little Dependency"

This Go proverb captures a fundamental difference from PHP culture.

In PHP/Composer land, you reach for packages freely:

```json
{
    "require": {
        "symfony/string": "^6.0",
        "nesbot/carbon": "^2.0",
        "ramsey/uuid": "^4.0",
        "league/csv": "^9.0"
    }
}
```

Each package brings transitive dependencies, potential conflicts, and maintenance burden. But the PHP community considers this normal—packages are how you avoid reinventing wheels.

Go culture is more conservative:

- The standard library is comprehensive and preferred
- Third-party packages require justification
- Copying small utility functions is acceptable

```go
// Go developer's typical response to "which UUID library?"
import "github.com/google/uuid"  // Just this one, it's from Google

// But for simpler utilities, just write it
func formatBytes(b int64) string {
    const unit = 1024
    if b < unit {
        return fmt.Sprintf("%d B", b)
    }
    // ... simple formatting code ...
}
```

The overhead of importing a package for `formatBytes` isn't worth the dependency. In PHP, you might import `league/bytes` without thinking twice.

## Why Go Feels Boring (And Why That's Good)

Coming from PHP's expressiveness, Go can feel painfully boring:

- No generics until recently (1.18)
- No exceptions
- No inheritance
- No magic methods
- No annotations
- No operator overloading
- No function overloading

This is by design. Go optimises for reading code, not writing it. When every codebase uses the same limited feature set, you can read any Go code fluently.

Compare two hypothetical codebases:

**PHP Project A** might use:
- Traits extensively
- Magic methods for ORM
- Annotations for routing
- Custom Collection classes with operator overloading

**PHP Project B** might use:
- Interfaces exclusively (no traits)
- Explicit repository patterns
- YAML routing
- Plain arrays with array functions

Both are valid PHP, but reading one after the other requires mental context-switching.

**Go Project A** and **Go Project B** will look almost identical. They'll use the same patterns because Go's feature set is small enough that everyone converges on similar solutions.

This consistency has profound benefits for large organisations and open source. Any Go developer can contribute to any Go project with minimal ramp-up time.

## Symfony's "Magic" vs Go's Transparency

Let's examine a concrete example: dependency injection.

### Symfony's Approach

```yaml
# services.yaml (usually autoconfigured)
services:
    _defaults:
        autowire: true
        autoconfigure: true

    App\:
        resource: '../src/'
```

```php
class OrderService
{
    public function __construct(
        private OrderRepository $repository,
        private MailerInterface $mailer,
        private LoggerInterface $logger,
    ) {}
}
```

How does Symfony know which implementations to inject? It scans your codebase, reads interfaces, matches types, and wires everything together. The process involves:

1. Compiler passes
2. Service definitions
3. Autowiring logic
4. Proxy generation (for lazy services)
5. Container compilation

This is powerful but opaque. When it works, it's magical. When it breaks, you're debugging XML service definitions and compiler pass execution order.

### Go's Approach

```go
type OrderService struct {
    repository OrderRepository
    mailer     Mailer
    logger     *slog.Logger
}

func NewOrderService(repo OrderRepository, mailer Mailer, logger *slog.Logger) *OrderService {
    return &OrderService{
        repository: repo,
        mailer:     mailer,
        logger:     logger,
    }
}

// In main.go
func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    db := connectToDatabase()
    repo := NewOrderRepository(db)
    mailer := NewSMTPMailer(smtpConfig)

    orderService := NewOrderService(repo, mailer, logger)
    // Use orderService
}
```

Every dependency is explicitly constructed and passed. There's no scanning, no matching, no magic. The wiring code might be tedious to write, but it's trivial to understand and debug.

If you prefer tooling assistance, code generators like [Wire](https://github.com/google/wire) can generate the wiring code—but they do so at compile time, producing explicit code you can read.

## Summary

- **PHP's pragmatism** prioritises getting things done quickly; Go's minimalism prioritises long-term maintainability
- **Magic vs explicitness** is a trade-off between convenience and traceability
- **Feature richness vs simplicity** affects code consistency across projects
- **Dependency culture** differs significantly between the ecosystems
- **Go's "boring" design** enables universal readability

---

## Exercises

1. **Philosophy Archaeology**: Read the original PHP RFC for a feature (e.g., attributes, arrow functions). Then read a Go proposal that was rejected. Compare the reasoning. What values drive each decision?

2. **Magic Removal**: Take a Symfony controller with autowiring, ParamConverters, and serialisation groups. Rewrite it with everything explicit—no framework magic. How many hidden steps become visible?

3. **Consistency Check**: Find three open-source Go projects in different domains (web, CLI, library). Note the structural similarities. Then do the same for three PHP projects. Which ecosystem shows more consistency?

4. **Dependency Audit**: Run `composer show` on a PHP project. Count the total number of packages (direct + transitive). Then run `go mod graph` on a Go project. Compare the dependency counts and discuss why they differ.

5. **Simplicity Exercise**: Implement a simple in-memory cache in PHP three different ways (array, class with magic methods, class with explicit methods). Then implement it in Go. Which PHP version is closest to the Go version?

6. **One-Way Principle**: List five things PHP allows multiple ways to do. For each, explain Go's singular approach. Do you lose expressiveness or gain consistency?

7. **Boring Code Review**: Write the most "clever" PHP code you can—using all available language features expressively. Then write equivalent Go code. Which would you prefer to maintain in five years?

8. **Values Reflection**: Write a short essay explaining which philosophy (PHP's or Go's) matches your personal values as a developer. Has it changed since you started learning Go?
