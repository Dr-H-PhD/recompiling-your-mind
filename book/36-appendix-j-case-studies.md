# Appendix J: Case Studies

Real-world case studies of PHP-to-Go migrations, including motivations, challenges, and outcomes.

---

## Case Study 1: E-Commerce API Migration

**Company Profile:** Mid-sized e-commerce platform
**Stack Before:** PHP 7.4, Symfony 4, MySQL, Redis
**Traffic:** 50,000 requests/minute peak

### The Problem

The product catalogue API was the bottleneck. During sales events:

- Response times exceeded 2 seconds
- PHP-FPM workers maxed out at 500
- Redis connection pools exhausted
- Horizontal scaling became expensive

```php
// Before: Symfony controller
class CatalogueController extends AbstractController
{
    public function search(Request $request): JsonResponse
    {
        $products = $this->productRepository->search(
            $request->get('q'),
            $request->get('filters', [])
        );

        // Each request = 1 PHP-FPM worker blocked
        // 500 workers = 500 concurrent requests max
        return $this->json($products);
    }
}
```

### The Solution

Migrated the catalogue API to Go while keeping the rest in PHP:

```go
// After: Go handler
func (h *CatalogueHandler) Search(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Single Go process handles thousands of concurrent requests
    // Each request is a goroutine, not an OS process
    products, err := h.repo.Search(ctx, r.URL.Query())
    if err != nil {
        writeError(w, err)
        return
    }

    writeJSON(w, products)
}
```

### Migration Approach

1. **Week 1-2:** Set up Go service with routing infrastructure
2. **Week 3-4:** Implement catalogue search endpoint
3. **Week 5:** Shadow traffic testing (both PHP and Go, compare results)
4. **Week 6:** Gradual traffic shift via nginx (10% → 50% → 100%)
5. **Week 7-8:** Monitor and optimise

### Results

| Metric | PHP | Go | Improvement |
|--------|-----|-----|-------------|
| p50 latency | 180ms | 12ms | 15x faster |
| p99 latency | 2.1s | 85ms | 25x faster |
| Memory per instance | 512MB | 64MB | 8x less |
| Concurrent connections | 500 | 50,000 | 100x more |
| Instances needed | 12 | 2 | 6x fewer |
| Monthly cloud cost | $4,800 | $400 | 12x cheaper |

### Lessons Learned

1. **Start with stateless APIs:** Easiest to migrate and test
2. **Shadow traffic is crucial:** Found edge cases before switching
3. **Keep shared database:** Don't migrate everything at once
4. **Connection pooling matters:** Go's `database/sql` handles this automatically

---

## Case Study 2: Background Job Processing

**Company Profile:** SaaS document processing platform
**Stack Before:** PHP 8.0, Laravel, Horizon (Redis queues)
**Volume:** 500,000 jobs/day

### The Problem

Document processing jobs (PDF generation, image conversion) were slow:

- Average job time: 45 seconds
- Workers frequently OOM'd
- Memory leaks in long-running PHP processes
- Supervisor constantly restarting workers

```php
// Before: Laravel job
class ProcessDocument implements ShouldQueue
{
    public function handle()
    {
        // Memory accumulates across jobs
        // PHP wasn't designed for long-running processes
        $document = Document::find($this->documentId);
        $document->process();
        $document->generatePdf();
        $document->notifyUser();
    }
}
```

### The Solution

Built a Go worker service that consumed from the same Redis queues:

```go
// After: Go worker
func (w *Worker) Process(ctx context.Context, job Job) error {
    // Each job is a goroutine - independent memory
    // No accumulation between jobs

    doc, err := w.repo.Find(ctx, job.DocumentID)
    if err != nil {
        return fmt.Errorf("find document: %w", err)
    }

    // Process concurrently where possible
    g, ctx := errgroup.WithContext(ctx)

    g.Go(func() error {
        return w.processor.Process(ctx, doc)
    })

    g.Go(func() error {
        return w.pdfGenerator.Generate(ctx, doc)
    })

    if err := g.Wait(); err != nil {
        return err
    }

    return w.notifier.Notify(ctx, doc.UserID)
}
```

### Results

| Metric | PHP (Laravel) | Go | Improvement |
|--------|---------------|-----|-------------|
| Avg job time | 45s | 8s | 5.6x faster |
| Jobs/hour/worker | 80 | 450 | 5.6x more |
| Memory per worker | 256MB | 32MB | 8x less |
| Workers needed | 24 | 4 | 6x fewer |
| Restart frequency | 10/hour | 0/week | Stable |

### Key Insight

Go's concurrency model let us parallelise within a single job. The PDF generation and image processing ran concurrently:

