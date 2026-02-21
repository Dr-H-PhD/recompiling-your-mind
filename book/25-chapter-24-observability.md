# Chapter 24: Observability

PHP logging typically uses Monolog with various handlers. Go's observability stack is different—structured logging, Prometheus metrics, and OpenTelemetry tracing form the modern approach.

## Structured Logging (slog vs Monolog)

Monolog:
```php
$logger->info('User created', [
    'user_id' => $user->getId(),
    'email' => $user->getEmail(),
]);
// Output depends on handler (JSON, line format, etc.)
```

Go's `log/slog` (Go 1.21+):
```go
import "log/slog"

func main() {
    // JSON output
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    slog.Info("User created",
        "user_id", user.ID,
        "email", user.Email,
    )
}

// Output:
// {"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"User created","user_id":123,"email":"alice@example.com"}
```

### Log Levels

```go
slog.Debug("Detailed info", "key", "value")
slog.Info("Normal operation", "key", "value")
slog.Warn("Something unusual", "key", "value")
slog.Error("Something failed", "error", err, "key", "value")
```

### Contextual Logging

```go
// Add context that applies to all subsequent logs
logger := slog.With("request_id", requestID, "user_id", userID)
logger.Info("Processing request")
logger.Info("Request completed")
```

### Handler Configuration

```go
// JSON with custom options
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
    AddSource: true,  // Include file:line
})

// Text for development
handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})
```

### Request Logging Middleware

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        requestID := uuid.New().String()

        // Add to context for downstream use
        ctx := context.WithValue(r.Context(), "request_id", requestID)
        r = r.WithContext(ctx)

        // Wrap response writer to capture status
        lrw := &loggingResponseWriter{ResponseWriter: w, status: 200}

        next.ServeHTTP(lrw, r)

        slog.Info("HTTP request",
            "method", r.Method,
            "path", r.URL.Path,
            "status", lrw.status,
            "duration", time.Since(start),
            "request_id", requestID,
        )
    })
}
```

## Metrics with Prometheus

PHP metrics might use StatsD or custom solutions. Go typically uses Prometheus.

### Setup

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
}

func main() {
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":8080", nil)
}
```

### Metrics Middleware

```go
func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        lrw := &loggingResponseWriter{ResponseWriter: w, status: 200}

        next.ServeHTTP(lrw, r)

        duration := time.Since(start).Seconds()
        status := strconv.Itoa(lrw.status)

        httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
        httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
    })
}
```

### Metric Types

```go
// Counter: Only goes up
requestCount := prometheus.NewCounter(prometheus.CounterOpts{
    Name: "requests_total",
})
requestCount.Inc()

// Gauge: Can go up or down
activeConnections := prometheus.NewGauge(prometheus.GaugeOpts{
    Name: "active_connections",
})
activeConnections.Inc()
activeConnections.Dec()

// Histogram: Distribution of values
requestDuration := prometheus.NewHistogram(prometheus.HistogramOpts{
    Name:    "request_duration_seconds",
    Buckets: []float64{.001, .005, .01, .05, .1, .5, 1},
})
requestDuration.Observe(0.042)

// Summary: Similar to histogram with percentiles
requestLatency := prometheus.NewSummary(prometheus.SummaryOpts{
    Name:       "request_latency_seconds",
    Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
})
```

## Tracing with OpenTelemetry

Distributed tracing tracks requests across services.

### Setup

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() (*trace.TracerProvider, error) {
    exporter, err := otlptracehttp.New(context.Background())
    if err != nil {
        return nil, err
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("myapp"),
        )),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

### Creating Spans

```go
var tracer = otel.Tracer("myapp")

func handleRequest(ctx context.Context) error {
    ctx, span := tracer.Start(ctx, "handleRequest")
    defer span.End()

    // Add attributes
    span.SetAttributes(
        attribute.String("user_id", userID),
        attribute.Int("item_count", len(items)),
    )

    // Call other services (context propagates trace)
    if err := callDatabase(ctx); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    return nil
}
```

### HTTP Instrumentation

```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

handler := otelhttp.NewHandler(mux, "server")
http.ListenAndServe(":8080", handler)
```

## Health Checks

Kubernetes and load balancers need health endpoints:

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

func readinessHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()

        if err := db.PingContext(ctx); err != nil {
            http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
            return
        }

        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Ready"))
    }
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthHandler)       // Liveness
    mux.HandleFunc("/ready", readinessHandler(db)) // Readiness
}
```

### Health Check Best Practices

- **Liveness** (`/health`): Is the process running? Keep simple.
- **Readiness** (`/ready`): Can the process handle traffic? Check dependencies.
- **Startup** (`/startup`): Has the process finished initialising?

## Error Tracking (Sentry Integration)

```go
import "github.com/getsentry/sentry-go"

func init() {
    sentry.Init(sentry.ClientOptions{
        Dsn:         os.Getenv("SENTRY_DSN"),
        Environment: os.Getenv("ENV"),
        Release:     version,
    })
}

func handleError(err error) {
    sentry.CaptureException(err)
}

// HTTP middleware
func sentryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                sentry.CurrentHub().Recover(err)
                http.Error(w, "Internal error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

## Summary

- **Structured logging** with `slog` outputs JSON for log aggregation
- **Prometheus metrics** expose counters, gauges, and histograms
- **OpenTelemetry** provides distributed tracing
- **Health checks** enable container orchestration
- **Error tracking** with Sentry captures exceptions

---

## Exercises

1. **Structured Logging**: Replace `fmt.Println` with `slog` throughout an application.

2. **Request ID Propagation**: Add request ID to all logs within a request lifecycle.

3. **Prometheus Metrics**: Add request count and duration metrics. View in Prometheus.

4. **Custom Metrics**: Create business metrics (orders placed, users registered, etc.).

5. **OpenTelemetry Setup**: Add tracing to a multi-service application. View traces in Jaeger.

6. **Health Checks**: Implement health, readiness, and startup probes with dependency checks.

7. **Sentry Integration**: Set up Sentry. Trigger errors and verify they appear in Sentry.

8. **Observability Dashboard**: Create a Grafana dashboard showing logs, metrics, and traces together.
