# Chapter 10: Web Development Without a Framework

Symfony gives you everything: routing, controllers, request handling, response building, middleware, sessions, security. In Go, you build these yourself—but it's easier than you think.

## Building HTTP Handlers

Symfony controllers are classes with action methods:

```php
class UserController extends AbstractController
{
    #[Route('/users/{id}', methods: ['GET'])]
    public function show(int $id): Response
    {
        $user = $this->userRepository->find($id);
        return $this->json($user);
    }
}
```

Go handlers are functions with a specific signature:

```go
func (h *UserHandler) Show(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")  // Go 1.22+
    user, err := h.repo.Find(r.Context(), id)
    if err != nil {
        http.Error(w, "user not found", http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```

### The Handler Interface

Go's `http.Handler` interface is simple:

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

Any type implementing this method is a handler. The `http.HandlerFunc` adapter turns functions into handlers:

```go
// Function
func hello(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello!")
}

// Convert to Handler
var h http.Handler = http.HandlerFunc(hello)
```

### Handler Structs

For handlers with dependencies, use structs:

```go
type UserHandler struct {
    repo   UserRepository
    logger *slog.Logger
}

func NewUserHandler(repo UserRepository, logger *slog.Logger) *UserHandler {
    return &UserHandler{repo: repo, logger: logger}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
    users, err := h.repo.FindAll(r.Context())
    if err != nil {
        h.logger.Error("failed to list users", "error", err)
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    writeJSON(w, http.StatusOK, users)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    var input CreateUserInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }
    user, err := h.repo.Create(r.Context(), input)
    if err != nil {
        http.Error(w, "failed to create user", http.StatusInternalServerError)
        return
    }
    writeJSON(w, http.StatusCreated, user)
}
```

## Middleware Patterns (Like Symfony Middlewares)

Symfony uses event listeners and kernel events for cross-cutting concerns:

```php
class AuthenticationListener
{
    public function onKernelRequest(RequestEvent $event): void
    {
        $request = $event->getRequest();
        // Check authentication
    }
}
```