```go
// Process multiple pages concurrently
var wg sync.WaitGroup
sem := make(chan struct{}, 10) // Limit to 10 concurrent

for _, page := range doc.Pages {
    wg.Add(1)
    sem <- struct{}{}

    go func(p Page) {
        defer wg.Done()
        defer func() { <-sem }()
        processPage(p)
    }(page)
}

wg.Wait()
```

---

## Case Study 3: Real-Time Notifications

**Company Profile:** Social media analytics dashboard
**Stack Before:** PHP 7.4, Symfony, Mercure (SSE)
**Users:** 10,000 concurrent dashboard users

### The Problem

Real-time notifications via Server-Sent Events (SSE) were problematic:

- Each SSE connection held a PHP-FPM worker
- 10,000 users = 10,000 blocked workers
- Scaling to more workers was impractical
- Mercure hub added complexity

### The Solution

Implemented WebSocket server in Go:

```go
type NotificationHub struct {
    clients    map[*Client]bool
    broadcast  chan Message
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

func (h *NotificationHub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()

        case client := <-h.unregister:
            h.mu.Lock()
            delete(h.clients, client)
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
```

### Results

| Metric | PHP + Mercure | Go WebSockets | Improvement |
|--------|---------------|---------------|-------------|
| Concurrent connections | 500 | 50,000 | 100x more |
| Memory for 10k users | 5GB | 200MB | 25x less |
| Message latency | 200ms | 5ms | 40x faster |
| Infrastructure | Complex | Simple | - |

---

## Case Study 4: Microservices Decomposition

**Company Profile:** Fintech startup
**Stack Before:** PHP 8.1 monolith, Symfony 6
**Challenge:** Scale payment processing independently

### Migration Strategy

Used the Strangler Fig pattern over 6 months:

**Month 1-2: Infrastructure**
- Set up Go service template
- Implemented shared authentication (JWT)
- Created nginx routing rules

**Month 3-4: Payment Service**
```go
// New payment service in Go
type PaymentService struct {
    db        *sql.DB      // Same database as PHP
    stripe    *stripe.API
    publisher *amqp.Channel // RabbitMQ for events
}

func (s *PaymentService) Process(ctx context.Context, p Payment) error {
    // Process payment
    charge, err := s.stripe.Charge(ctx, p)
    if err != nil {
        return err
    }

    // Save to shared database
    if err := s.savePayment(ctx, p, charge); err != nil {
        return err
    }

    // Publish event for PHP to consume
    return s.publisher.Publish("payment.completed", PaymentEvent{
        ID:     p.ID,
        Status: "completed",
    })
}
```

**Month 5-6: Traffic Migration**
- Routed `/api/payments/*` to Go
- PHP consumed payment events via Symfony Messenger
- Gradually migrated related endpoints

### Architecture

```
                    ┌─────────────┐
                    │   nginx     │
                    └──────┬──────┘
                           │
            ┌──────────────┼──────────────┐
            │              │              │
            ▼              ▼              ▼
    ┌───────────────┐ ┌─────────┐ ┌───────────────┐
    │ Go Payment    │ │   PHP   │ │ Go Analytics  │
    │   Service     │ │ Monolith│ │   Service     │
    └───────┬───────┘ └────┬────┘ └───────────────┘
            │              │
            │   RabbitMQ   │
            └──────┬───────┘
                   │
            ┌──────┴──────┐
            │  PostgreSQL │
            └─────────────┘
```

### Results After 6 Months

| Service | Language | Requests/sec | Latency (p99) |
|---------|----------|--------------|---------------|
| Payments | Go | 5,000 | 45ms |
| Analytics | Go | 8,000 | 30ms |
| User/Auth | PHP | 1,200 | 180ms |
| Admin | PHP | 200 | 250ms |

**Key Takeaway:** Not everything needs to be Go. Keep low-traffic admin in PHP.

---

## Common Patterns Across Case Studies

### What Worked

1. **Start with bounded contexts:** APIs, workers, real-time features
2. **Share database initially:** Avoid distributed transaction complexity
3. **Use message queues:** Let PHP and Go communicate asynchronously
4. **Shadow traffic:** Test before switching
5. **Gradual rollout:** Use feature flags and percentage routing

### What Didn't Work

1. **Big bang rewrites:** Always failed or stalled
2. **Ignoring team skills:** Training takes time
3. **Premature optimisation:** Migrate for the right reasons
4. **Migrating everything:** Some PHP code is fine

### When to Migrate

✅ **Good candidates:**
- High-traffic APIs
- CPU-intensive background jobs
- Real-time features (WebSockets, SSE)
- Services needing concurrency

❌ **Keep in PHP:**
- Admin dashboards
- Content management
- Complex business logic that works
- Teams without Go experience

---

*These case studies are composites based on real migration experiences. Specific numbers have been adjusted for confidentiality.*
