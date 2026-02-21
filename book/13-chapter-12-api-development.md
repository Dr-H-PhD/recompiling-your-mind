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

## gRPC: High-Performance APIs

PHP developers typically use REST. gRPC offers binary serialisation, code generation, and streaming—popular for microservices communication.

### Why gRPC?

| Feature | REST/JSON | gRPC |
|---------|-----------|------|
| Serialisation | JSON (text) | Protocol Buffers (binary) |
| Contract | OpenAPI (optional) | .proto files (required) |
| Streaming | Limited | Native bidirectional |
| Code generation | Optional | Built-in |
| Performance | Good | Excellent |
| Browser support | Native | Via grpc-web |

Use gRPC for service-to-service communication. Use REST for public APIs and browser clients.

### Protocol Buffers

Define your service contract in `.proto` files:

```protobuf
// user.proto
syntax = "proto3";

package user;

option go_package = "myapp/pb";

message User {
    int64 id = 1;
    string name = 2;
    string email = 3;
    google.protobuf.Timestamp created_at = 4;
}

message GetUserRequest {
    int64 id = 1;
}

message CreateUserRequest {
    string name = 1;
    string email = 2;
}

message ListUsersRequest {
    int32 page_size = 1;
    string page_token = 2;
}

message ListUsersResponse {
    repeated User users = 1;
    string next_page_token = 2;
}

service UserService {
    rpc GetUser(GetUserRequest) returns (User);
    rpc CreateUser(CreateUserRequest) returns (User);
    rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
    rpc WatchUsers(ListUsersRequest) returns (stream User);  // Server streaming
}
```

Generate Go code:

```bash
protoc --go_out=. --go-grpc_out=. user.proto
```

### Implementing a gRPC Server

```go
import (
    "google.golang.org/grpc"
    pb "myapp/pb"
)

type userServer struct {
    pb.UnimplementedUserServiceServer
    repo UserRepository
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    user, err := s.repo.Find(ctx, req.Id)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "user not found")
    }
    return toProtoUser(user), nil
}

func (s *userServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
    user := &User{
        Name:  req.Name,
        Email: req.Email,
    }

    if err := s.repo.Create(ctx, user); err != nil {
        return nil, status.Errorf(codes.Internal, "failed to create user")
    }

    return toProtoUser(user), nil
}

// Server streaming
func (s *userServer) WatchUsers(req *pb.ListUsersRequest, stream pb.UserService_WatchUsersServer) error {
    users, err := s.repo.List(stream.Context())
    if err != nil {
        return status.Errorf(codes.Internal, "failed to list users")
    }

    for _, user := range users {
        if err := stream.Send(toProtoUser(user)); err != nil {
            return err
        }
    }
    return nil
}

func main() {
    lis, _ := net.Listen("tcp", ":50051")
    grpcServer := grpc.NewServer()
    pb.RegisterUserServiceServer(grpcServer, &userServer{})
    grpcServer.Serve(lis)
}
```

### gRPC Client

```go
func main() {
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    // Unary call
    user, err := client.GetUser(context.Background(), &pb.GetUserRequest{Id: 1})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %v\n", user)

    // Streaming call
    stream, err := client.WatchUsers(context.Background(), &pb.ListUsersRequest{})
    if err != nil {
        log.Fatal(err)
    }

    for {
        user, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("Received: %v\n", user)
    }
}
```

### gRPC Middleware (Interceptors)

```go
func loggingInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    start := time.Now()
    resp, err := handler(ctx, req)
    log.Printf("method=%s duration=%v err=%v", info.FullMethod, time.Since(start), err)
    return resp, err
}

func authInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
    }

    tokens := md.Get("authorization")
    if len(tokens) == 0 {
        return nil, status.Errorf(codes.Unauthenticated, "missing token")
    }

    // Validate token...
    return handler(ctx, req)
}

// Apply interceptors
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(loggingInterceptor, authInterceptor),
)
```

### Client-Side Streaming

The client sends multiple messages; the server responds once after receiving all:

