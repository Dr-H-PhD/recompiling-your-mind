# Chapter 26: Security

PHP developers rely on Symfony Security, CSRF protection, and the security-checker. Go has no built-in security framework—you build security into your application explicitly. This chapter covers essential security practices for Go applications.

## OWASP Top 10 in Go

The OWASP Top 10 represents the most critical web application security risks. Let's address each in Go.

### A01: Broken Access Control

PHP/Symfony uses voters and access control lists:

```php
#[IsGranted('ROLE_ADMIN')]
public function deleteUser(User $user): Response
{
    // Only admins can delete
}

// Voter
public function supports(string $attribute, mixed $subject): bool
{
    return $subject instanceof Post && $attribute === 'EDIT';
}
```

Go implements access control in middleware and handlers:

```go
// Role-based middleware
func requireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := auth.UserFromContext(r.Context())
            if user == nil || !user.HasRole(role) {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Resource-based authorisation (like Symfony voters)
type Authoriser interface {
    CanEdit(ctx context.Context, user *User, resource any) bool
    CanDelete(ctx context.Context, user *User, resource any) bool
}

type PostAuthoriser struct{}

func (a *PostAuthoriser) CanEdit(ctx context.Context, user *User, resource any) bool {
    post, ok := resource.(*Post)
    if !ok {
        return false
    }
    // Owner or admin can edit
    return post.AuthorID == user.ID || user.HasRole("admin")
}

// Handler with authorisation
func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
    user := auth.UserFromContext(r.Context())
    post, _ := h.repo.Find(r.Context(), postID)

    if !h.authoriser.CanEdit(r.Context(), user, post) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    // Proceed with update
}
```

### A02: Cryptographic Failures

Use Go's `crypto` package correctly:

```go
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "golang.org/x/crypto/bcrypt"
)

// Password hashing (use bcrypt, not SHA)
func hashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(hash), err
}

func verifyPassword(hashedPassword, password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
    return err == nil
}

// Encryption (AES-GCM)
func encrypt(plaintext []byte, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return nil, err
    }

    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonceSize := gcm.NonceSize()
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

    return gcm.Open(nil, nonce, ciphertext, nil)
}

// Secure random token generation
func generateToken(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}
```

### A03: Injection

Go's `database/sql` uses parameterised queries by default:

```go
// SAFE: Parameterised query
row := db.QueryRow("SELECT * FROM users WHERE email = $1", email)

// DANGEROUS: String concatenation
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email) // SQL INJECTION!
row := db.QueryRow(query) // Don't do this!

// For dynamic queries, use a query builder
import sq "github.com/Masterminds/squirrel"

query, args, _ := sq.Select("*").
    From("users").
    Where(sq.Eq{"email": email}).
    PlaceholderFormat(sq.Dollar).
    ToSql()
row := db.QueryRow(query, args...)
```

Command injection:

```go
import "os/exec"

// DANGEROUS: Shell injection
cmd := exec.Command("sh", "-c", "grep "+userInput+" /var/log/app.log") // INJECTION!

// SAFE: Pass arguments separately
cmd := exec.Command("grep", userInput, "/var/log/app.log")
```

### A04: Insecure Design

Design security in from the start:

```go
// Defence in depth: validate at multiple layers
type CreateOrderRequest struct {
    ProductID string  `json:"product_id" validate:"required,uuid"`
    Quantity  int     `json:"quantity" validate:"required,min=1,max=100"`
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    var req CreateOrderRequest

    // Layer 1: Input validation
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid JSON")
        return
    }

    if err := validate.Struct(req); err != nil {
        writeError(w, http.StatusBadRequest, "validation failed")
        return
    }

    // Layer 2: Business logic validation
    product, err := h.productRepo.Find(r.Context(), req.ProductID)
    if err != nil {
        writeError(w, http.StatusNotFound, "product not found")
        return
    }

    if product.Stock < req.Quantity {
        writeError(w, http.StatusConflict, "insufficient stock")
        return
    }

    // Layer 3: Authorisation
    user := auth.UserFromContext(r.Context())
    if !user.CanPurchase() {
        writeError(w, http.StatusForbidden, "account suspended")
        return
    }

    // Proceed with order...
}
```

