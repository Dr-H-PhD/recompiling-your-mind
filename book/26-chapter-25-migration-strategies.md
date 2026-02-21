# Chapter 25: Migration Strategies

Migrating from PHP to Go isn't typically a big-bang rewrite. This chapter covers practical strategies for gradual migration, drawing on patterns used successfully in production.

## Strangler Fig Pattern

The strangler fig tree grows around its host, eventually replacing it. Apply this to your PHP application:

1. **New features in Go**: Build new functionality in Go
2. **Route requests**: Proxy to PHP or Go based on path
3. **Migrate incrementally**: Move existing features one by one
4. **Remove PHP**: When all features migrated, retire PHP

### Implementation

```
                    ┌─────────────────┐
    Request ───────►│   Load Balancer │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
    ┌─────────────────┐         ┌─────────────────┐
    │   Go Service    │         │   PHP Service   │
    │  (new features) │         │  (legacy code)  │
    └─────────────────┘         └─────────────────┘
              │                             │
              └──────────────┬──────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │    Database     │
                    └─────────────────┘
```

### Routing at the Load Balancer

nginx:
```nginx
upstream go_service {
    server go-app:8080;
}

upstream php_service {
    server php-fpm:9000;
}

server {
    # New API endpoints → Go
    location /api/v2/ {
        proxy_pass http://go_service;
    }

    # Legacy endpoints → PHP
    location / {
        fastcgi_pass php_service;
    }
}
```

## Running PHP and Go Side-by-Side

### Shared Authentication

Both services need to validate the same sessions/tokens:

```go
// Go: Validate PHP session
func validatePHPSession(sessionID string) (*User, error) {
    // Option 1: Shared Redis session store
    data, err := redis.Get("PHPREDIS_SESSION:" + sessionID).Result()

    // Option 2: Call PHP validation endpoint
    resp, err := http.Get(phpURL + "/api/validate-session?id=" + sessionID)

    // Option 3: Shared JWT with same secret
    claims, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
        return []byte(sharedSecret), nil
    })
}
```

### Session Sharing via Redis

PHP:
```php
// php.ini
session.save_handler = redis
session.save_path = "tcp://redis:6379"
```

Go:
```go
import "github.com/go-redis/redis/v8"

func getSessionUser(sessionID string) (*User, error) {
    data, err := redisClient.Get(ctx, "PHPREDIS_SESSION:"+sessionID).Bytes()
    if err != nil {
        return nil, err
    }

    // PHP serialises sessions—use a PHP deserialiser or standardise on JSON
    var session map[string]interface{}
    // Decode PHP serialisation or JSON
    return extractUser(session)
}
```

### JWT Sharing

PHP:
```php
use Firebase\JWT\JWT;

$token = JWT::encode([
    'user_id' => $user->getId(),
    'exp' => time() + 3600,
], getenv('JWT_SECRET'), 'HS256');
```

Go:
```go
import "github.com/golang-jwt/jwt/v5"

func validateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        return []byte(os.Getenv("JWT_SECRET")), nil
    })
    if err != nil {
        return nil, err
    }
    return token.Claims.(*Claims), nil
}
```

## API Gateway Approaches

A dedicated API gateway can route between PHP and Go:

```
Client → API Gateway → PHP Service
                     → Go Service
                     → Other Services
```

### Using Kong or Similar

```yaml
# Kong declarative config
services:
  - name: go-users
    url: http://go-service:8080
    routes:
      - paths: ["/api/v2/users"]

  - name: php-legacy
    url: http://php-service
    routes:
      - paths: ["/api/v1/", "/legacy/"]
```

### Go as the Gateway

Go can serve as the gateway itself:

```go
func main() {
    goHandler := newGoHandler()
    phpProxy := newReverseProxy("http://php-service")

    mux := http.NewServeMux()

    // Go handles new API
    mux.Handle("/api/v2/", goHandler)

    // Proxy everything else to PHP
    mux.Handle("/", phpProxy)

    http.ListenAndServe(":8080", mux)
}

func newReverseProxy(target string) *httputil.ReverseProxy {
    url, _ := url.Parse(target)
    return httputil.NewSingleHostReverseProxy(url)
}
```

## Database Sharing Strategies

