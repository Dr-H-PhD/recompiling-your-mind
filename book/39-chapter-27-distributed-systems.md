# Chapter 27: Distributed Systems

PHP applications typically run as monoliths behind a load balancer. Go developers often build distributed systems—microservices that must coordinate, handle failures, and maintain consistency. This chapter covers the fundamentals.

## Why Distributed Systems?

PHP's request-response model is simple: a request arrives, PHP handles it, and the process ends. No state persists between requests. Scaling means adding more PHP-FPM workers.

Go applications often:
- Run as long-lived processes
- Maintain connections to multiple services
- Handle concurrent requests across services
- Need to coordinate state across nodes

This introduces new challenges that PHP developers haven't faced.

## The CAP Theorem

The CAP theorem states that a distributed system can provide at most two of these three guarantees:

- **Consistency**: Every read receives the most recent write
- **Availability**: Every request receives a response
- **Partition Tolerance**: The system continues operating despite network failures

Since network partitions are inevitable, you must choose between consistency and availability during partitions.

### CP Systems (Consistency + Partition Tolerance)

```go
// Example: Distributed lock with strong consistency
type DistributedLock struct {
    client *etcd.Client
}

func (l *DistributedLock) Acquire(ctx context.Context, key string, ttl int64) (*Lock, error) {
    lease, err := l.client.Grant(ctx, ttl)
    if err != nil {
        return nil, err
    }

    // Compare-and-swap ensures only one holder
    txn := l.client.Txn(ctx).
        If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
        Then(clientv3.OpPut(key, "", clientv3.WithLease(lease.ID)))

    resp, err := txn.Commit()
    if err != nil {
        return nil, err
    }

    if !resp.Succeeded {
        return nil, errors.New("lock already held")
    }

    return &Lock{key: key, lease: lease.ID, client: l.client}, nil
}
```

CP systems (etcd, ZooKeeper, Consul) sacrifice availability—they may reject requests during partitions to maintain consistency.

### AP Systems (Availability + Partition Tolerance)

```go
// Example: Eventually consistent cache
type DistributedCache struct {
    local     map[string]Value
    peers     []string
    mu        sync.RWMutex
}

func (c *DistributedCache) Set(key string, value Value) {
    c.mu.Lock()
    c.local[key] = value
    c.mu.Unlock()

    // Asynchronously replicate to peers (eventually consistent)
    go func() {
        for _, peer := range c.peers {
            c.replicateToPeer(peer, key, value)
        }
    }()
}

func (c *DistributedCache) Get(key string) (Value, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    v, ok := c.local[key]
    return v, ok  // May return stale data
}
```

AP systems (Cassandra, DynamoDB, DNS) remain available but may return stale data during partitions.

### Choosing Consistency Models

| Use Case | Model | Example |
|----------|-------|---------|
| Financial transactions | Strong consistency | Bank transfers |
| User sessions | Eventual consistency | Shopping cart |
| Configuration | Strong consistency | Feature flags |
| Analytics | Eventual consistency | Page views |
| Inventory | Strong consistency | Stock levels |
| User profiles | Eventual consistency | Display names |

## Service Discovery

PHP applications use static configuration. Distributed systems need dynamic service discovery.

### Consul Integration

```go
import (
    "github.com/hashicorp/consul/api"
)

type ServiceRegistry struct {
    client *api.Client
}

func (r *ServiceRegistry) Register(name, address string, port int, healthCheck string) error {
    registration := &api.AgentServiceRegistration{
        ID:      fmt.Sprintf("%s-%s-%d", name, address, port),
        Name:    name,
        Address: address,
        Port:    port,
        Check: &api.AgentServiceCheck{
            HTTP:     healthCheck,
            Interval: "10s",
            Timeout:  "5s",
        },
    }

    return r.client.Agent().ServiceRegister(registration)
}

func (r *ServiceRegistry) Discover(name string) ([]*api.ServiceEntry, error) {
    services, _, err := r.client.Health().Service(name, "", true, nil)
    return services, err
}

// Client with service discovery
type ServiceClient struct {
    registry *ServiceRegistry
    service  string
    client   *http.Client
}

func (c *ServiceClient) Call(ctx context.Context, path string) (*http.Response, error) {
    services, err := c.registry.Discover(c.service)
    if err != nil {
        return nil, err
    }

    if len(services) == 0 {
        return nil, errors.New("no healthy instances")
    }

    // Simple round-robin (production should use better load balancing)
    instance := services[rand.Intn(len(services))]
    url := fmt.Sprintf("http://%s:%d%s", instance.Service.Address, instance.Service.Port, path)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    return c.client.Do(req)
}
```

