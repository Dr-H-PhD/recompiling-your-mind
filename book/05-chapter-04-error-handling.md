# Chapter 4: Error Handling — The Hardest Shift

If there's one aspect of Go that drives PHP developers crazy, it's error handling. The constant `if err != nil` checks feel primitive, verbose, and frankly annoying.

But error handling is where Go's philosophy shines most clearly. Once you internalise it, you'll understand why many Go developers consider it superior to exceptions.

## Why `if err != nil` Feels Wrong at First

Your PHP brain has been trained to expect exceptions:

```php
public function getUser(int $id): User
{
    $user = $this->repository->find($id);
    if (!$user) {
        throw new UserNotFoundException($id);
    }
    return $user;
}

// Caller
try {
    $user = $service->getUser($id);
    // Happy path continues
} catch (UserNotFoundException $e) {
    // Handle error
}
```

The error handling is separated from the main logic. You write the happy path, and exceptions handle the unhappy path elsewhere.

Now look at Go:

```go
func (s *Service) GetUser(id int) (User, error) {
    user, err := s.repository.Find(id)
    if err != nil {
        return User{}, fmt.Errorf("getting user %d: %w", id, err)
    }
    return user, nil
}

// Caller
user, err := service.GetUser(id)
if err != nil {
    // Handle error
}
// Happy path continues
```

The error check interrupts the flow. Every function that can fail returns an error. Every call site checks it. The happy path is littered with error handling.

This feels *wrong* when you're used to exceptions. Where's the separation of concerns? Why is error handling polluting every function?

### The Visibility Trade-off

Consider this PHP code:

```php
public function processOrder(Order $order): void
{
    $this->validator->validate($order);
    $payment = $this->paymentGateway->charge($order);
    $this->inventory->reserve($order->getItems());
    $this->mailer->sendConfirmation($order, $payment);
    $this->analytics->track('order.completed', $order);
}
```

Clean, readable, focused. But how many ways can this fail? Each method might throw. The `validate` might throw multiple exception types. The `charge` could fail for network, fraud, or insufficient funds. The `reserve` could fail if items are out of stock.

None of these failure modes are visible. You'd need to read each method's implementation or documentation to know what might happen.

Now the Go version:

```go
func (s *Service) ProcessOrder(order Order) error {
    if err := s.validator.Validate(order); err != nil {
        return fmt.Errorf("validating order: %w", err)
    }
    payment, err := s.paymentGateway.Charge(order)
    if err != nil {
        return fmt.Errorf("charging payment: %w", err)
    }
    if err := s.inventory.Reserve(order.Items); err != nil {
        return fmt.Errorf("reserving inventory: %w", err)
    }
    if err := s.mailer.SendConfirmation(order, payment); err != nil {
        return fmt.Errorf("sending confirmation: %w", err)
    }
    if err := s.analytics.Track("order.completed", order); err != nil {
        // Log but don't fail on analytics errors
        s.logger.Error("analytics tracking failed", "error", err)
    }
    return nil
}
```

More verbose? Yes. But look at what's visible:
- Every operation that can fail is marked with `err`
- You can see exactly how each error is handled
- The analytics error is explicitly logged but not propagated
- The error context ("validating order", "charging payment") creates an error trail

## Exceptions vs Explicit Errors: The Philosophical Divide

Exceptions and error returns represent fundamentally different philosophies.

### Exceptions: Errors as Exceptional Events

The exception model treats errors as *exceptional*—things that shouldn't happen in normal operation. When they occur, control flow jumps to a handler, potentially far up the call stack:

```php
// Deep in the call stack
public function parseConfig(string $json): array
{
    $config = json_decode($json, true);
    if (json_last_error() !== JSON_ERROR_NONE) {
        throw new ConfigParseException(json_last_error_msg());
    }
    return $config;
}

// Far up the call stack
public function bootstrap(): void
{
    try {
        $config = $this->loadConfig();
        $this->initializeServices($config);
        $this->startServer();
    } catch (ConfigParseException $e) {
        // Handle config errors
    } catch (ServiceException $e) {
        // Handle service errors
    } catch (Exception $e) {
        // Catch-all
    }
}
```

The error handling is centralised. The intervening code doesn't need to know about or handle the exceptions—they bubble up automatically.