```protobuf
// average.proto
service AverageService {
    rpc ComputeAverage(stream Number) returns (AverageResponse);
}

message Number {
    int32 value = 1;
}

message AverageResponse {
    double average = 1;
}
```

Server implementation:

```go
func (s *averageServer) ComputeAverage(stream pb.AverageService_ComputeAverageServer) error {
    var sum, count int32

    for {
        num, err := stream.Recv()
        if err == io.EOF {
            // Client finished sending
            average := float64(sum) / float64(count)
            return stream.SendAndClose(&pb.AverageResponse{Average: average})
        }
        if err != nil {
            return err
        }
        sum += num.Value
        count++
    }
}
```

Client implementation:

```go
func sendNumbers(client pb.AverageServiceClient, numbers []int32) (*pb.AverageResponse, error) {
    stream, err := client.ComputeAverage(context.Background())
    if err != nil {
        return nil, err
    }

    for _, num := range numbers {
        if err := stream.Send(&pb.Number{Value: num}); err != nil {
            return nil, err
        }
    }

    // Close stream and get response
    return stream.CloseAndRecv()
}
```

### Bi-Directional Streaming

Both client and server send streams simultaneously—ideal for chat, gaming, or real-time collaboration:

```protobuf
// chat.proto
service ChatService {
    rpc Chat(stream ChatMessage) returns (stream ChatMessage);
}

message ChatMessage {
    string user = 1;
    string content = 2;
    int64 timestamp = 3;
}
```

Server implementation:

```go
func (s *chatServer) Chat(stream pb.ChatService_ChatServer) error {
    for {
        msg, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }

        // Broadcast to all clients (simplified)
        response := &pb.ChatMessage{
            User:      msg.User,
            Content:   msg.Content,
            Timestamp: time.Now().Unix(),
        }

        if err := stream.Send(response); err != nil {
            return err
        }
    }
}
```

Client with concurrent send/receive:

```go
func chat(client pb.ChatServiceClient) error {
    stream, err := client.Chat(context.Background())
    if err != nil {
        return err
    }

    // Receive messages in goroutine
    go func() {
        for {
            msg, err := stream.Recv()
            if err == io.EOF {
                return
            }
            if err != nil {
                log.Printf("receive error: %v", err)
                return
            }
            fmt.Printf("[%s]: %s\n", msg.User, msg.Content)
        }
    }()

    // Send messages from stdin
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        if err := stream.Send(&pb.ChatMessage{
            User:    "me",
            Content: scanner.Text(),
        }); err != nil {
            return err
        }
    }

    return stream.CloseSend()
}
```

### Deadlines and Timeouts

PHP relies on `max_execution_time`. gRPC uses context deadlines for precise timeout control:

```go
// Client: set deadline
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

user, err := client.GetUser(ctx, &pb.GetUserRequest{Id: 1})
if err != nil {
    if status.Code(err) == codes.DeadlineExceeded {
        log.Println("request timed out")
    }
    return err
}
```

Server: respect the deadline:

```go
func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    // Check if deadline already exceeded
    if ctx.Err() == context.DeadlineExceeded {
        return nil, status.Error(codes.DeadlineExceeded, "deadline exceeded")
    }

    // Pass context to downstream calls
    user, err := s.repo.Find(ctx, req.Id)
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return nil, status.Error(codes.DeadlineExceeded, "deadline exceeded")
        }
        return nil, status.Error(codes.Internal, "database error")
    }

    return toProtoUser(user), nil
}
```

### Retry Patterns

Implement retries with exponential backoff for transient failures:

```go
func withRetry[T any](
    ctx context.Context,
    fn func() (T, error),
    maxRetries int,
    initialBackoff time.Duration,
) (T, error) {
    var result T
    var lastErr error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        result, lastErr = fn()
        if lastErr == nil {
            return result, nil
        }

        // Only retry on transient errors
        if !isRetryable(lastErr) {
            return result, lastErr
        }

        if attempt < maxRetries {
            backoff := initialBackoff * time.Duration(1<<attempt)
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return result, ctx.Err()
            }
        }
    }

    return result, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetryable(err error) bool {
    code := status.Code(err)
    return code == codes.Unavailable ||
           code == codes.DeadlineExceeded ||
           code == codes.ResourceExhausted
}

// Usage
user, err := withRetry(ctx, func() (*pb.User, error) {
    return client.GetUser(ctx, &pb.GetUserRequest{Id: 1})
}, 3, 100*time.Millisecond)
```