### DNS-Based Discovery

```go
import "net"

func discoverService(serviceName string) ([]string, error) {
    // SRV records for service discovery
    _, addrs, err := net.LookupSRV("", "", serviceName)
    if err != nil {
        return nil, err
    }

    var endpoints []string
    for _, addr := range addrs {
        endpoints = append(endpoints, fmt.Sprintf("%s:%d", addr.Target, addr.Port))
    }
    return endpoints, nil
}
```

## Circuit Breakers

Prevent cascade failures when services are unhealthy.

```go
type CircuitBreaker struct {
    mu           sync.Mutex
    state        State
    failures     int
    successes    int
    lastFailure  time.Time
    threshold    int
    timeout      time.Duration
    halfOpenMax  int
}

type State int

const (
    StateClosed State = iota
    StateOpen
    StateHalfOpen
)

func (cb *CircuitBreaker) Execute(fn func() error) error {
    cb.mu.Lock()

    switch cb.state {
    case StateOpen:
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = StateHalfOpen
            cb.successes = 0
        } else {
            cb.mu.Unlock()
            return errors.New("circuit breaker is open")
        }
    }

    cb.mu.Unlock()

    err := fn()

    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()

        if cb.state == StateHalfOpen || cb.failures >= cb.threshold {
            cb.state = StateOpen
        }
        return err
    }

    if cb.state == StateHalfOpen {
        cb.successes++
        if cb.successes >= cb.halfOpenMax {
            cb.state = StateClosed
            cb.failures = 0
        }
    } else {
        cb.failures = 0
    }

    return nil
}

// Usage
func (c *Client) CallWithCircuitBreaker(ctx context.Context) error {
    return c.breaker.Execute(func() error {
        return c.doRequest(ctx)
    })
}
```

### Using gobreaker

```go
import "github.com/sony/gobreaker"

func newCircuitBreaker(name string) *gobreaker.CircuitBreaker {
    return gobreaker.NewCircuitBreaker(gobreaker.Settings{
        Name:        name,
        MaxRequests: 5,                    // Requests in half-open
        Interval:    60 * time.Second,     // Reset interval in closed
        Timeout:     30 * time.Second,     // Time in open before half-open
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures > 5
        },
        OnStateChange: func(name string, from, to gobreaker.State) {
            log.Printf("Circuit breaker %s: %s -> %s", name, from, to)
        },
    })
}

func (c *Client) Call(ctx context.Context) (interface{}, error) {
    result, err := c.cb.Execute(func() (interface{}, error) {
        return c.doRequest(ctx)
    })
    return result, err
}
```

## Retries with Backoff

```go
type RetryConfig struct {
    MaxRetries  int
    InitialWait time.Duration
    MaxWait     time.Duration
    Multiplier  float64
}

func WithRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
    var lastErr error
    wait := cfg.InitialWait

    for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }

        lastErr = err

        // Don't retry non-retryable errors
        if !isRetryable(err) {
            return err
        }

        if attempt == cfg.MaxRetries {
            break
        }

        // Wait with jitter
        jitter := time.Duration(rand.Int63n(int64(wait) / 2))
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(wait + jitter):
        }

        // Exponential backoff
        wait = time.Duration(float64(wait) * cfg.Multiplier)
        if wait > cfg.MaxWait {
            wait = cfg.MaxWait
        }
    }

    return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetryable(err error) bool {
    // Network errors, 5xx responses, etc.
    var netErr net.Error
    if errors.As(err, &netErr) {
        return netErr.Temporary()
    }

    var httpErr *HTTPError
    if errors.As(err, &httpErr) {
        return httpErr.StatusCode >= 500
    }

    return false
}
```