Go uses middleware—functions that wrap handlers:

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        slog.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "duration", time.Since(start),
        )
    })
}

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if !isValidToken(token) {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### Chaining Middleware

Stack middleware by nesting:

```go
handler := authMiddleware(loggingMiddleware(actualHandler))
```

Or create a helper:

```go
func chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
    for i := len(middlewares) - 1; i >= 0; i-- {
        h = middlewares[i](h)
    }
    return h
}

handler := chain(actualHandler, loggingMiddleware, authMiddleware)
```

### Passing Data Through Context

Symfony stores data in request attributes. Go uses context:

```go
type contextKey string

const userKey contextKey = "user"

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := authenticateUser(r)
        if user == nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        ctx := context.WithValue(r.Context(), userKey, user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func getUser(ctx context.Context) *User {
    user, _ := ctx.Value(userKey).(*User)
    return user
}

// In handler
func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
    user := getUser(r.Context())
    // ...
}
```

## Routing: `http.ServeMux` vs Symfony Routing

Symfony Routing is powerful:

```php
#[Route('/users/{id}', name: 'user_show', requirements: ['id' => '\d+'])]
public function show(int $id): Response { }

#[Route('/posts/{slug}', name: 'post_show')]
public function post(string $slug): Response { }
```

Go 1.22 improved `http.ServeMux` significantly:

```go
mux := http.NewServeMux()

// Method + path patterns
mux.HandleFunc("GET /users/{id}", userHandler.Show)
mux.HandleFunc("POST /users", userHandler.Create)
mux.HandleFunc("PUT /users/{id}", userHandler.Update)
mux.HandleFunc("DELETE /users/{id}", userHandler.Delete)

// Wildcards
mux.HandleFunc("GET /files/{path...}", fileHandler.Serve)

// Access path values
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")  // Extract {id}
}
```

### When You Need More

For complex routing (regex constraints, named routes, reverse routing), use a router package:

```go
// Using chi
r := chi.NewRouter()
r.Get("/users/{id:[0-9]+}", userHandler.Show)
r.Get("/posts/{slug:[a-z-]+}", postHandler.Show)
```

## Request Validation Without Annotations

Symfony Validator uses annotations:

```php
class CreateUserInput
{
    #[Assert\NotBlank]
    #[Assert\Email]
    public string $email;

    #[Assert\NotBlank]
    #[Assert\Length(min: 8)]
    public string $password;
}
```

Go doesn't have annotations. Use struct tags with a validation library:

```go
import "github.com/go-playground/validator/v10"

type CreateUserInput struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

var validate = validator.New()

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var input CreateUserInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }

    if err := validate.Struct(input); err != nil {
        errors := formatValidationErrors(err)
        writeJSON(w, http.StatusUnprocessableEntity, errors)
        return
    }

    // Input is valid
}
```

### Manual Validation

For simple cases, validate manually:

```go
func (input CreateUserInput) Validate() error {
    if input.Email == "" {
        return errors.New("email is required")
    }
    if !strings.Contains(input.Email, "@") {
        return errors.New("invalid email format")
    }
    if len(input.Password) < 8 {
        return errors.New("password must be at least 8 characters")
    }
    return nil
}
```

## Response Patterns

Create helper functions for common responses:

```go
func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
    writeJSON(w, status, map[string]string{"error": message})
}

// Usage
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
    user, err := h.repo.Find(r.Context(), id)
    if errors.Is(err, ErrNotFound) {
        writeError(w, http.StatusNotFound, "user not found")
        return
    }
    if err != nil {
        writeError(w, http.StatusInternalServerError, "internal error")
        return
    }
    writeJSON(w, http.StatusOK, user)
}
```

## Session Management Without Symfony Session

Symfony provides session management out of the box:

```php
$session = $request->getSession();
$session->set('user_id', $userId);
$userId = $session->get('user_id');
```

Go needs a session library. `gorilla/sessions` is popular:

```go
import "github.com/gorilla/sessions"

var store = sessions.NewCookieStore([]byte("secret-key"))

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    session, _ := store.Get(r, "session-name")
    session.Values["user_id"] = user.ID
    session.Save(r, w)
}

func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
    session, _ := store.Get(r, "session-name")
    userID, ok := session.Values["user_id"].(int)
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    // ...
}
```

### Stateless APIs

Many Go APIs are stateless, using JWTs instead of sessions:

```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tokenString := extractToken(r)
        claims, err := validateJWT(tokenString)
        if err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        ctx := context.WithValue(r.Context(), userKey, claims.UserID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Putting It Together: Complete Server

```go
func main() {
    // Dependencies
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    db := connectDatabase()
    userRepo := repository.NewUserRepository(db)
    userHandler := handler.NewUserHandler(userRepo, logger)

    // Router
    mux := http.NewServeMux()
    mux.HandleFunc("GET /users", userHandler.List)
    mux.HandleFunc("POST /users", userHandler.Create)
    mux.HandleFunc("GET /users/{id}", userHandler.Show)
    mux.HandleFunc("PUT /users/{id}", userHandler.Update)
    mux.HandleFunc("DELETE /users/{id}", userHandler.Delete)

    // Middleware stack
    handler := chain(mux,
        recoveryMiddleware,
        loggingMiddleware,
        corsMiddleware,
    )

    // Server
    server := &http.Server{
        Addr:         ":8080",
        Handler:      handler,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
    }

    logger.Info("server starting", "addr", server.Addr)
    if err := server.ListenAndServe(); err != nil {
        logger.Error("server error", "error", err)
    }
}
```

## HTTP Clients: Calling External APIs

PHP developers rely heavily on HTTP clients—Guzzle, Symfony HttpClient, or even `file_get_contents()`. Go's `net/http` package provides a powerful client that outperforms most PHP alternatives.

### PHP vs Go: Quick Comparison

```php
// Guzzle
$client = new GuzzleHttp\Client(['timeout' => 10]);
$response = $client->get('https://api.example.com/users');
$data = json_decode($response->getBody(), true);

// Symfony HttpClient
$client = HttpClient::create(['timeout' => 10]);
$response = $client->request('GET', 'https://api.example.com/users');
$data = $response->toArray();
```

```go
// Go standard library
resp, err := http.Get("https://api.example.com/users")
if err != nil {
    return err
}
defer resp.Body.Close()

var data []User
json.NewDecoder(resp.Body).Decode(&data)
```

### Basic Requests

```go
// GET request
resp, err := http.Get("https://api.example.com/users")
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("unexpected status: %d", resp.StatusCode)
}