### Circuit Breaker Pattern

Prevent cascading failures by failing fast when a service is unhealthy:

```go
import "github.com/sony/gobreaker"

type UserClient struct {
    client pb.UserServiceClient
    cb     *gobreaker.CircuitBreaker
}

func NewUserClient(conn *grpc.ClientConn) *UserClient {
    settings := gobreaker.Settings{
        Name:        "user-service",
        MaxRequests: 5,                // Requests allowed in half-open state
        Interval:    10 * time.Second, // Reset counts after interval
        Timeout:     30 * time.Second, // Time in open state before half-open
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            // Open circuit if failure rate > 50%
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return counts.Requests >= 10 && failureRatio >= 0.5
        },
    }

    return &UserClient{
        client: pb.NewUserServiceClient(conn),
        cb:     gobreaker.NewCircuitBreaker(settings),
    }
}

func (c *UserClient) GetUser(ctx context.Context, id int64) (*pb.User, error) {
    result, err := c.cb.Execute(func() (interface{}, error) {
        return c.client.GetUser(ctx, &pb.GetUserRequest{Id: id})
    })

    if err != nil {
        if err == gobreaker.ErrOpenState {
            return nil, status.Error(codes.Unavailable, "service temporarily unavailable")
        }
        return nil, err
    }

    return result.(*pb.User), nil
}
```

### TLS/mTLS Security

Production gRPC requires TLS encryption. Generate certificates:

```bash
# Generate CA
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 365 -key ca.key -out ca.crt -subj "/CN=MyCA"

# Generate server certificate
openssl genrsa -out server.key 4096
openssl req -new -key server.key -out server.csr -subj "/CN=localhost"
openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out server.crt

# For mTLS: generate client certificate
openssl genrsa -out client.key 4096
openssl req -new -key client.key -out client.csr -subj "/CN=client"
openssl x509 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out client.crt
```

Server with TLS:

```go
func createTLSServer() (*grpc.Server, error) {
    cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
    if err != nil {
        return nil, err
    }

    creds := credentials.NewTLS(&tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientAuth:   tls.NoClientCert, // TLS only
    })

    return grpc.NewServer(grpc.Creds(creds)), nil
}
```

Server with mTLS (mutual authentication):

```go
func createMTLSServer() (*grpc.Server, error) {
    cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
    if err != nil {
        return nil, err
    }

    // Load CA cert to verify clients
    caCert, err := os.ReadFile("ca.crt")
    if err != nil {
        return nil, err
    }
    caPool := x509.NewCertPool()
    caPool.AppendCertsFromPEM(caCert)

    creds := credentials.NewTLS(&tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientAuth:   tls.RequireAndVerifyClientCert,
        ClientCAs:    caPool,
    })

    return grpc.NewServer(grpc.Creds(creds)), nil
}
```

Client with TLS:

```go
func createTLSClient(serverAddr string) (*grpc.ClientConn, error) {
    caCert, err := os.ReadFile("ca.crt")
    if err != nil {
        return nil, err
    }
    caPool := x509.NewCertPool()
    caPool.AppendCertsFromPEM(caCert)

    creds := credentials.NewTLS(&tls.Config{
        RootCAs: caPool,
    })

    return grpc.Dial(serverAddr, grpc.WithTransportCredentials(creds))
}
```

Client with mTLS:

```go
func createMTLSClient(serverAddr string) (*grpc.ClientConn, error) {
    cert, err := tls.LoadX509KeyPair("client.crt", "client.key")
    if err != nil {
        return nil, err
    }

    caCert, err := os.ReadFile("ca.crt")
    if err != nil {
        return nil, err
    }
    caPool := x509.NewCertPool()
    caPool.AppendCertsFromPEM(caCert)

    creds := credentials.NewTLS(&tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caPool,
    })

    return grpc.Dial(serverAddr, grpc.WithTransportCredentials(creds))
}
```

