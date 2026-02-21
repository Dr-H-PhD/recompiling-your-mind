# Chapter 14: Configuration and Environment

Symfony's configuration system is comprehensive: YAML files, environment variables, parameters, service bindings. Go's approach is simpler but requires more explicit code.

## No `.env` Magic: Explicit Configuration

Symfony Dotenv loads `.env` files automatically:

```bash
# .env
DATABASE_URL=mysql://user:pass@localhost/db
MAILER_DSN=smtp://localhost
APP_SECRET=abc123
```

```php
// Automatically available
$_ENV['DATABASE_URL'];
$this->getParameter('database_url');
```

Go reads environment variables directly:

```go
import "os"

func main() {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Fatal("DATABASE_URL is required")
    }

    // Or with default
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
}
```

### Loading `.env` Files

Use `godotenv` if you want `.env` file loading:

```go
import "github.com/joho/godotenv"

func main() {
    // Load .env file (optional in production)
    godotenv.Load()

    dbURL := os.Getenv("DATABASE_URL")
}
```

But many Go developers skip `.env` files entirely, preferring:
- Environment variables set by the deployment platform
- Configuration files (YAML, JSON, TOML)
- Command-line flags

## Viper vs symfony/dotenv

Viper is Go's most comprehensive configuration library:

```go
import "github.com/spf13/viper"

func loadConfig() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("/etc/myapp/")

    // Environment variables override file values
    viper.AutomaticEnv()
    viper.SetEnvPrefix("MYAPP")

    // Defaults
    viper.SetDefault("server.port", 8080)
    viper.SetDefault("server.timeout", "30s")

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, err
        }
        // Config file not found; use defaults and env vars
    }

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }

    return &config, nil
}
```

### Configuration Struct

```go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
}

type ServerConfig struct {
    Port    int           `mapstructure:"port"`
    Timeout time.Duration `mapstructure:"timeout"`
}

type DatabaseConfig struct {
    URL             string `mapstructure:"url"`
    MaxConnections  int    `mapstructure:"max_connections"`
}
```

### Config File

```yaml
# config.yaml
server:
  port: 8080
  timeout: 30s

database:
  url: postgres://localhost/myapp
  max_connections: 25
```

### Environment Variable Override

```bash
MYAPP_SERVER_PORT=9000 ./myapp
# Uses 9000 instead of 8080
```

## Feature Flags Patterns

Symfony might use a feature flag bundle. Go uses simple configuration:

```go
type FeatureFlags struct {
    NewCheckout    bool `mapstructure:"new_checkout"`
    BetaDashboard  bool `mapstructure:"beta_dashboard"`
    ExperimentalAPI bool `mapstructure:"experimental_api"`
}

type Config struct {
    Features FeatureFlags `mapstructure:"features"`
}

// Usage
if config.Features.NewCheckout {
    return newCheckoutHandler(w, r)
}
return legacyCheckoutHandler(w, r)
```

### More Sophisticated Feature Flags

For percentage rollouts or user targeting:

```go
type FeatureFlag struct {
    Enabled    bool     `mapstructure:"enabled"`
    Percentage int      `mapstructure:"percentage"`
    Users      []string `mapstructure:"users"`
}

func (f FeatureFlag) IsEnabledFor(userID string) bool {
    if !f.Enabled {
        return false
    }

    // Specific users
    for _, u := range f.Users {
        if u == userID {
            return true
        }
    }

    // Percentage rollout
    if f.Percentage > 0 {
        hash := hashUserID(userID)
        return hash%100 < f.Percentage
    }

    return f.Enabled && len(f.Users) == 0
}
```

## 12-Factor App Principles in Go

The 12-factor methodology is natural in Go:

### III. Config: Store config in environment

```go
type Config struct {
    DatabaseURL string
    RedisURL    string
    Port        int
}

func LoadFromEnv() Config {
    return Config{
        DatabaseURL: mustEnv("DATABASE_URL"),
        RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
        Port:        getEnvInt("PORT", 8080),
    }
}

func mustEnv(key string) string {
    val := os.Getenv(key)
    if val == "" {
        log.Fatalf("%s is required", key)
    }
    return val
}

func getEnv(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}
```

