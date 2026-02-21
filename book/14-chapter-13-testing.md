# Chapter 13: Testing — A Different Philosophy

PHPUnit is the standard for PHP testing—assertions, mocks, data providers, coverage. Go's testing approach is deliberately simpler, built into the language and standard library.

## Table-Driven Tests

PHPUnit uses data providers:

```php
#[DataProvider('additionProvider')]
public function testAdd(int $a, int $b, int $expected): void
{
    $this->assertEquals($expected, $this->calculator->add($a, $b));
}

public static function additionProvider(): array
{
    return [
        [1, 2, 3],
        [0, 0, 0],
        [-1, 1, 0],
    ];
}
```

Go uses table-driven tests:

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 1, 2, 3},
        {"zeros", 0, 0, 0},
        {"negative and positive", -1, 1, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

### Why Table-Driven?

1. **Easy to add cases**: Just add a row to the table
2. **Clear structure**: Input → expected output
3. **Named subtests**: Each case runs as `TestAdd/positive_numbers`
4. **Parallel execution**: Add `t.Parallel()` for concurrent tests

```go
func TestProcess(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "hello", "HELLO", false},
        {"empty input", "", "", true},
        {"special chars", "hello!", "HELLO!", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // Run subtests in parallel

            got, err := Process(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("Process() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Process() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## No Assertions Library (By Design)

PHPUnit has rich assertions:

```php
$this->assertEquals($expected, $actual);
$this->assertContains($item, $array);
$this->assertInstanceOf(User::class, $result);
$this->assertGreaterThan(0, $count);
```

Go's `testing` package has only `t.Error`, `t.Errorf`, `t.Fatal`, `t.Fatalf`:

```go
if got != want {
    t.Errorf("got %v, want %v", got, want)
}

if user == nil {
    t.Fatal("user is nil")  // Stops the test
}
```

### Why No Assertions?

Go's philosophy: assertions are just `if` statements with better errors. The testing package doesn't need to provide them—you write clear comparison code.

### Third-Party Options

If you want assertions, use `testify`:

```go
import "github.com/stretchr/testify/assert"

func TestUser(t *testing.T) {
    user := NewUser("Alice")

    assert.Equal(t, "Alice", user.Name)
    assert.NotNil(t, user.ID)
    assert.True(t, user.IsActive())
}
```

But many Go developers prefer vanilla testing for consistency.

## Mocking with Interfaces (vs Prophecy/Mockery)

PHP uses mocking libraries:

```php
$repository = $this->createMock(UserRepository::class);
$repository
    ->expects($this->once())
    ->method('find')
    ->with(42)
    ->willReturn($user);

$service = new UserService($repository);
```

Go mocks via interfaces:

```go
// Interface to mock
type UserRepository interface {
    Find(ctx context.Context, id int) (*User, error)
}

// Test mock implementation
type mockUserRepo struct {
    user *User
    err  error
}

func (m *mockUserRepo) Find(ctx context.Context, id int) (*User, error) {
    return m.user, m.err
}

// Test
func TestGetUser(t *testing.T) {
    expectedUser := &User{ID: 42, Name: "Alice"}
    repo := &mockUserRepo{user: expectedUser}
    service := NewUserService(repo)

    user, err := service.GetUser(context.Background(), 42)

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if user.Name != "Alice" {
        t.Errorf("got name %s, want Alice", user.Name)
    }
}
```

### Why Manual Mocks?

1. **Type-safe**: The compiler ensures mock implements interface
2. **Explicit**: You see exactly what the mock does
3. **Flexible**: Add any behaviour you need
4. **No runtime reflection**: Pure Go code

### Mock Generation Tools

For large interfaces, generate mocks:

```go
//go:generate mockgen -source=repository.go -destination=mock_repository.go

type UserRepository interface {
    Find(ctx context.Context, id int) (*User, error)
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id int) error
}
```

`mockgen` creates a mock with expectation setting and verification.

## Integration Tests

PHPUnit integration tests often use Symfony's WebTestCase:

```php
class UserControllerTest extends WebTestCase
{
    public function testCreateUser(): void
    {
        $client = static::createClient();
        $client->request('POST', '/api/users', [], [],
            ['CONTENT_TYPE' => 'application/json'],
            json_encode(['name' => 'Alice'])
        );

        $this->assertResponseStatusCodeSame(201);
    }
}
```

Go uses `httptest`:

```go
func TestCreateUser(t *testing.T) {
    // Setup
    repo := NewInMemoryUserRepo()
    handler := NewUserHandler(repo)

    // Create request
    body := strings.NewReader(`{"name":"Alice"}`)
    req := httptest.NewRequest("POST", "/users", body)
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    // Execute
    handler.Create(rec, req)

    // Assert
    if rec.Code != http.StatusCreated {
        t.Errorf("status = %d; want %d", rec.Code, http.StatusCreated)
    }

    var response User
    json.NewDecoder(rec.Body).Decode(&response)
    if response.Name != "Alice" {
        t.Errorf("name = %s; want Alice", response.Name)
    }
}
```

### Testing the Full Stack

```go
func TestAPI(t *testing.T) {
    // Setup real server
    db := setupTestDB(t)
    server := NewServer(db)

    ts := httptest.NewServer(server)
    defer ts.Close()

    // Make HTTP request
    resp, err := http.Post(ts.URL+"/users", "application/json",
        strings.NewReader(`{"name":"Alice"}`))
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        t.Errorf("status = %d; want %d", resp.StatusCode, http.StatusCreated)
    }
}
```

## Benchmarking Built-In

PHPUnit benchmarking requires additional tooling. Go has built-in benchmarks:

```go
func BenchmarkProcess(b *testing.B) {
    input := "test data"

    for i := 0; i < b.N; i++ {
        Process(input)
    }
}
```

Run with:

```bash
go test -bench=. -benchmem