### Health Checks

gRPC has a standard health checking protocol. Implement it for load balancers and Kubernetes:

```go
import "google.golang.org/grpc/health"
import healthpb "google.golang.org/grpc/health/grpc_health_v1"

func main() {
    server := grpc.NewServer()

    // Register your services
    pb.RegisterUserServiceServer(server, &userServer{})

    // Register health service
    healthServer := health.NewServer()
    healthpb.RegisterHealthServer(server, healthServer)

    // Set service status
    healthServer.SetServingStatus("user.UserService", healthpb.HealthCheckResponse_SERVING)

    // Update status based on dependencies
    go func() {
        for {
            if dbHealthy() {
                healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
            } else {
                healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
            }
            time.Sleep(10 * time.Second)
        }
    }()

    lis, _ := net.Listen("tcp", ":50051")
    server.Serve(lis)
}
```

Check health from client or CLI:

```bash
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

## GraphQL: Flexible Queries

GraphQL lets clients request exactly the data they need—no over-fetching or under-fetching. PHP has webonyx/graphql-php or API Platform's GraphQL support.

### Why GraphQL?

| Use Case | Best Choice |
|----------|-------------|
| Fixed data requirements | REST |
| Variable data requirements | GraphQL |
| Simple CRUD | REST |
| Complex nested data | GraphQL |
| Microservices internal | gRPC |
| Mobile apps (bandwidth) | GraphQL |

### gqlgen: Go's GraphQL Library

gqlgen generates type-safe Go code from your GraphQL schema.

Define your schema:

```graphql
# schema.graphql
type User {
    id: ID!
    name: String!
    email: String!
    posts: [Post!]!
    createdAt: Time!
}

type Post {
    id: ID!
    title: String!
    content: String!
    author: User!
}

type Query {
    user(id: ID!): User
    users(limit: Int = 10, offset: Int = 0): [User!]!
    post(id: ID!): Post
}

type Mutation {
    createUser(input: CreateUserInput!): User!
    updateUser(id: ID!, input: UpdateUserInput!): User!
    deleteUser(id: ID!): Boolean!
}

input CreateUserInput {
    name: String!
    email: String!
}

input UpdateUserInput {
    name: String
    email: String
}

scalar Time
```

Generate code:

```bash
go run github.com/99designs/gqlgen generate
```

### Implementing Resolvers

```go
type Resolver struct {
    userRepo UserRepository
    postRepo PostRepository
}

// Query resolvers
func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) {
    return r.userRepo.FindByID(ctx, id)
}

func (r *queryResolver) Users(ctx context.Context, limit *int, offset *int) ([]*model.User, error) {
    l, o := 10, 0
    if limit != nil {
        l = *limit
    }
    if offset != nil {
        o = *offset
    }
    return r.userRepo.List(ctx, l, o)
}

// Mutation resolvers
func (r *mutationResolver) CreateUser(ctx context.Context, input model.CreateUserInput) (*model.User, error) {
    user := &model.User{
        Name:  input.Name,
        Email: input.Email,
    }
    if err := r.userRepo.Create(ctx, user); err != nil {
        return nil, err
    }
    return user, nil
}

// Field resolvers (N+1 prevention with dataloaders)
func (r *userResolver) Posts(ctx context.Context, obj *model.User) ([]*model.Post, error) {
    return r.postRepo.FindByAuthorID(ctx, obj.ID)
}
```

### DataLoaders for N+1 Prevention

GraphQL's nested queries can cause N+1 problems. DataLoaders batch requests:

```go
import "github.com/graph-gophers/dataloader/v7"

type Loaders struct {
    PostsByAuthor *dataloader.Loader[string, []*model.Post]
}

