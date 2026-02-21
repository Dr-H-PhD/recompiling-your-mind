# Chapter 12: API Development

Symfony's API Platform or FOSRestBundle provide complete API solutions. Go developers typically build APIs from smaller pieces. This chapter covers the patterns.

## JSON APIs: Encoding/Decoding Patterns

Symfony Serializer handles complex cases:

```php
$user = $serializer->deserialize($json, User::class, 'json');
$json = $serializer->serialize($user, 'json', ['groups' => ['public']]);
```

Go uses `encoding/json`:

```go
// Decode
var user User
if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
    http.Error(w, "invalid JSON", http.StatusBadRequest)
    return
}

// Encode
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(user)
```

### Struct Tags Control Serialisation

```go
type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Password  string    `json:"-"`              // Never serialised
    CreatedAt time.Time `json:"created_at"`
    DeletedAt *time.Time `json:"deleted_at,omitempty"` // Omit if nil
}
```

### Different Input/Output Structs

Unlike PHP where you might use serialisation groups, Go often uses separate structs:

```go
// Input (what clients send)
type CreateUserInput struct {
    Name     string `json:"name" validate:"required"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

// Output (what API returns)
type UserResponse struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// Domain model (internal)
type User struct {
    ID           int
    Name         string
    Email        string
    PasswordHash string
    CreatedAt    time.Time
}

// Conversion
func (u User) ToResponse() UserResponse {
    return UserResponse{
        ID:        u.ID,
        Name:      u.Name,
        Email:     u.Email,
        CreatedAt: u.CreatedAt,
    }
}
```

### Custom Marshalling

For complex serialisation, implement `json.Marshaler`:

```go
type Money struct {
    Amount   int64  // Stored in cents
    Currency string
}

func (m Money) MarshalJSON() ([]byte, error) {
    return json.Marshal(map[string]interface{}{
        "amount":   float64(m.Amount) / 100,
        "currency": m.Currency,
    })
}

func (m *Money) UnmarshalJSON(data []byte) error {
    var raw struct {
        Amount   float64 `json:"amount"`
        Currency string  `json:"currency"`
    }
    if err := json.Unmarshal(data, &raw); err != nil {
        return err
    }
    m.Amount = int64(raw.Amount * 100)
    m.Currency = raw.Currency
    return nil
}
```

## OpenAPI/Swagger Integration

Symfony has NelmioApiDocBundle for OpenAPI generation. Go has several options:

### swag (Generate from Comments)

```go
// @Summary Create a new user
// @Description Create a user with the input payload
// @Tags users
// @Accept json
// @Produce json
// @Param user body CreateUserInput true "User input"
// @Success 201 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Router /users [post]
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

Run `swag init` to generate OpenAPI spec.

### oapi-codegen (Generate from Spec)

Write OpenAPI spec first, generate Go code:

```yaml
# openapi.yaml
paths:
  /users:
    post:
      operationId: createUser
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserInput'
      responses:
        '201':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserResponse'
```

```bash
oapi-codegen -generate types,server openapi.yaml > api.gen.go
```

This generates types and server interfaces you implement.

## Authentication Middleware (vs Symfony Security)

Symfony Security provides:
- Firewalls
- Voters
- Guards
- User providers

Go uses middleware:

```go
func authMiddleware(tokenService TokenService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            if token == "" {
                writeError(w, http.StatusUnauthorized, "missing token")
                return
            }

            claims, err := tokenService.Validate(token)
            if err != nil {
                writeError(w, http.StatusUnauthorized, "invalid token")
                return
            }

            ctx := context.WithValue(r.Context(), userClaimsKey, claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func extractBearerToken(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return strings.TrimPrefix(auth, "Bearer ")
    }
    return ""
}
```

### Role-Based Access

```go
func requireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := getUserClaims(r.Context())
            if claims == nil {
                writeError(w, http.StatusUnauthorized, "not authenticated")
                return
            }

            if !claims.HasRole(role) {
                writeError(w, http.StatusForbidden, "insufficient permissions")
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// Usage
mux.Handle("DELETE /users/{id}", chain(handler,
    authMiddleware(tokenSvc),
    requireRole("admin"),
))
```

## Validation Patterns (vs Symfony Validator)

Symfony Validator uses annotations:

```php
class CreateUserInput
{
    #[Assert\NotBlank(message: "Name is required")]
    #[Assert\Length(min: 2, max: 100)]
    public string $name;

    #[Assert\Email]
    public string $email;
}
```

