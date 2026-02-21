# Chapter 5: From Classes to Structs

PHP classes are rich constructs with constructors, destructors, inheritance, traits, interfaces, visibility modifiers, magic methods, and more. Go structs are deliberately simple: they're just collections of fields.

This simplicity is jarring at first, but it leads to cleaner designs.

## No Constructors: The `New*` Pattern

In PHP, the constructor is special:

```php
class User
{
    public function __construct(
        private string $name,
        private string $email,
        private DateTimeImmutable $createdAt = new DateTimeImmutable(),
    ) {}
}

$user = new User('Alice', 'alice@example.com');
```

Go has no constructors. You create structs directly or via factory functions:

```go
type User struct {
    Name      string
    Email     string
    CreatedAt time.Time
}

// Direct creation (all fields)
user := User{
    Name:      "Alice",
    Email:     "alice@example.com",
    CreatedAt: time.Now(),
}

// Factory function (conventional)
func NewUser(name, email string) *User {
    return &User{
        Name:      name,
        Email:     email,
        CreatedAt: time.Now(),
    }
}

user := NewUser("Alice", "alice@example.com")
```

### The `New*` Convention

The `New*` prefix is Go's convention for factory functions:

- `NewUser(name, email)` — create a User
- `NewServer(config)` — create a Server
- `NewClient(options...)` — create a Client with options

These aren't special—they're just functions that return your type. But they provide:

1. **Validation**: Check invariants before creation
2. **Defaults**: Set fields callers shouldn't specify
3. **Privacy**: Work with unexported fields

```go
func NewUser(name, email string) (*User, error) {
    if name == "" {
        return nil, errors.New("name is required")
    }
    if !strings.Contains(email, "@") {
        return nil, errors.New("invalid email")
    }
    return &User{
        Name:      name,
        Email:     email,
        CreatedAt: time.Now(),
        id:        uuid.New(),  // unexported field
    }, nil
}
```

### When to Use Direct Struct Literals

Not everything needs a factory function. Use struct literals for:

- **Simple value types**: `Point{X: 10, Y: 20}`
- **Configuration structs**: `Config{Port: 8080, Debug: true}`
- **Test data**: `User{Name: "test"}`

Use `New*` functions when you need:

- **Validation**: Ensure invariants hold
- **Defaults**: Set fields automatically
- **Unexported fields**: Access private state
- **Non-trivial setup**: Connect, initialise, register

## Methods as Functions with Receivers

PHP methods live inside the class:

```php
class Calculator
{
    private int $value = 0;

    public function add(int $n): self
    {
        $this->value += $n;
        return $this;
    }

    public function getValue(): int
    {
        return $this->value;
    }
}
```

Go methods are functions declared with a receiver:

```go
type Calculator struct {
    value int
}

func (c *Calculator) Add(n int) *Calculator {
    c.value += n
    return c
}

func (c *Calculator) Value() int {
    return c.value
}
```

The receiver `(c *Calculator)` is like `$this`—it's the instance the method operates on. But there's a key difference: the receiver is *explicit*.

### The Explicit Receiver

In PHP, `$this` is implicit—you don't declare it:

```php
public function getName(): string
{
    return $this->name;  // $this appears magically
}
```

In Go, the receiver is part of the function signature:

```go
func (u *User) Name() string {
    return u.name  // u is explicitly declared
}
```

You can name the receiver anything, but convention is:

- Use the first letter of the type: `u` for `User`, `s` for `Server`
- Be consistent within a type
- Avoid generic names like `this` or `self`

### Methods Are Just Functions

Syntactically, methods are sugar for functions with the receiver as the first parameter:

```go
// Method syntax
func (u *User) Greet() string {
    return "Hello, " + u.Name
}
user.Greet()

// Equivalent function call
(*User).Greet(&user)  // Method expression
```

This isn't just trivia—it means methods can be passed as functions:

```go
greet := user.Greet  // Method value
fmt.Println(greet())  // Calls user.Greet()

// Or extract method for a type
greetFunc := (*User).Greet
fmt.Println(greetFunc(&user))
```

## Value Receivers vs Pointer Receivers

This is one of Go's most confusing aspects for PHP developers.

### PHP: Always References (Sort Of)

In PHP, objects are always passed by reference (technically, by handle):

```php
function modify(User $user): void
{
    $user->name = 'Modified';  // Affects the original
}
```

### Go: Value vs Pointer Receivers

In Go, you choose:

```go
// Value receiver: operates on a copy
func (u User) FullName() string {
    return u.FirstName + " " + u.LastName
}

// Pointer receiver: operates on the original
func (u *User) SetName(name string) {
    u.FirstName = name  // Modifies the original
}
```

### When to Use Which

**Use pointer receiver when:**
- The method modifies the receiver
- The struct is large (avoid copying)
- Consistency—if some methods need pointers, use pointers for all

**Use value receiver when:**
- The struct is small and immutable
- The method doesn't modify state
- You want a defensive copy

```go
// Small immutable type: value receivers
type Point struct {
    X, Y float64
}

func (p Point) Distance(other Point) float64 {
    dx := p.X - other.X
    dy := p.Y - other.Y
    return math.Sqrt(dx*dx + dy*dy)
}

// Larger mutable type: pointer receivers
type Server struct {
    config Config
    router *Router
    db     *sql.DB
    // ... more fields
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

### The Automatic Dereference

Go automatically takes addresses and dereferences for method calls:

```go
user := User{Name: "Alice"}
user.SetName("Bob")  // Go converts to (&user).SetName("Bob")

