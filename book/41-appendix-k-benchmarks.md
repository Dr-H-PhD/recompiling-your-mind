# Appendix K: Performance Benchmarks

Comparative benchmarks between PHP and Go for common operations. These benchmarks illustrate typical performance differences, not absolute values — your mileage may vary.

---

## Test Environment

- **Hardware:** Apple M2 Pro, 16GB RAM
- **PHP:** 8.3.2, OPcache enabled
- **Go:** 1.22.0
- **Database:** PostgreSQL 16, local socket
- **HTTP:** wrk for load testing

---

## 1. HTTP Server Performance

### Benchmark: Simple JSON API

**PHP (Laravel):**
```php
Route::get('/api/users', function () {
    return response()->json([
        'users' => User::take(10)->get()
    ]);
});
```

**Go (net/http):**
```go
func usersHandler(w http.ResponseWriter, r *http.Request) {
    users, _ := repo.FindUsers(r.Context(), 10)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "users": users,
    })
}
```

### Results (wrk -t12 -c400 -d30s)

| Metric | PHP (Laravel) | Go (net/http) | Factor |
|--------|---------------|---------------|--------|
| Requests/sec | 2,340 | 89,500 | 38x |
| Latency (avg) | 170ms | 4.5ms | 38x |
| Latency (p99) | 890ms | 12ms | 74x |
| Memory | 512MB | 24MB | 21x |
| CPU usage | 380% | 95% | 4x |

### Analysis

The difference is primarily due to:
1. Go's compiled nature vs PHP's interpreted
2. Go's built-in concurrency vs PHP-FPM process model
3. Go's efficient memory allocation

---

## 2. JSON Serialisation

### Benchmark: Encode/Decode 10,000 Objects

**PHP:**
```php
$start = microtime(true);
for ($i = 0; $i < 10000; $i++) {
    $json = json_encode($users);
    $decoded = json_decode($json, true);
}
$elapsed = microtime(true) - $start;
```

**Go:**
```go
start := time.Now()
for i := 0; i < 10000; i++ {
    data, _ := json.Marshal(users)
    json.Unmarshal(data, &decoded)
}
elapsed := time.Since(start)
```

### Results

| Operation | PHP | Go | Factor |
|-----------|-----|-----|--------|
| Encode (10k ops) | 1.2s | 0.08s | 15x |
| Decode (10k ops) | 1.8s | 0.12s | 15x |
| Memory per op | 2.1KB | 0.3KB | 7x |

---

## 3. Database Operations

### Benchmark: 1,000 Sequential Inserts

**PHP (PDO):**
```php
$stmt = $pdo->prepare("INSERT INTO users (name, email) VALUES (?, ?)");
for ($i = 0; $i < 1000; $i++) {
    $stmt->execute(["User $i", "user$i@example.com"]);
}
```

**Go (database/sql):**
```go
stmt, _ := db.Prepare("INSERT INTO users (name, email) VALUES ($1, $2)")
for i := 0; i < 1000; i++ {
    stmt.Exec(fmt.Sprintf("User %d", i), fmt.Sprintf("user%d@example.com", i))
}
```

### Results

| Operation | PHP | Go | Factor |
|-----------|-----|-----|--------|
| 1,000 inserts | 2.8s | 1.9s | 1.5x |
| With batch (100/batch) | 0.4s | 0.25s | 1.6x |
| Concurrent (10 goroutines) | N/A | 0.3s | 9x vs PHP |

### Key Insight

Database operations are I/O-bound, so the language difference is smaller. Go's advantage comes from concurrent execution:

```go
// Go can insert concurrently
var wg sync.WaitGroup
ch := make(chan int, 100)

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for id := range ch {
            stmt.Exec(fmt.Sprintf("User %d", id), ...)
        }
    }()
}

for i := 0; i < 1000; i++ {
    ch <- i
}
close(ch)
wg.Wait()
```

---

## 4. String Processing

### Benchmark: Parse 100MB Log File

**PHP:**
```php
$lines = file('access.log');
$ips = [];
foreach ($lines as $line) {
    if (preg_match('/^(\d+\.\d+\.\d+\.\d+)/', $line, $m)) {
        $ips[$m[1]] = ($ips[$m[1]] ?? 0) + 1;
    }
}
```

**Go:**
```go
file, _ := os.Open("access.log")
scanner := bufio.NewScanner(file)
ips := make(map[string]int)
re := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)`)