### Error Returns: Errors as Values

Go treats errors as ordinary values—data to be inspected, transformed, and passed along:

```go
// Deep in the call stack
func parseConfig(data []byte) (Config, error) {
    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        return Config{}, fmt.Errorf("parsing config: %w", err)
    }
    return config, nil
}

// Each level handles or propagates
func loadConfig() (Config, error) {
    data, err := os.ReadFile("config.json")
    if err != nil {
        return Config{}, fmt.Errorf("reading config file: %w", err)
    }
    return parseConfig(data)
}

func bootstrap() error {
    config, err := loadConfig()
    if err != nil {
        return fmt.Errorf("loading config: %w", err)
    }
    // ... continue
}
```

Every function explicitly handles or propagates errors. There's no invisible control flow—you can trace the error path by reading the code linearly.

### Why Go Chose Explicit Errors

Go's designers had experience with exceptions in other languages and found them problematic:

1. **Invisible control flow**: Exceptions can jump anywhere, making code flow unpredictable
2. **Easy to forget**: It's easy to omit `catch` blocks for exceptions you didn't know could occur
3. **Cleanup complexity**: `finally` blocks and exception-safe code are error-prone
4. **Performance**: Exception handling has runtime overhead

Error values solve these issues:
- Control flow is explicit and linear
- The return type forces you to acknowledge errors
- Cleanup uses `defer`, which is straightforward
- Error handling is just function return overhead

## Error Wrapping and the `%w` Verb

PHP exceptions carry their own context—message, code, stack trace:

```php
throw new OrderException(
    "Payment failed for order $orderId",
    code: OrderException::PAYMENT_FAILED,
    previous: $paymentException
);
```

Go errors are simpler by design, but can be wrapped to build context:

```go
// Basic error
return errors.New("payment failed")

// With context using fmt.Errorf
return fmt.Errorf("order %s: payment failed", orderID)

// Wrapping another error (preserves the original)
return fmt.Errorf("processing order %s: %w", orderID, err)
```

The `%w` verb is crucial. It wraps the error while preserving the original, allowing inspection with `errors.Is` and `errors.As`:

```go
var ErrNotFound = errors.New("not found")

func (r *Repo) Find(id int) (User, error) {
    // ...
    return User{}, ErrNotFound
}

func (s *Service) GetUser(id int) (User, error) {
    user, err := s.repo.Find(id)
    if err != nil {
        return User{}, fmt.Errorf("finding user %d: %w", id, err)
    }
    return user, nil
}

// Caller can check for specific error
user, err := service.GetUser(42)
if errors.Is(err, ErrNotFound) {
    // Handle not found specifically
}
```

The wrapped error message might be `"finding user 42: not found"` but `errors.Is` still recognises it as `ErrNotFound`.

## Custom Error Types (Like Symfony's Custom Exceptions)

In PHP, you create custom exceptions by extending Exception:

```php
class ValidationException extends Exception
{
    private array $errors;

    public function __construct(array $errors)
    {
        $this->errors = $errors;
        parent::__construct('Validation failed');
    }

    public function getErrors(): array
    {
        return $this->errors;
    }
}
```

In Go, any type implementing the `error` interface is an error:

```go
// The error interface
type error interface {
    Error() string
}

// Custom error type
type ValidationError struct {
    Fields map[string]string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed: %d errors", len(e.Fields))
}

// Usage
func Validate(user User) error {
    errors := make(map[string]string)
    if user.Email == "" {
        errors["email"] = "required"
    }
    if len(errors) > 0 {
        return &ValidationError{Fields: errors}
    }
    return nil
}

// Caller extracts details
var validationErr *ValidationError
if errors.As(err, &validationErr) {
    for field, msg := range validationErr.Fields {
        fmt.Printf("%s: %s\n", field, msg)
    }
}
```

`errors.As` unwraps to find an error of a specific type, similar to `catch (ValidationException $e)` in PHP.

## When to Panic (Almost Never)

Go has `panic` for truly exceptional situations:

```go
func mustParseURL(s string) *url.URL {
    u, err := url.Parse(s)
    if err != nil {
        panic(fmt.Sprintf("invalid URL: %s", s))
    }
    return u
}
```

But panic should be rare. It's for:

1. **Programmer errors**: Bugs that indicate broken invariants (like array out of bounds)
2. **Initialisation failures**: When the program can't continue (config missing at startup)
3. **Impossible states**: Conditions that "can't happen" but you want to detect

Never panic for:
- User input errors
- Network failures
- File not found
- Any error a caller might want to handle

The convention `Must*` (like `template.Must()`) indicates a function that panics on error—use only with known-good values or during initialisation.

## Learning to Love Explicit Error Paths

After enough Go code, something shifts. You start to appreciate:

### 1. Error Paths Are Visible

Reading Go code, you can trace exactly what happens on failure. No need to check documentation or source code for exception types.

### 2. Errors Get Context

Each level adds information:

```
"processing order abc123: charging payment: connecting to gateway: dial tcp: connection refused"
```

This error message tells you the entire call path. You know exactly where it failed.

### 3. Forced Consideration

The return type `(T, error)` forces you to decide: handle it, propagate it, or explicitly ignore it. You can't accidentally forget.

### 4. Easy Testing

Error paths are just return values—easy to test:

```go
func TestGetUser_NotFound(t *testing.T) {
    repo := &MockRepo{err: ErrNotFound}
    service := NewService(repo)

    _, err := service.GetUser(1)

    if !errors.Is(err, ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```

## No More Try/Catch Blocks

The absence of try/catch changes how you structure code:

### PHP: Group Operations, Handle Failures Together

```php
try {
    $user = $this->userService->find($id);
    $orders = $this->orderService->findByUser($user);
    $recommendations = $this->recService->forUser($user);
    return compact('user', 'orders', 'recommendations');
} catch (UserNotFoundException $e) {
    throw new NotFoundException("User not found");
} catch (ServiceException $e) {
    $this->logger->error("Service error", ['exception' => $e]);
    throw new InternalErrorException();
}
```

### Go: Handle Each Failure Inline

```go
user, err := s.userService.Find(id)
if err != nil {
    if errors.Is(err, ErrNotFound) {
        return nil, NewNotFoundError("user not found")
    }
    return nil, fmt.Errorf("finding user: %w", err)
}

orders, err := s.orderService.FindByUser(user)
if err != nil {
    s.logger.Error("failed to fetch orders", "error", err)
    // Continue without orders (graceful degradation)
    orders = nil
}

recommendations, err := s.recService.ForUser(user)
if err != nil {
    s.logger.Error("failed to fetch recommendations", "error", err)
    recommendations = nil
}

return &Response{User: user, Orders: orders, Recommendations: recommendations}, nil
```

The Go version makes decisions explicit at each step: propagate the error, transform it, log it, or ignore it.

## Summary

- **`if err != nil`** is verbose but makes every failure path visible
- **Errors as values** enable straightforward handling, wrapping, and testing
- **Error wrapping** (`%w`) builds context while preserving the original error
- **Custom error types** carry additional data, like custom exceptions
- **Panic** is for programmer errors, not expected failures
- **Explicit error handling** forces you to consider failures at every step

---

## Exercises

1. **Exception Inventory**: List all exception types thrown in a Symfony service class. Convert each to a Go error type or sentinel error. Compare the calling patterns.

2. **Error Context Chain**: Write a Go function that calls three other functions, each of which can fail. Wrap errors at each level with context. Verify the final error message contains the full trace.

3. **Graceful Degradation**: Design a Go service that calls three external APIs. If one fails, the others should still succeed. Compare to implementing the same with PHP exceptions.

4. **Custom Error Type**: Create a Go `ValidationError` type that holds multiple field errors. Implement `Error()` and write code using `errors.As` to extract the field errors.

5. **Panic vs Error**: Identify three scenarios where panic is appropriate and three where it's not. Implement examples of each.

6. **Error Handling Patterns**: Implement three different error handling strategies in Go:
   - Propagate with context
   - Transform to a different error type
   - Log and suppress (with explicit `_ =` ignore)

7. **Test Error Paths**: Write a table-driven test that verifies a function returns the correct error types for different failure scenarios.

8. **PHP to Go Migration**: Take a PHP controller action with multiple try/catch blocks. Convert it to Go with explicit error handling. Count the error checks. Does the code still read cleanly?