body, err := io.ReadAll(resp.Body)

// POST with JSON
data := map[string]string{"name": "John", "email": "john@example.com"}
jsonBody, _ := json.Marshal(data)

resp, err := http.Post(
    "https://api.example.com/users",
    "application/json",
    bytes.NewReader(jsonBody),
)
```

### Custom HTTP Client

The default `http.DefaultClient` has no timeout—dangerous for production. Always create a custom client:

```go
client := &http.Client{
    Timeout: 10 * time.Second,
}

resp, err := client.Get("https://api.example.com/users")
```

### Transport Configuration

For high-performance applications, configure the transport layer:

```go
transport := &http.Transport{
    // Connection pooling
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,

    // Timeouts
    DialContext: (&net.Dialer{
        Timeout:   5 * time.Second,   // Connection timeout
        KeepAlive: 30 * time.Second,  // Keep-alive interval
    }).DialContext,
    TLSHandshakeTimeout:   5 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
    ResponseHeaderTimeout: 10 * time.Second,
}

client := &http.Client{
    Transport: transport,
    Timeout:   30 * time.Second,  // Total request timeout
}
```

**Key differences from PHP:**
- **Connection pooling is automatic**: Go reuses TCP connections
- **Thread-safe**: One client instance for all goroutines
- **No external dependencies**: Built into the standard library

### Making Complex Requests

For headers, authentication, or custom methods, use `http.NewRequest`:

```go
func callAPI(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, method, url, body)
    if err != nil {
        return nil, err
    }

    // Headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("User-Agent", "MyApp/1.0")

    // Custom client (reuse this!)
    return client.Do(req)
}

// Usage
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := callAPI(ctx, "PUT", "https://api.example.com/users/123", bytes.NewReader(jsonBody))
```

### JSON API Client Pattern

Create a reusable API client—similar to how you'd wrap Guzzle:

```go
type APIClient struct {
    baseURL    string
    httpClient *http.Client
    apiKey     string
}

func NewAPIClient(baseURL, apiKey string) *APIClient {
    return &APIClient{
        baseURL: baseURL,
        apiKey:  apiKey,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (c *APIClient) do(ctx context.Context, method, path string, body, result any) error {
    var bodyReader io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return err
        }
        bodyReader = bytes.NewReader(jsonBody)
    }

    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-API-Key", c.apiKey)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("API error %d: %s", resp.StatusCode, body)
    }

    if result != nil {
        return json.NewDecoder(resp.Body).Decode(result)
    }
    return nil
}

func (c *APIClient) GetUser(ctx context.Context, id string) (*User, error) {
    var user User
    err := c.do(ctx, "GET", "/users/"+id, nil, &user)
    return &user, err
}