### A05: Security Misconfiguration

Secure defaults and explicit configuration:

```go
// Secure HTTP server configuration
server := &http.Server{
    Addr:              ":8443",
    Handler:           handler,
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      10 * time.Second,
    IdleTimeout:       120 * time.Second,
    ReadHeaderTimeout: 5 * time.Second,
    MaxHeaderBytes:    1 << 20, // 1 MB

    TLSConfig: &tls.Config{
        MinVersion:               tls.VersionTLS12,
        PreferServerCipherSuites: true,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
    },
}

// Security headers middleware
func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        next.ServeHTTP(w, r)
    })
}
```

### A06: Vulnerable Components

Track dependencies with `go mod`:

```bash
# Check for known vulnerabilities
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Update dependencies
go get -u ./...
go mod tidy
```

### A07: Authentication Failures

Implement secure authentication:

```go
// Rate limiting for login attempts
type LoginRateLimiter struct {
    attempts map[string][]time.Time
    mu       sync.Mutex
    maxAttempts int
    window      time.Duration
}

func (l *LoginRateLimiter) Allow(identifier string) bool {
    l.mu.Lock()
    defer l.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-l.window)

    // Clean old attempts
    var recent []time.Time
    for _, t := range l.attempts[identifier] {
        if t.After(cutoff) {
            recent = append(recent, t)
        }
    }

    if len(recent) >= l.maxAttempts {
        return false
    }

    l.attempts[identifier] = append(recent, now)
    return true
}

// Secure session handling
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    var creds Credentials
    json.NewDecoder(r.Body).Decode(&creds)

    // Rate limit by IP
    ip := realIP(r)
    if !h.rateLimiter.Allow(ip) {
        http.Error(w, "Too many attempts", http.StatusTooManyRequests)
        return
    }

    user, err := h.userRepo.FindByEmail(r.Context(), creds.Email)
    if err != nil {
        // Timing-safe response (don't reveal if user exists)
        time.Sleep(100 * time.Millisecond)
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    if !verifyPassword(user.PasswordHash, creds.Password) {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    // Generate secure session
    sessionID, _ := generateToken(32)
    h.sessionStore.Create(sessionID, user.ID, 24*time.Hour)

    http.SetCookie(w, &http.Cookie{
        Name:     "session",
        Value:    sessionID,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        Path:     "/",
        MaxAge:   86400,
    })
}
```

### A08: Software and Data Integrity

Verify data integrity:

```go
import (
    "crypto/hmac"
    "crypto/sha256"
)

// Sign data
func sign(data []byte, secret []byte) []byte {
    h := hmac.New(sha256.New, secret)
    h.Write(data)
    return h.Sum(nil)
}

// Verify signature
func verify(data, signature, secret []byte) bool {
    expected := sign(data, secret)
    return hmac.Equal(signature, expected)
}

// Signed cookies
func setSignedCookie(w http.ResponseWriter, name, value string, secret []byte) {
    signature := sign([]byte(value), secret)
    signedValue := base64.StdEncoding.EncodeToString(signature) + "." + value

    http.SetCookie(w, &http.Cookie{
        Name:     name,
        Value:    signedValue,
        HttpOnly: true,
        Secure:   true,
    })
}

func getSignedCookie(r *http.Request, name string, secret []byte) (string, error) {
    cookie, err := r.Cookie(name)
    if err != nil {
        return "", err
    }

    parts := strings.SplitN(cookie.Value, ".", 2)
    if len(parts) != 2 {
        return "", errors.New("invalid cookie format")
    }

    signature, _ := base64.StdEncoding.DecodeString(parts[0])
    value := parts[1]

    if !verify([]byte(value), signature, secret) {
        return "", errors.New("invalid signature")
    }

    return value, nil
}
```

### A09: Security Logging and Monitoring

Log security events:

```go
import "log/slog"

// Security event logger
type SecurityLogger struct {
    logger *slog.Logger
}

func (l *SecurityLogger) LogAuthFailure(ctx context.Context, email, ip, reason string) {
    l.logger.WarnContext(ctx, "authentication failure",
        "event", "auth_failure",
        "email", email,
        "ip", ip,
        "reason", reason,
    )
}

func (l *SecurityLogger) LogAuthSuccess(ctx context.Context, userID, ip string) {
    l.logger.InfoContext(ctx, "authentication success",
        "event", "auth_success",
        "user_id", userID,
        "ip", ip,
    )
}

func (l *SecurityLogger) LogAccessDenied(ctx context.Context, userID, resource, action string) {
    l.logger.WarnContext(ctx, "access denied",
        "event", "access_denied",
        "user_id", userID,
        "resource", resource,
        "action", action,
    )
}

func (l *SecurityLogger) LogSuspiciousActivity(ctx context.Context, details map[string]any) {
    l.logger.ErrorContext(ctx, "suspicious activity detected",
        "event", "suspicious_activity",
        "details", details,
    )
}
```

### A10: Server-Side Request Forgery (SSRF)

Validate URLs and restrict outbound requests:

```go
import (
    "net"
    "net/url"
)

// SSRF protection
func isAllowedURL(rawURL string) error {
    parsed, err := url.Parse(rawURL)
    if err != nil {
        return err
    }

    // Only allow HTTPS
    if parsed.Scheme != "https" {
        return errors.New("only HTTPS allowed")
    }

    // Resolve hostname
    ips, err := net.LookupIP(parsed.Hostname())
    if err != nil {
        return err
    }

    for _, ip := range ips {
        // Block private/internal IPs
        if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
            return errors.New("internal addresses not allowed")
        }
    }

    // Allowlist of domains (optional)
    allowedDomains := []string{"api.example.com", "cdn.example.com"}
    allowed := false
    for _, domain := range allowedDomains {
        if parsed.Hostname() == domain {
            allowed = true
            break
        }
    }
    if !allowed {
        return errors.New("domain not in allowlist")
    }

    return nil
}

func fetchURL(rawURL string) ([]byte, error) {
    if err := isAllowedURL(rawURL); err != nil {
        return nil, fmt.Errorf("URL validation failed: %w", err)
    }

    resp, err := http.Get(rawURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    return io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // Limit response size
}
```

## TLS Configuration

### Server TLS

```go
func loadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, err
    }

    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
        CurvePreferences: []tls.CurveID{
            tls.X25519,
            tls.CurveP256,
        },
    }, nil
}

func main() {
    tlsConfig, err := loadTLSConfig("cert.pem", "key.pem")
    if err != nil {
        log.Fatal(err)
    }

    server := &http.Server{
        Addr:      ":443",
        Handler:   handler,
        TLSConfig: tlsConfig,
    }

    log.Fatal(server.ListenAndServeTLS("", ""))
}
```

### Client TLS

```go
func createSecureClient(caCertFile string) (*http.Client, error) {
    caCert, err := os.ReadFile(caCertFile)
    if err != nil {
        return nil, err
    }

    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)

    return &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                RootCAs:    caCertPool,
                MinVersion: tls.VersionTLS12,
            },
        },
        Timeout: 30 * time.Second,
    }, nil
}

// Mutual TLS (mTLS)
func createMTLSClient(caCert, clientCert, clientKey string) (*http.Client, error) {
    cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
    if err != nil {
        return nil, err
    }

    caCertPEM, _ := os.ReadFile(caCert)
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCertPEM)

    return &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                Certificates: []tls.Certificate{cert},
                RootCAs:      caCertPool,
                MinVersion:   tls.VersionTLS12,
            },
        },
    }, nil
}
```

## Secrets Management

Symfony uses environment variables and the secrets vault. Go applications need explicit secrets handling.

### Environment Variables

```go
// Load secrets from environment
type Config struct {
    DatabaseURL   string
    JWTSecret     []byte
    EncryptionKey []byte
}

func LoadConfig() (*Config, error) {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        return nil, errors.New("DATABASE_URL required")
    }

    jwtSecret := os.Getenv("JWT_SECRET")
    if len(jwtSecret) < 32 {
        return nil, errors.New("JWT_SECRET must be at least 32 characters")
    }

    encKey := os.Getenv("ENCRYPTION_KEY")
    keyBytes, err := base64.StdEncoding.DecodeString(encKey)
    if err != nil || len(keyBytes) != 32 {
        return nil, errors.New("ENCRYPTION_KEY must be 32 bytes base64 encoded")
    }

    return &Config{
        DatabaseURL:   dbURL,
        JWTSecret:     []byte(jwtSecret),
        EncryptionKey: keyBytes,
    }, nil
}
```