userPtr := &User{Name: "Carol"}
userPtr.FullName()   // Go converts to (*userPtr).FullName()
```

This convenience can mask whether you're working with a pointer or value—be mindful when it matters.

## Where Did `$this` Go?

In PHP, `$this` is always available in non-static methods:

```php
class Service
{
    public function process(): void
    {
        $this->validate();
        $this->save();
        $this->notify();
    }
}
```

In Go, the receiver name replaces `$this`:

```go
func (s *Service) Process() {
    s.validate()
    s.save()
    s.notify()
}
```

The explicit naming has benefits:

1. **Clarity**: You see exactly what `s` is
2. **Shadowing prevention**: No conflict with local `this` variable
3. **Consistency**: Same pattern in all methods

But it requires adjustment. Your fingers will type `$this->` and Go will complain.

## Private/Public via Case (No Keywords)

PHP uses visibility keywords:

```php
class User
{
    private string $id;
    protected string $name;
    public string $email;

    private function validate(): void { }
    public function save(): void { }
}
```

Go uses capitalisation:

```go
type User struct {
    id    string  // unexported (private to package)
    Name  string  // exported (public)
    Email string  // exported
}

func (u *User) validate() { }  // unexported
func (u *User) Save() { }      // exported
```

- **Uppercase first letter**: Exported (public)
- **Lowercase first letter**: Unexported (private to the package)

There's no "protected" equivalent. Unexported means only the package can access it—not subpackages, not embedding types.

### Package-Level Privacy

Importantly, unexported fields are visible to all code in the same package:

```go
// user.go
type User struct {
    id   string
    Name string
}

// repository.go (same package)
func (r *Repo) Save(u *User) error {
    // Can access u.id because we're in the same package
    return r.db.Exec("INSERT INTO users (id, name) VALUES (?, ?)", u.id, u.Name)
}
```

This is different from PHP's private, which restricts access to the class itself.

## Symfony Services vs Go Structs

Let's convert a typical Symfony service to Go.

### Symfony Service

```php
#[AsService]
class OrderService
{
    public function __construct(
        private OrderRepository $repository,
        private PaymentGateway $payment,
        private MailerInterface $mailer,
        private LoggerInterface $logger,
    ) {}

    public function createOrder(Cart $cart): Order
    {
        $order = Order::fromCart($cart);

        $this->repository->save($order);

        $this->payment->charge($order);

        $this->mailer->send(
            new OrderConfirmationEmail($order)
        );

        $this->logger->info('Order created', ['id' => $order->getId()]);

        return $order;
    }
}
```

### Go Equivalent

```go
type OrderService struct {
    repository OrderRepository  // interface
    payment    PaymentGateway   // interface
    mailer     Mailer           // interface
    logger     *slog.Logger
}

func NewOrderService(
    repo OrderRepository,
    payment PaymentGateway,
    mailer Mailer,
    logger *slog.Logger,
) *OrderService {
    return &OrderService{
        repository: repo,
        payment:    payment,
        mailer:     mailer,
        logger:     logger,
    }
}

func (s *OrderService) CreateOrder(cart Cart) (Order, error) {
    order := OrderFromCart(cart)

    if err := s.repository.Save(order); err != nil {
        return Order{}, fmt.Errorf("saving order: %w", err)
    }

    if err := s.payment.Charge(order); err != nil {
        return Order{}, fmt.Errorf("charging payment: %w", err)
    }

    if err := s.mailer.Send(NewOrderConfirmationEmail(order)); err != nil {
        s.logger.Error("failed to send confirmation", "error", err, "order_id", order.ID)
        // Don't fail on email errors
    }

    s.logger.Info("order created", "id", order.ID)

    return order, nil
}
```

Key differences:

1. **No autowiring**: Dependencies are passed explicitly
2. **Factory function**: `NewOrderService` replaces constructor
3. **Error handling**: Each operation returns an error
4. **No attributes**: Configuration is explicit code

## Summary

- **No constructors**: Use `New*` factory functions for initialisation
- **Explicit receivers**: Methods declare their receiver like a parameter
- **Value vs pointer receivers**: Choose based on mutation and size
- **Case-based visibility**: Uppercase = exported, lowercase = unexported
- **Package-level privacy**: No class-level private—only package boundaries

---

## Exercises

1. **Constructor Migration**: Take three PHP classes with different constructor patterns (required parameters, optional with defaults, many dependencies). Convert each to Go using `New*` functions.

2. **Receiver Selection**: Write a Go type with 5 methods. Decide for each method whether it should use a value or pointer receiver. Justify each choice.

3. **Method Expression**: Write a Go program that extracts a method from a struct and passes it to another function. When would this be useful?

4. **Visibility Audit**: Take a PHP class with mixed visibility (private, protected, public). Convert to Go and note which fields would need restructuring due to package-level privacy.

5. **Zero Value Safety**: Design a Go struct where the zero value is invalid. Then redesign it so the zero value is usable. Which design is better?

6. **Builder Pattern**: Implement the builder pattern in Go for a complex struct. Compare to PHP's fluent setters.

7. **Service Conversion**: Convert a Symfony service with 5+ dependencies to Go. Include the wiring code in `main.go`. Count the lines of explicit wiring versus Symfony's implicit wiring.

8. **Immutable Types**: Design an immutable `Money` type in Go with value receiver methods that return new values. Compare to a mutable PHP Money class.