func (c *APIClient) CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
    var user User
    err := c.do(ctx, "POST", "/users", input, &user)
    return &user, err
}
```

### Concurrent Requests

Go excels at making parallel requests—something PHP struggles with:

```go
func fetchAllUsers(ctx context.Context, ids []string) ([]*User, error) {
    users := make([]*User, len(ids))
    errors := make([]error, len(ids))

    var wg sync.WaitGroup
    for i, id := range ids {
        wg.Add(1)
        go func(idx int, userID string) {
            defer wg.Done()
            users[idx], errors[idx] = apiClient.GetUser(ctx, userID)
        }(i, id)
    }
    wg.Wait()

    // Check for errors
    for _, err := range errors {
        if err != nil {
            return nil, err
        }
    }
    return users, nil
}
```

### Rate Limiting

Respect API rate limits with a semaphore:

```go
func fetchWithRateLimit(ctx context.Context, urls []string, maxConcurrent int) ([][]byte, error) {
    results := make([][]byte, len(urls))
    sem := make(chan struct{}, maxConcurrent)

    var wg sync.WaitGroup
    var mu sync.Mutex
    var firstErr error

    for i, url := range urls {
        wg.Add(1)
        go func(idx int, u string) {
            defer wg.Done()

            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release

            resp, err := http.Get(u)
            if err != nil {
                mu.Lock()
                if firstErr == nil {
                    firstErr = err
                }
                mu.Unlock()
                return
            }
            defer resp.Body.Close()

            body, _ := io.ReadAll(resp.Body)
            results[idx] = body
        }(i, url)
    }

    wg.Wait()
    return results, firstErr
}
```

### Retry with Backoff

```go
func fetchWithRetry(ctx context.Context, url string, maxRetries int) (*http.Response, error) {
    var lastErr error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
        resp, err := client.Do(req)

        if err == nil && resp.StatusCode < 500 {
            return resp, nil
        }

        if resp != nil {
            resp.Body.Close()
        }
        lastErr = err

        // Exponential backoff
        backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
        select {
        case <-time.After(backoff):
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### Best Practices

1. **Reuse `http.Client`**: Create once, use everywhere—connections are pooled
2. **Always set timeouts**: The default client has none
3. **Always close `resp.Body`**: Use `defer resp.Body.Close()`
4. **Use context for cancellation**: Pass `ctx` through your call chain
5. **Check status codes**: Success doesn't guarantee 2xx
6. **Handle connection errors**: Networks fail; retry when appropriate

## When You Need a Framework: Gin and Echo

So far we've used only the standard library. But sometimes you want more structure—especially coming from Symfony. Go has excellent web frameworks that feel familiar.

### Why Consider a Framework?

| Need | net/http | Framework |
|------|----------|-----------|
| Basic routing | ✓ | ✓ |
| Path parameters | ✓ (Go 1.22+) | ✓ |
| Route groups | Manual | Built-in |
| Parameter validation | Manual | Built-in |
| JSON binding | Manual | One-liner |
| Error handling | Manual | Centralised |

For PHP developers: Gin/Echo are closer to Slim PHP than to Symfony—lightweight and fast.

### Gin: The Most Popular

Gin is the Symfony of Go frameworks—widely used, well-documented, battle-tested.

```bash
go get -u github.com/gin-gonic/gin
```

```go
package main

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()  // Includes logger and recovery middleware

    // Routes - similar to Symfony annotations
    r.GET("/users", listUsers)
    r.GET("/users/:id", getUser)      // :id like Symfony {id}
    r.POST("/users", createUser)
    r.PUT("/users/:id", updateUser)
    r.DELETE("/users/:id", deleteUser)

    r.Run(":8080")
}

// Handler - simpler than net/http
func getUser(c *gin.Context) {
    id := c.Param("id")  // Path parameter

    user, err := findUser(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }

    c.JSON(http.StatusOK, user)
}

// JSON binding - like Symfony's handleRequest()
func createUser(c *gin.Context) {
    var input CreateUserInput

    // Binds JSON and validates
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Input is bound and validated
    user := createUserFromInput(input)
    c.JSON(http.StatusCreated, user)
}
```

### Gin Middleware

```go
func main() {
    r := gin.New()  // Without default middleware

    // Global middleware
    r.Use(gin.Logger())
    r.Use(gin.Recovery())

    // Route groups with middleware - like Symfony firewall
    api := r.Group("/api")
    api.Use(authMiddleware())
    {
        api.GET("/users", listUsers)
        api.POST("/users", createUser)
    }

    // Admin routes with different middleware
    admin := r.Group("/admin")
    admin.Use(authMiddleware(), adminMiddleware())
    {
        admin.GET("/stats", getStats)
    }
}

// Gin middleware signature
func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if !isValid(token) {
            c.AbortWithStatusJSON(http.StatusUnauthorized,
                gin.H{"error": "unauthorized"})
            return
        }
        c.Set("user_id", extractUserID(token))  // Like request attributes
        c.Next()
    }
}
```

### Gin Validation with Tags

```go
type CreateUserInput struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    Name     string `json:"name" binding:"required,max=100"`
}

func createUser(c *gin.Context) {
    var input CreateUserInput

    // ShouldBindJSON validates using binding tags
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusUnprocessableEntity, gin.H{
            "error":   "validation failed",
            "details": err.Error(),
        })
        return
    }

    // All validation passed
}
```

### Echo: The Alternative

Echo is similar to Gin but with different design choices—slightly faster, different API style.

```bash
go get -u github.com/labstack/echo/v4
```

```go
package main

import (
    "net/http"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    e := echo.New()

    // Built-in middleware
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())

    // Routes
    e.GET("/users", listUsers)
    e.GET("/users/:id", getUser)
    e.POST("/users", createUser)

    e.Start(":8080")
}

// Echo handler - returns error
func getUser(c echo.Context) error {
    id := c.Param("id")

    user, err := findUser(id)
    if err != nil {
        return c.JSON(http.StatusNotFound, map[string]string{
            "error": "user not found",
        })
    }

    return c.JSON(http.StatusOK, user)
}

// Echo binding
func createUser(c echo.Context) error {
    var input CreateUserInput

    if err := c.Bind(&input); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": err.Error(),
        })
    }

    // Validate with external validator
    if err := c.Validate(&input); err != nil {
        return c.JSON(http.StatusUnprocessableEntity, map[string]string{
            "error": err.Error(),
        })
    }

    return c.JSON(http.StatusCreated, createUserFromInput(input))
}
```

### Echo Route Groups

```go
func main() {
    e := echo.New()

    // API group
    api := e.Group("/api")
    api.Use(authMiddleware)

    api.GET("/users", listUsers)
    api.POST("/users", createUser)

    // Nested groups
    v1 := api.Group("/v1")
    v1.GET("/legacy", legacyHandler)

    v2 := api.Group("/v2")
    v2.GET("/modern", modernHandler)
}

// Echo middleware signature
func authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        token := c.Request().Header.Get("Authorization")
        if !isValid(token) {
            return c.JSON(http.StatusUnauthorized, map[string]string{
                "error": "unauthorized",
            })
        }
        c.Set("user_id", extractUserID(token))
        return next(c)
    }
}
```

### Framework Comparison

| Feature | Gin | Echo | net/http |
|---------|-----|------|----------|
| Performance | Excellent | Excellent | Good |
| Learning curve | Low | Low | Lowest |
| JSON binding | Built-in | Built-in | Manual |
| Validation | Built-in | Plugin | Manual |
| Route groups | ✓ | ✓ | Manual |
| Middleware | ✓ | ✓ | Manual |
| WebSocket | Plugin | Built-in | Manual |
| Dependencies | Minimal | Minimal | None |

### When to Use What

**Use `net/http` when:**
- Building simple APIs
- Minimising dependencies
- Learning Go
- Maximum control needed

**Use Gin when:**
- Building larger APIs
- Team familiarity matters
- Need built-in validation
- Coming from Django/Flask

**Use Echo when:**
- Need WebSocket support
- Prefer error-returning handlers
- Building microservices
- Need automatic TLS

### Framework-Agnostic Tip

Design your business logic independently:

```go
// Service layer - no framework dependency
type UserService struct {
    repo UserRepository
}

func (s *UserService) Create(ctx context.Context, input CreateUserInput) (*User, error) {
    // Business logic here
}

// Gin handler
func ginCreateUser(svc *UserService) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input CreateUserInput
        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        user, err := svc.Create(c.Request.Context(), input)
        // ...
    }
}

// Echo handler - same service, different adapter
func echoCreateUser(svc *UserService) echo.HandlerFunc {
    return func(c echo.Context) error {
        var input CreateUserInput
        if err := c.Bind(&input); err != nil {
            return c.JSON(400, map[string]string{"error": err.Error()})
        }
        user, err := svc.Create(c.Request().Context(), input)
        // ...
    }
}
```

This is the Go equivalent of Symfony's hexagonal architecture—your domain logic stays clean.

## WebSockets: Real-Time Communication

PHP's traditional request-response model struggles with real-time features. Solutions like Ratchet or Swoole exist, but they require separate processes. Go handles WebSockets naturally with goroutines.

### Why WebSockets in Go?

- **Native concurrency**: Each connection runs in its own goroutine
- **Low overhead**: Thousands of connections with minimal memory
- **Same binary**: No separate WebSocket server process
- **Standard patterns**: Channels for message distribution

### Using gorilla/websocket

```go
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // Allow all origins in development
        return true
    },
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("upgrade error: %v", err)
        return
    }
    defer conn.Close()

    for {
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
                log.Printf("read error: %v", err)
            }
            break
        }

        // Echo message back
        if err := conn.WriteMessage(messageType, message); err != nil {
            log.Printf("write error: %v", err)
            break
        }
    }
}

func main() {
    http.HandleFunc("/ws", handleWebSocket)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Chat Room Pattern

A practical example: broadcast messages to all connected clients.

```go
type Client struct {
    conn *websocket.Conn
    send chan []byte
}

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan []byte),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()

        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()

        case message := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
            h.mu.RUnlock()
        }
    }
}