### HashiCorp Vault Integration

```go
import vault "github.com/hashicorp/vault/api"

type SecretStore struct {
    client *vault.Client
}

func NewSecretStore(addr, token string) (*SecretStore, error) {
    config := vault.DefaultConfig()
    config.Address = addr

    client, err := vault.NewClient(config)
    if err != nil {
        return nil, err
    }

    client.SetToken(token)

    return &SecretStore{client: client}, nil
}

func (s *SecretStore) GetSecret(path string) (map[string]interface{}, error) {
    secret, err := s.client.Logical().Read(path)
    if err != nil {
        return nil, err
    }
    if secret == nil {
        return nil, errors.New("secret not found")
    }
    return secret.Data["data"].(map[string]interface{}), nil
}

// Usage
func main() {
    store, _ := NewSecretStore("https://vault.example.com", os.Getenv("VAULT_TOKEN"))

    secrets, _ := store.GetSecret("secret/data/myapp/database")
    dbPassword := secrets["password"].(string)
}
```

### AWS Secrets Manager

```go
import (
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func getAWSSecret(ctx context.Context, secretName string) (string, error) {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return "", err
    }

    client := secretsmanager.NewFromConfig(cfg)

    result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: &secretName,
    })
    if err != nil {
        return "", err
    }

    return *result.SecretString, nil
}
```

### Secret Rotation

```go
type RotatingSecret struct {
    mu         sync.RWMutex
    value      []byte
    lastRotate time.Time
    ttl        time.Duration
    fetch      func() ([]byte, error)
}

func (s *RotatingSecret) Get() ([]byte, error) {
    s.mu.RLock()
    if time.Since(s.lastRotate) < s.ttl {
        value := s.value
        s.mu.RUnlock()
        return value, nil
    }
    s.mu.RUnlock()

    s.mu.Lock()
    defer s.mu.Unlock()

    // Double-check after acquiring write lock
    if time.Since(s.lastRotate) < s.ttl {
        return s.value, nil
    }

    value, err := s.fetch()
    if err != nil {
        return nil, err
    }

    s.value = value
    s.lastRotate = time.Now()
    return value, nil
}
```

## CORS Configuration

```go
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
    originSet := make(map[string]bool)
    for _, o := range allowedOrigins {
        originSet[o] = true
    }

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")

            if originSet[origin] {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
                w.Header().Set("Access-Control-Allow-Credentials", "true")
                w.Header().Set("Access-Control-Max-Age", "86400")
            }

            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusNoContent)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

## Summary

- **Access control** is implemented via middleware and authorisation checks
- **Cryptography** uses Go's `crypto` package with bcrypt for passwords
- **Injection prevention** relies on parameterised queries and argument separation
- **Security headers** are added via middleware
- **Rate limiting** protects against brute force attacks
- **TLS** should use version 1.2+ with strong cipher suites
- **Secrets** should be loaded from environment or secret managers
- **CORS** requires explicit configuration for cross-origin requests
- **govulncheck** identifies vulnerable dependencies

---

## Exercises

1. **Password Hashing**: Implement secure password hashing with bcrypt. Add timing-safe comparison.

2. **JWT Middleware**: Create JWT authentication middleware with proper error handling and claim validation.

3. **Rate Limiter**: Build a sliding window rate limiter for API endpoints. Test with concurrent requests.

4. **Security Headers**: Create a middleware that adds all recommended security headers. Verify with security scanners.

5. **SSRF Protection**: Implement URL validation that blocks internal addresses and only allows specific domains.

6. **Secret Rotation**: Build a secret manager that automatically rotates credentials from Vault.

7. **Audit Logging**: Implement comprehensive security event logging with correlation IDs.

8. **mTLS Server**: Set up a server with mutual TLS that requires client certificates.