## Distributed Transactions

PHP's single-database transactions don't work across services. Use patterns like Saga.

### Saga Pattern

```go
type Step struct {
    Name       string
    Execute    func(ctx context.Context) error
    Compensate func(ctx context.Context) error
}

type Saga struct {
    steps     []Step
    completed []int
}

func (s *Saga) Run(ctx context.Context) error {
    for i, step := range s.steps {
        if err := step.Execute(ctx); err != nil {
            // Compensate completed steps in reverse order
            for j := len(s.completed) - 1; j >= 0; j-- {
                idx := s.completed[j]
                if compErr := s.steps[idx].Compensate(ctx); compErr != nil {
                    log.Printf("Compensation failed for %s: %v", s.steps[idx].Name, compErr)
                }
            }
            return fmt.Errorf("step %s failed: %w", step.Name, err)
        }
        s.completed = append(s.completed, i)
    }
    return nil
}

// Usage: Order creation saga
func createOrderSaga(order *Order) *Saga {
    return &Saga{
        steps: []Step{
            {
                Name: "reserve_inventory",
                Execute: func(ctx context.Context) error {
                    return inventoryService.Reserve(ctx, order.Items)
                },
                Compensate: func(ctx context.Context) error {
                    return inventoryService.Release(ctx, order.Items)
                },
            },
            {
                Name: "charge_payment",
                Execute: func(ctx context.Context) error {
                    return paymentService.Charge(ctx, order.UserID, order.Total)
                },
                Compensate: func(ctx context.Context) error {
                    return paymentService.Refund(ctx, order.UserID, order.Total)
                },
            },
            {
                Name: "create_shipment",
                Execute: func(ctx context.Context) error {
                    return shippingService.CreateShipment(ctx, order)
                },
                Compensate: func(ctx context.Context) error {
                    return shippingService.CancelShipment(ctx, order.ID)
                },
            },
        },
    }
}
```

### Outbox Pattern

Ensure message delivery with database transactions:

```go
type Outbox struct {
    db *sql.DB
}

func (o *Outbox) SaveWithEvents(ctx context.Context, entity any, events []Event) error {
    tx, err := o.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Save entity
    if err := saveEntity(tx, entity); err != nil {
        return err
    }

    // Save events to outbox table
    for _, event := range events {
        data, _ := json.Marshal(event)
        _, err := tx.ExecContext(ctx, `
            INSERT INTO outbox (event_type, payload, created_at)
            VALUES ($1, $2, $3)
        `, event.Type(), data, time.Now())
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}

// Background worker publishes outbox events
func (o *Outbox) ProcessOutbox(ctx context.Context, publisher EventPublisher) error {
    rows, err := o.db.QueryContext(ctx, `
        SELECT id, event_type, payload
        FROM outbox
        WHERE published_at IS NULL
        ORDER BY created_at
        LIMIT 100
    `)
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var id int64
        var eventType string
        var payload []byte
        rows.Scan(&id, &eventType, &payload)

        if err := publisher.Publish(ctx, eventType, payload); err != nil {
            continue  // Retry later
        }

        o.db.ExecContext(ctx, `
            UPDATE outbox SET published_at = $1 WHERE id = $2
        `, time.Now(), id)
    }

    return nil
}
```

## Leader Election

Coordinate a single leader across nodes:

```go
import (
    clientv3 "go.etcd.io/etcd/client/v3"
    "go.etcd.io/etcd/client/v3/concurrency"
)

type LeaderElection struct {
    client   *clientv3.Client
    session  *concurrency.Session
    election *concurrency.Election
    nodeID   string
}

func NewLeaderElection(client *clientv3.Client, prefix, nodeID string) (*LeaderElection, error) {
    session, err := concurrency.NewSession(client, concurrency.WithTTL(10))
    if err != nil {
        return nil, err
    }

    election := concurrency.NewElection(session, prefix)

    return &LeaderElection{
        client:   client,
        session:  session,
        election: election,
        nodeID:   nodeID,
    }, nil
}

func (le *LeaderElection) Campaign(ctx context.Context) error {
    return le.election.Campaign(ctx, le.nodeID)
}

func (le *LeaderElection) Resign(ctx context.Context) error {
    return le.election.Resign(ctx)
}

func (le *LeaderElection) IsLeader(ctx context.Context) bool {
    resp, err := le.election.Leader(ctx)
    if err != nil {
        return false
    }
    return string(resp.Kvs[0].Value) == le.nodeID
}

// Usage
func runWorker(ctx context.Context, le *LeaderElection) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
        }

        log.Println("Campaigning for leadership...")
        if err := le.Campaign(ctx); err != nil {
            log.Printf("Campaign failed: %v", err)
            time.Sleep(5 * time.Second)
            continue
        }

        log.Println("Became leader, starting work...")
        doLeaderWork(ctx)

        le.Resign(ctx)
    }
}
```

## Health Checks

```go
type HealthChecker struct {
    checks map[string]func(context.Context) error
    mu     sync.RWMutex
}

func (h *HealthChecker) Register(name string, check func(context.Context) error) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.checks[name] = check
}

type HealthStatus struct {
    Status string            `json:"status"`
    Checks map[string]string `json:"checks"`
}

func (h *HealthChecker) Check(ctx context.Context) HealthStatus {
    h.mu.RLock()
    defer h.mu.RUnlock()

    status := HealthStatus{
        Status: "healthy",
        Checks: make(map[string]string),
    }

    for name, check := range h.checks {
        if err := check(ctx); err != nil {
            status.Status = "unhealthy"
            status.Checks[name] = err.Error()
        } else {
            status.Checks[name] = "ok"
        }
    }

    return status
}

// Common checks
func DatabaseCheck(db *sql.DB) func(context.Context) error {
    return func(ctx context.Context) error {
        return db.PingContext(ctx)
    }
}

func RedisCheck(rdb *redis.Client) func(context.Context) error {
    return func(ctx context.Context) error {
        return rdb.Ping(ctx).Err()
    }
}

func DependencyCheck(url string) func(context.Context) error {
    return func(ctx context.Context) error {
        req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return err
        }
        resp.Body.Close()
        if resp.StatusCode >= 400 {
            return fmt.Errorf("unhealthy: status %d", resp.StatusCode)
        }
        return nil
    }
}

// Handler
func (h *HealthChecker) Handler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()

        status := h.Check(ctx)

        w.Header().Set("Content-Type", "application/json")
        if status.Status != "healthy" {
            w.WriteHeader(http.StatusServiceUnavailable)
        }
        json.NewEncoder(w).Encode(status)
    })
}
```

## Summary

- **CAP theorem**: Choose between consistency and availability during partitions
- **Service discovery**: Use Consul, etcd, or DNS for dynamic endpoint lookup
- **Circuit breakers**: Prevent cascade failures when dependencies are unhealthy
- **Retries**: Use exponential backoff with jitter for transient failures
- **Sagas**: Coordinate distributed transactions with compensating actions
- **Outbox pattern**: Ensure reliable message delivery with database transactions
- **Leader election**: Coordinate single-leader work using consensus systems
- **Health checks**: Monitor dependency health for load balancer integration

---

## Exercises

1. **Circuit Breaker**: Implement a circuit breaker with closed, open, and half-open states. Test with a flaky service.

2. **Service Discovery**: Set up Consul and implement service registration and discovery in a Go application.

3. **Retry Logic**: Build a retry mechanism with exponential backoff and jitter. Handle non-retryable errors.

4. **Saga Implementation**: Implement the order creation saga with proper compensation on failure.

5. **Outbox Pattern**: Add an outbox table and background worker to reliably publish events.

6. **Leader Election**: Use etcd to implement leader election. Verify only one node acts as leader.

7. **Health Aggregation**: Create a health checker that aggregates checks from multiple dependencies.

8. **Distributed Lock**: Implement a distributed lock using Redis or etcd. Handle lock expiration and renewal.