Both services typically share a database during migration.

### Shared Read, Separate Write

- PHP writes to its tables
- Go writes to its tables
- Both read from all tables
- Clear ownership prevents conflicts

### Event-Driven Sync

```
PHP writes → Database → Triggers → Events → Go consumes
Go writes → Database → Triggers → Events → PHP consumes
```

Using Debezium or similar for change data capture:

```go
// Go: Consume database changes
func consumeChanges(ch <-chan ChangeEvent) {
    for event := range ch {
        switch event.Table {
        case "users":
            syncUser(event)
        case "orders":
            syncOrder(event)
        }
    }
}
```

### Eventual Consistency

Accept that data might be briefly inconsistent:

```go
// Go service caches PHP data
func getUser(id int) (*User, error) {
    // Try cache first
    if user, ok := cache.Get(id); ok {
        return user, nil
    }

    // Call PHP API for authoritative data
    user, err := phpClient.GetUser(id)
    if err != nil {
        return nil, err
    }

    // Cache for next time
    cache.Set(id, user, 5*time.Minute)
    return user, nil
}
```

## Gradual Team Transition

Technical migration is only half the challenge. Team transition matters equally.

### Training Path

1. **Go basics**: Syntax, types, control flow (1 week)
2. **Go idioms**: Error handling, interfaces, packages (2 weeks)
3. **Concurrency**: Goroutines, channels, patterns (2 weeks)
4. **Production code**: Review and contribute to Go services
5. **Lead features**: Own a feature from design to deployment

### Pairing and Review

- Pair PHP developers with Go-experienced developers
- Review all Go code from PHP developers carefully
- Discuss idiomatic alternatives, not just correctness

### Start Small

First Go services should be:
- Low-risk (non-critical path)
- Well-defined scope
- Good learning opportunities
- Not time-sensitive

## Case Study: Migrating a Symfony Application

Consider a typical Symfony app:
- REST API for mobile apps
- Admin dashboard (Twig templates)
- Background workers (Messenger)
- Doctrine ORM entities

### Migration Plan

**Phase 1: New API Endpoints (Month 1-2)**
- Build `/api/v2/` in Go
- Share authentication via JWT
- Both services use same database
- Load balancer routes by path

**Phase 2: Migrate High-Traffic Endpoints (Month 3-4)**
- Identify top 20% of endpoints by traffic
- Rewrite in Go
- Run shadow traffic to verify
- Cut over one at a time

**Phase 3: Background Workers (Month 5)**
- Build Go workers consuming same queues
- Run PHP and Go workers in parallel
- Gradually increase Go worker count
- Retire PHP workers

**Phase 4: Admin Dashboard (Month 6-8)**
- Build new admin in Go (or modern frontend)
- Or: Keep PHP for admin (acceptable for low-traffic)

**Phase 5: Retire PHP (Month 9+)**
- All traffic to Go
- Remove PHP infrastructure
- Archive PHP code

### Success Metrics

- **Latency**: P99 latency of Go vs PHP endpoints
- **Throughput**: Requests per second per container
- **Resource usage**: Memory and CPU per request
- **Error rate**: Compare error rates during transition
- **Developer productivity**: Time to ship features

## Summary

- **Strangler fig** enables gradual migration without big-bang rewrites
- **Side-by-side operation** requires shared auth and careful routing
- **API gateways** simplify routing between services
- **Database sharing** is common during transition
- **Team transition** requires training, pairing, and patience
- **Start small** with low-risk, well-scoped services

---

## Exercises

1. **Strangler Design**: Draw a migration architecture for a Symfony app with 10 controllers.

2. **Reverse Proxy**: Implement a Go reverse proxy that routes to PHP for legacy paths.

3. **JWT Sharing**: Create matching JWT generation in PHP and validation in Go.

4. **Session Sharing**: Set up Redis session sharing between PHP and Go.

5. **Database Migration**: Design a strategy for migrating a Doctrine entity to Go without downtime.

6. **Traffic Shadowing**: Implement shadow traffic to compare PHP and Go responses.

7. **Migration Checklist**: Create a checklist for migrating a single endpoint from PHP to Go.

8. **Team Training Plan**: Design a 3-month training plan for transitioning PHP developers to Go.
