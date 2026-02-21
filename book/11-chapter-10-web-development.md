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

## Summary

- **Handlers** are functions or structs implementing `http.Handler`
- **Middleware** wraps handlers for cross-cutting concerns
- **Routing** uses `http.ServeMux` (Go 1.22+) or router libraries
- **Validation** uses struct tags or manual validation
- **Response helpers** provide consistent JSON responses
- **Sessions** use libraries like `gorilla/sessions` or JWT
- **Gin/Echo** provide Symfony-like convenience when needed

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