func NewLoaders(postRepo PostRepository) *Loaders {
    return &Loaders{
        PostsByAuthor: dataloader.NewBatchedLoader(func(ctx context.Context, authorIDs []string) []*dataloader.Result[[]*model.Post] {
            // Batch fetch all posts for all authors at once
            postsByAuthor, err := postRepo.FindByAuthorIDs(ctx, authorIDs)

            results := make([]*dataloader.Result[[]*model.Post], len(authorIDs))
            for i, id := range authorIDs {
                if err != nil {
                    results[i] = &dataloader.Result[[]*model.Post]{Error: err}
                } else {
                    results[i] = &dataloader.Result[[]*model.Post]{Data: postsByAuthor[id]}
                }
            }
            return results
        }),
    }
}

// Use in resolver
func (r *userResolver) Posts(ctx context.Context, obj *model.User) ([]*model.Post, error) {
    loaders := ctx.Value(loadersKey).(*Loaders)
    return loaders.PostsByAuthor.Load(ctx, obj.ID)()
}
```

### GraphQL Middleware

```go
func main() {
    srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
        Resolvers: &Resolver{},
    }))

    // Add complexity limit
    srv.Use(extension.FixedComplexityLimit(100))

    // Add logging
    srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
        op := graphql.GetOperationContext(ctx)
        log.Printf("GraphQL operation: %s", op.OperationName)
        return next(ctx)
    })

    // Add authentication
    srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
        user := auth.UserFromContext(ctx)
        if user == nil {
            return graphql.ErrorResponse(ctx, "unauthorized")
        }
        return next(ctx)
    })

    http.Handle("/graphql", srv)
    http.Handle("/playground", playground.Handler("GraphQL", "/graphql"))
    http.ListenAndServe(":8080", nil)
}
```

### Choosing Between REST, gRPC, and GraphQL

| Criterion | REST | gRPC | GraphQL |
|-----------|------|------|---------|
| Client control over data | Low | Low | High |
| Performance | Good | Excellent | Good |
| Learning curve | Low | Medium | Medium |
| Tooling maturity | Excellent | Good | Good |
| Browser support | Native | Limited | Native |
| Caching | Easy (HTTP) | Manual | Complex |
| Best for | Public APIs | Microservices | Mobile/frontend |

For PHP developers: REST is familiar territory. Use gRPC for internal services where performance matters. Use GraphQL when clients need flexible data fetching.

## Summary

- **JSON encoding** uses struct tags for field mapping
- **Separate structs** for input, output, and domain models
- **OpenAPI** via swag (generate from code) or oapi-codegen (generate from spec)
- **Authentication** is middleware that populates context
- **Validation** uses `go-playground/validator` with struct tags
- **Error responses** follow consistent structure
- **Versioning** is implemented manually (URL or header)
- **gRPC** provides high-performance binary communication with streaming
- **Four streaming patterns**: unary, server-streaming, client-streaming, bi-directional
- **Resilience**: deadlines, retries with backoff, circuit breakers
- **TLS/mTLS** secures gRPC in production
- **Health checks** enable load balancer and Kubernetes integration
- **GraphQL** enables flexible queries with client-controlled data fetching
- **Choose REST** for public APIs, **gRPC** for internal services, **GraphQL** for complex frontends

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

9. **gRPC Service**: Define a proto file for a simple service. Generate code and implement server and client.

10. **gRPC Streaming**: Add a server-streaming endpoint to your gRPC service. Test with a client that processes the stream.

11. **GraphQL API**: Set up gqlgen for a simple schema. Implement query and mutation resolvers.

12. **DataLoader Implementation**: Add dataloaders to prevent N+1 queries in nested GraphQL resolvers.

13. **Client-Side Streaming**: Implement a file upload service using client-side streaming. The client streams file chunks; the server responds with upload status.

14. **Bi-Directional Streaming**: Build a simple chat service where multiple clients can exchange messages in real-time.

15. **Deadline Propagation**: Create a service chain (A → B → C) where deadlines propagate correctly through all services.

16. **Circuit Breaker**: Implement a gRPC client with circuit breaker that fails fast when the downstream service is unhealthy.

17. **mTLS Setup**: Configure mutual TLS authentication between a gRPC client and server. Test that unauthenticated clients are rejected.

18. **Health Check Service**: Add the standard gRPC health check to a service. Test with `grpcurl` and configure Kubernetes readiness probe.