func (c *Client) readPump(hub *Hub) {
    defer func() {
        hub.unregister <- c
        c.conn.Close()
    }()

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
        hub.broadcast <- message
    }
}

func (c *Client) writePump() {
    defer c.conn.Close()

    for message := range c.send {
        if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
            break
        }
    }
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }

    client := &Client{
        conn: conn,
        send: make(chan []byte, 256),
    }

    hub.register <- client

    go client.writePump()
    go client.readPump(hub)
}
```

### Structured Messages

```go
type Message struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

type ChatMessage struct {
    User    string `json:"user"`
    Content string `json:"content"`
    Time    int64  `json:"time"`
}

func (c *Client) handleMessage(data []byte) {
    var msg Message
    if err := json.Unmarshal(data, &msg); err != nil {
        return
    }

    switch msg.Type {
    case "chat":
        var chat ChatMessage
        json.Unmarshal(msg.Payload, &chat)
        c.handleChat(chat)

    case "typing":
        c.handleTyping()

    case "ping":
        c.sendPong()
    }
}

func (c *Client) sendJSON(v interface{}) error {
    return c.conn.WriteJSON(v)
}
```

### Connection Management

```go
type Client struct {
    id       string
    conn     *websocket.Conn
    send     chan []byte
    hub      *Hub
    lastPing time.Time
}

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(512 * 1024)  // 512KB max message
    c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        c.lastPing = time.Now()
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
        c.handleMessage(message)
    }
}