# Output:
# BenchmarkProcess-8   1000000   1234 ns/op   256 B/op   2 allocs/op
```

### Benchmark Best Practices

```go
func BenchmarkProcess(b *testing.B) {
    input := generateLargeInput()  // Setup outside loop

    b.ResetTimer()  // Don't count setup time

    for i := 0; i < b.N; i++ {
        Process(input)
    }
}

// Compare implementations
func BenchmarkProcessV1(b *testing.B) {
    for i := 0; i < b.N; i++ {
        ProcessV1(input)
    }
}

func BenchmarkProcessV2(b *testing.B) {
    for i := 0; i < b.N; i++ {
        ProcessV2(input)
    }
}
```

## Coverage Tooling

PHPUnit coverage with Xdebug:

```bash
XDEBUG_MODE=coverage phpunit --coverage-html coverage/
```

Go coverage:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # Open in browser
go tool cover -func=coverage.out  # Print coverage by function
```

### Coverage in CI

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total:
# total:  (statements)  85.2%
```

## Test Containers for Integration Tests

PHPUnit might use Docker through manual setup. Go has `testcontainers-go`:

```go
import "github.com/testcontainers/testcontainers-go"

func TestWithPostgres(t *testing.T) {
    ctx := context.Background()

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "postgres:15",
            ExposedPorts: []string{"5432/tcp"},
            Env: map[string]string{
                "POSTGRES_PASSWORD": "test",
                "POSTGRES_DB":       "test",
            },
            WaitingFor: wait.ForListeningPort("5432/tcp"),
        },
        Started: true,
    })
    if err != nil {
        t.Fatal(err)
    }
    defer container.Terminate(ctx)

    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "5432")

    // Connect to container's postgres
    dsn := fmt.Sprintf("postgres://postgres:test@%s:%s/test", host, port.Port())
    db, err := sql.Open("postgres", dsn)
    // ... run tests
}
```

## Summary

- **Table-driven tests** are idiomatic for parameterised testing
- **No assertions library** by design; use `if` statements
- **Interface mocking** is manual but type-safe
- **httptest** provides test servers and recorders
- **Benchmarking** is built into `go test`
- **Coverage** via `go test -cover`
- **Test containers** for integration tests with real dependencies

---

## Exercises

1. **Table-Driven Conversion**: Convert a PHPUnit test with data provider to Go table-driven style.

2. **Mock Implementation**: Define an interface with 3 methods. Write a manual mock. Write a test using it.

3. **HTTP Handler Test**: Write tests for a handler covering success, validation error, and not found cases.

4. **Benchmark Comparison**: Write two implementations of the same function. Benchmark both. Identify the faster one.

5. **Coverage Analysis**: Run coverage on a package. Identify untested code paths. Add tests to increase coverage.

6. **Test Containers**: Set up a test with testcontainers for a real database. Run migrations. Execute queries.

7. **Parallel Tests**: Convert sequential tests to parallel using `t.Parallel()`. Verify they don't interfere.

8. **Test Helper Functions**: Create reusable test helpers for common setup (creating users, authenticated requests, etc.).