### VI. Processes: Execute as stateless processes

Go applications are naturally stateless—no session state in memory:

```go
// Bad: State in memory
var sessionStore = make(map[string]Session)

// Good: External state store
type SessionStore interface {
    Get(id string) (*Session, error)
    Set(id string, session *Session) error
}

func NewRedisSessionStore(client *redis.Client) SessionStore {
    // ...
}
```

### XI. Logs: Treat logs as event streams

```go
import "log/slog"

func main() {
    // Log to stdout as JSON
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    slog.Info("server starting", "port", port)
}
```

No log files—let the platform capture stdout.

## Secret Management

Symfony might use secrets with `symfony/secrets`:

```bash
php bin/console secrets:set DATABASE_PASSWORD
```

Go approaches vary:

### Environment Variables

Simple but limited:

```bash
DATABASE_PASSWORD=secret ./myapp
```

### Secret Files

Mount secrets as files:

```go
func loadSecret(path string) (string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(data)), nil
}

// Usage
dbPassword, err := loadSecret("/run/secrets/db_password")
```

### Secret Managers

For AWS Secrets Manager, HashiCorp Vault, etc.:

```go
import "github.com/aws/aws-sdk-go-v2/service/secretsmanager"

func loadFromSecretsManager(ctx context.Context, name string) (string, error) {
    client := secretsmanager.NewFromConfig(cfg)
    result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: &name,
    })
    if err != nil {
        return "", err
    }
    return *result.SecretString, nil
}
```

## No Symfony parameters.yaml

Symfony's parameters:

```yaml
# config/services.yaml
parameters:
    mailer.sender: 'noreply@example.com'

services:
    App\Mailer:
        arguments:
            $sender: '%mailer.sender%'
```

Go uses explicit wiring:

```go
type Config struct {
    Mailer MailerConfig `mapstructure:"mailer"`
}

type MailerConfig struct {
    Sender string `mapstructure:"sender"`
}

// Wiring
func main() {
    config := loadConfig()
    mailer := NewMailer(config.Mailer.Sender)
}
```

### Configuration Validation

Validate at startup:

```go
func (c *Config) Validate() error {
    if c.Database.URL == "" {
        return errors.New("database.url is required")
    }
    if c.Server.Port < 1 || c.Server.Port > 65535 {
        return errors.New("server.port must be between 1 and 65535")
    }
    if c.Server.Timeout <= 0 {
        return errors.New("server.timeout must be positive")
    }
    return nil
}

func main() {
    config, err := loadConfig()
    if err != nil {
        log.Fatal(err)
    }
    if err := config.Validate(); err != nil {
        log.Fatal("invalid configuration: ", err)
    }
}
```

## Summary

- **Environment variables** are read directly with `os.Getenv`
- **Viper** provides file-based config with environment override
- **Feature flags** are configuration values, not framework features
- **12-factor principles** align naturally with Go
- **Secret management** via environment, files, or secret managers
- **Explicit wiring** replaces Symfony's parameter injection

---

## Exercises

1. **Config Struct Design**: Design a configuration struct for a web application with database, cache, and HTTP server settings.

2. **Environment Loading**: Write a config loader that reads from environment variables with required vs optional handling.

3. **Viper Setup**: Set up Viper with a YAML config file, environment variable override, and defaults.

4. **Feature Flag System**: Implement a feature flag system with percentage rollout and user targeting.

5. **Secret Rotation**: Design a system that can reload secrets without restarting the application.

6. **Configuration Validation**: Add comprehensive validation to a config struct. Test with invalid configurations.

7. **Multi-Environment Config**: Support different configurations for development, staging, and production.

8. **Command-Line Flags**: Add command-line flag support using `flag` package or `cobra`. Override config file values.