Go uses `go-playground/validator`:

```go
import "github.com/go-playground/validator/v10"

type CreateUserInput struct {
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
}

var validate = validator.New()

func validateInput(input any) map[string]string {
    err := validate.Struct(input)
    if err == nil {
        return nil
    }

    errors := make(map[string]string)
    for _, err := range err.(validator.ValidationErrors) {
        field := strings.ToLower(err.Field())
        errors[field] = formatValidationMessage(err)
    }
    return errors
}

func formatValidationMessage(err validator.FieldError) string {
    switch err.Tag() {
    case "required":
        return "This field is required"
    case "email":
        return "Must be a valid email address"
    case "min":
        return fmt.Sprintf("Must be at least %s characters", err.Param())
    default:
        return "Invalid value"
    }
}
```

### Custom Validation

```go
func init() {
    validate.RegisterValidation("username", func(fl validator.FieldLevel) bool {
        username := fl.Field().String()
        return regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(username)
    })
}

type User struct {
    Username string `validate:"required,username,min=3,max=20"`
}
```

## Error Response Standards

Symfony normalises errors via the Serializer. Build a consistent error format:

```go
type ErrorResponse struct {
    Error   string            `json:"error"`
    Code    string            `json:"code,omitempty"`
    Details map[string]string `json:"details,omitempty"`
}

func writeError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func writeValidationError(w http.ResponseWriter, errors map[string]string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusUnprocessableEntity)
    json.NewEncoder(w).Encode(ErrorResponse{
        Error:   "Validation failed",
        Code:    "VALIDATION_ERROR",
        Details: errors,
    })
}
```

### Error Types for HTTP

```go
type HTTPError struct {
    Status  int
    Message string
    Code    string
}

func (e HTTPError) Error() string {
    return e.Message
}

var (
    ErrNotFound     = HTTPError{Status: 404, Message: "Resource not found", Code: "NOT_FOUND"}
    ErrUnauthorized = HTTPError{Status: 401, Message: "Unauthorized", Code: "UNAUTHORIZED"}
)

// In handler
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
    user, err := h.repo.Find(ctx, id)
    if err != nil {
        handleError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, user)
}

func handleError(w http.ResponseWriter, err error) {
    var httpErr HTTPError
    if errors.As(err, &httpErr) {
        writeError(w, httpErr.Status, httpErr.Message)
        return
    }
    // Log unexpected errors
    slog.Error("unexpected error", "error", err)
    writeError(w, http.StatusInternalServerError, "Internal server error")
}
```

## Versioning Strategies

Symfony supports URL, header, and query parameter versioning. Go doesn't have built-in support—implement your preferred strategy:

### URL Versioning

```go
mux := http.NewServeMux()
mux.Handle("/api/v1/", http.StripPrefix("/api/v1", v1Router))
mux.Handle("/api/v2/", http.StripPrefix("/api/v2", v2Router))
```

### Header Versioning

```go
func versionMiddleware(v1, v2 http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        version := r.Header.Get("API-Version")
        switch version {
        case "2", "2.0":
            v2.ServeHTTP(w, r)
        default:
            v1.ServeHTTP(w, r)
        }
    })
}
```

## Summary

- **JSON encoding** uses struct tags for field mapping
- **Separate structs** for input, output, and domain models
- **OpenAPI** via swag (generate from code) or oapi-codegen (generate from spec)
- **Authentication** is middleware that populates context
- **Validation** uses `go-playground/validator` with struct tags
- **Error responses** follow consistent structure
- **Versioning** is implemented manually (URL or header)

---

## Exercises

1. **Complete API Resource**: Build a full REST API for a resource with create, read, update, delete, and list operations.

2. **Custom Validation**: Add three custom validation rules (e.g., strong password, valid slug, future date).

3. **OpenAPI Generation**: Set up swag for a small API. Generate documentation and verify it matches your handlers.

4. **Error Handling System**: Create an error handling system with different error types (validation, not found, unauthorized, internal).

5. **Pagination**: Implement cursor-based pagination for a list endpoint. Include pagination metadata in response.

6. **Rate Limiting**: Add rate limiting middleware using a token bucket or sliding window algorithm.

7. **Request ID Tracing**: Add request ID middleware. Include the ID in logs and error responses.

8. **API Versioning**: Implement URL-based versioning with two API versions that differ in response format.