func (c *Client) writePump() {
    ticker := time.NewTicker(30 * time.Second)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

### Client-Side JavaScript

```javascript
class WebSocketClient {
    constructor(url) {
        this.url = url;
        this.reconnectDelay = 1000;
        this.connect();
    }

    connect() {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
            console.log('Connected');
            this.reconnectDelay = 1000;
        };

        this.ws.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            this.handleMessage(msg);
        };

        this.ws.onclose = () => {
            console.log('Disconnected, reconnecting...');
            setTimeout(() => this.connect(), this.reconnectDelay);
            this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
        };
    }

    send(type, payload) {
        if (this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type, payload }));
        }
    }

    handleMessage(msg) {
        switch (msg.type) {
            case 'chat':
                this.onChat(msg.payload);
                break;
            case 'users':
                this.onUsers(msg.payload);
                break;
        }
    }
}

const client = new WebSocketClient('ws://localhost:8080/ws');
client.onChat = (msg) => console.log(`${msg.user}: ${msg.content}`);
```

### Scaling WebSockets

For multiple server instances, use Redis pub/sub:

```go
import "github.com/go-redis/redis/v8"

type DistributedHub struct {
    *Hub
    redis   *redis.Client
    channel string
}

func (h *DistributedHub) Run() {
    // Subscribe to Redis channel
    pubsub := h.redis.Subscribe(context.Background(), h.channel)
    defer pubsub.Close()

    go func() {
        for msg := range pubsub.Channel() {
            // Broadcast to local clients
            h.broadcastLocal([]byte(msg.Payload))
        }
    }()

    // Handle local events
    for {
        select {
        case client := <-h.register:
            h.addClient(client)

        case client := <-h.unregister:
            h.removeClient(client)

        case message := <-h.broadcast:
            // Publish to Redis (all instances receive it)
            h.redis.Publish(context.Background(), h.channel, message)
        }
    }
}
```

## Summary

- **Handlers** are functions or structs implementing `http.Handler`
- **Middleware** wraps handlers for cross-cutting concerns
- **Routing** uses `http.ServeMux` (Go 1.22+) or router libraries
- **Validation** uses struct tags or manual validation
- **Response helpers** provide consistent JSON responses
- **Sessions** use libraries like `gorilla/sessions` or JWT
- **HTTP clients** use `http.Client` with custom transport for connection pooling
- **Concurrent requests** leverage goroutines for parallel API calls
- **Gin/Echo** provide Symfony-like convenience when needed
- **WebSockets** enable real-time communication with gorilla/websocket
- **Hub pattern** manages broadcasting to multiple clients

---

## Exercises

1. **Full CRUD API**: Build a complete REST API for a resource using only `net/http`. Include all HTTP methods.

2. **Middleware Chain**: Implement logging, recovery (panic handling), and request ID middleware. Chain them correctly.

3. **Authentication Flow**: Implement login/logout with JWT tokens. Store user info in context.

4. **Validation Layer**: Create a validation system for request bodies. Handle validation errors with proper HTTP responses.

5. **Response Writer Wrapper**: Create a `ResponseWriter` wrapper that captures the status code for logging middleware.

6. **Route Groups**: Implement route grouping with shared middleware (e.g., `/api/v1/users` with auth middleware).

7. **Error Handling**: Design an error type that carries HTTP status codes. Use it throughout handlers.

8. **Graceful Shutdown**: Implement graceful shutdown that waits for active requests to complete.

9. **WebSocket Echo**: Build a WebSocket echo server that returns messages to the sender.

10. **Chat Room**: Implement a multi-user chat room with user join/leave notifications.

11. **Presence System**: Add "user is typing" indicators to the chat room using WebSocket messages.

12. **Reconnection**: Implement client-side reconnection with exponential backoff.

13. **API Client**: Build a reusable HTTP client for a public API (e.g., GitHub, JSONPlaceholder). Include proper error handling.

14. **Concurrent Fetcher**: Fetch data from multiple URLs concurrently with a configurable concurrency limit.

15. **Retry Middleware**: Create HTTP client middleware that retries failed requests with exponential backoff.

16. **Request Logging**: Build an `http.RoundTripper` that logs all outgoing requests and responses.