for scanner.Scan() {
    if m := re.FindString(scanner.Text()); m != "" {
        ips[m]++
    }
}
```

### Results

| Metric | PHP | Go | Factor |
|--------|-----|-----|--------|
| Time | 8.5s | 0.9s | 9x |
| Memory | 890MB | 45MB | 20x |

### With Concurrency (Go only)

```go
// Split file into chunks, process in parallel
var wg sync.WaitGroup
results := make(chan map[string]int, runtime.NumCPU())

// Process chunks concurrently
// Time: 0.25s (34x faster than PHP)
```

---

## 5. Cryptographic Operations

### Benchmark: Hash 100,000 Passwords

**PHP:**
```php
for ($i = 0; $i < 100000; $i++) {
    password_hash("password$i", PASSWORD_BCRYPT, ['cost' => 10]);
}
```

**Go:**
```go
for i := 0; i < 100000; i++ {
    bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("password%d", i)), 10)
}
```

### Results (Single-threaded)

| Operation | PHP | Go | Factor |
|-----------|-----|-----|--------|
| 100k bcrypt hashes | 285s | 278s | ~1x |

### Results (Go with Concurrency)

```go
var wg sync.WaitGroup
sem := make(chan struct{}, runtime.NumCPU())

for i := 0; i < 100000; i++ {
    wg.Add(1)
    sem <- struct{}{}
    go func(i int) {
        defer wg.Done()
        defer func() { <-sem }()
        bcrypt.GenerateFromPassword(...)
    }(i)
}
// Time: 35s (8x faster on 8-core machine)
```

### Key Insight

CPU-bound operations like bcrypt are similar per-operation. Go's advantage is parallelism, which PHP-FPM can't match within a single request.

---

## 6. Memory Efficiency

### Benchmark: In-Memory Cache (1 Million Entries)

**PHP:**
```php
$cache = [];
for ($i = 0; $i < 1000000; $i++) {
    $cache["key_$i"] = "value_$i";
}
// Memory: 256MB
```

**Go:**
```go
cache := make(map[string]string, 1000000)
for i := 0; i < 1000000; i++ {
    cache[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
}
// Memory: 89MB
```

### Results

| Structure | PHP | Go | Factor |
|-----------|-----|-----|--------|
| 1M map entries | 256MB | 89MB | 2.9x |
| 1M objects | 512MB | 120MB | 4.3x |
| Empty process | 15MB | 2MB | 7.5x |

---

## 7. Startup Time

### Benchmark: Cold Start to First Response

| Scenario | PHP (Laravel) | Go | Factor |
|----------|---------------|-----|--------|
| Cold start | 180ms | 5ms | 36x |
| Warm start (OPcache) | 45ms | N/A | - |
| Docker container start | 2.5s | 50ms | 50x |

### Impact

- Serverless: Go's fast start makes it ideal for AWS Lambda, Cloud Functions
- Kubernetes: Go pods scale up instantly vs PHP needing warm-up

---

## 8. Concurrency Under Load

### Benchmark: 10,000 Concurrent Connections

Test: Hold 10,000 connections, each making a request every second.

| Metric | PHP-FPM | Go |
|--------|---------|-----|
| Max connections | 500 | 50,000+ |
| Memory for 10k | Impossible | 150MB |
| Workers needed | N/A | 1 process |

### PHP Limitation

PHP-FPM's process-per-request model means:
- 10,000 connections = 10,000 processes needed
- Each process: ~20MB = 200GB RAM
- Impractical at scale

### Go Solution

```go
// Single Go process handles 10k connections
server := &http.Server{
    Handler:      handler,
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 30 * time.Second,
}
server.ListenAndServe()
// Each connection is a goroutine (~2KB stack)
// 10k connections = ~20MB
```

---

## Summary Table

| Category | PHP Relative | Go Relative | Notes |
|----------|--------------|-------------|-------|
| HTTP throughput | 1x | 30-50x | Process vs goroutine model |
| JSON processing | 1x | 10-20x | Compiled vs interpreted |
| Database (single) | 1x | 1.5x | I/O bound, similar |
| Database (concurrent) | 1x | 5-10x | Go's concurrency wins |
| String processing | 1x | 5-10x | Memory efficiency |
| Crypto operations | 1x | 1x | CPU bound, same algorithms |
| Crypto (parallel) | 1x | Nx | N = CPU cores |
| Memory usage | 1x | 3-10x less | - |
| Startup time | 1x | 30-50x | Critical for serverless |
| Concurrent connections | 500 max | 50,000+ | Different models |

---

## When Performance Doesn't Matter

Not every application needs Go's performance:

- Admin dashboards with 10 users
- CRUD apps with moderate traffic
- Content websites
- Prototypes and MVPs

PHP is "fast enough" for many use cases. Migrate for the right reasons, not premature optimisation.

---

*Benchmark methodology and code available in the companion repository.*
