# Appendix A: PHP-to-Go Phrasebook

Quick reference for common PHP/Symfony patterns and their Go equivalents.

## Language Basics

| PHP | Go |
|-----|-----|
| `$variable = 42;` | `variable := 42` |
| `$arr = [];` | `arr := []T{}` or `arr := make([]T, 0)` |
| `$map = [];` | `m := make(map[K]V)` |
| `function name($a) {}` | `func name(a T) {}` |
| `public function` | `func (r *Receiver) Method()` (uppercase) |
| `private function` | `func (r *Receiver) method()` (lowercase) |
| `class Foo {}` | `type Foo struct {}` |
| `new Foo()` | `&Foo{}` or `NewFoo()` |
| `$this->property` | `r.property` |
| `$this->method()` | `r.Method()` |
| `null` | `nil` |
| `true/false` | `true/false` |
| `echo "text";` | `fmt.Println("text")` |

## Control Flow

| PHP | Go |
|-----|-----|
| `if ($x) { }` | `if x { }` |
| `if ($x) { } else { }` | `if x { } else { }` |
| `elseif` | `else if` |
| `switch ($x) { case 1: break; }` | `switch x { case 1: }` (no break needed) |
| `for ($i = 0; $i < 10; $i++)` | `for i := 0; i < 10; i++` |
| `foreach ($arr as $v)` | `for _, v := range arr` |
| `foreach ($arr as $k => $v)` | `for k, v := range arr` |
| `while ($cond) { }` | `for cond { }` |
| `try { } catch { }` | `if err != nil { }` |
| `throw new Exception()` | `return errors.New()` |

## Types

| PHP | Go |
|-----|-----|
| `int` | `int`, `int64`, `int32` |
| `float` | `float64`, `float32` |
| `string` | `string` |
| `bool` | `bool` |
| `array` (sequential) | `[]T` (slice) |
| `array` (associative) | `map[K]V` |
| `?string` (nullable) | `*string` or custom null type |
| `mixed` | `any` or `interface{}` |
| `object` | `struct` |
| `callable` | `func(args) returns` |

## String Operations

| PHP | Go |
|-----|-----|
| `strlen($s)` | `len(s)` (bytes) or `utf8.RuneCountInString(s)` |
| `$s1 . $s2` | `s1 + s2` or `fmt.Sprintf("%s%s", s1, s2)` |
| `strpos($s, $sub)` | `strings.Index(s, sub)` |
| `substr($s, $start, $len)` | `s[start:start+len]` |
| `str_replace($old, $new, $s)` | `strings.Replace(s, old, new, -1)` |
| `explode(",", $s)` | `strings.Split(s, ",")` |
| `implode(",", $arr)` | `strings.Join(arr, ",")` |
| `trim($s)` | `strings.TrimSpace(s)` |
| `strtolower($s)` | `strings.ToLower(s)` |
| `sprintf("%s", $v)` | `fmt.Sprintf("%s", v)` |

## Array/Slice Operations

| PHP | Go |
|-----|-----|
| `count($arr)` | `len(arr)` |
| `$arr[] = $v` | `arr = append(arr, v)` |
| `array_push($arr, $v)` | `arr = append(arr, v)` |
| `array_pop($arr)` | `arr = arr[:len(arr)-1]` |
| `array_slice($arr, $start, $len)` | `arr[start:start+len]` |
| `in_array($v, $arr)` | Loop or `slices.Contains(arr, v)` |
| `array_keys($arr)` | `maps.Keys(m)` (Go 1.21+) |
| `array_values($arr)` | `maps.Values(m)` (Go 1.21+) |
| `array_merge($a, $b)` | `slices.Concat(a, b)` (Go 1.22+) |
| `array_filter($arr, $fn)` | Loop with condition |
| `array_map($fn, $arr)` | Loop with transformation |
| `usort($arr, $fn)` | `slices.SortFunc(arr, fn)` |

## Error Handling

| PHP | Go |
|-----|-----|
| `throw new Exception($msg)` | `return fmt.Errorf("msg: %w", err)` |
| `try { } catch (E $e) { }` | `if err != nil { }` |
| `$e->getMessage()` | `err.Error()` |
| `$e instanceof MyException` | `errors.Is(err, ErrMy)` or `errors.As(err, &myErr)` |
| Custom exception class | Custom error type implementing `error` |
| `finally { }` | `defer func() { }()` |

## Doctrine ORM → database/sql

| Doctrine | Go |
|----------|-----|
| `$em->find(User::class, $id)` | `db.QueryRowContext(ctx, "SELECT...", id).Scan(&u)` |
| `$em->persist($entity)` | `db.ExecContext(ctx, "INSERT...", fields...)` |
| `$em->flush()` | Transactions: `tx.Commit()` |
| `$repo->findBy(['status' => $s])` | `db.QueryContext(ctx, "SELECT...WHERE status=$1", s)` |
| `$qb->select()...->getQuery()` | Raw SQL or squirrel query builder |
| `@Entity` | `type Entity struct {}` |
| `@Column` | Struct fields |
| `@OneToMany` | Separate queries or JOINs |

## Symfony HttpFoundation → net/http

| Symfony | Go |
|---------|-----|
| `$request->query->get('key')` | `r.URL.Query().Get("key")` |
| `$request->request->get('key')` | `r.FormValue("key")` |
| `$request->getContent()` | `io.ReadAll(r.Body)` |
| `$request->headers->get('X-Foo')` | `r.Header.Get("X-Foo")` |
| `$request->getMethod()` | `r.Method` |
| `$request->getPathInfo()` | `r.URL.Path` |
| `new Response($body, 200)` | `w.WriteHeader(200); w.Write([]byte(body))` |
| `new JsonResponse($data)` | `json.NewEncoder(w).Encode(data)` |
| `$response->headers->set(...)` | `w.Header().Set(...)` |

## Symfony Services

| Symfony | Go |
|---------|-----|
| `#[AsService]` | No equivalent—just a struct |
| Constructor injection | Pass dependencies to `New*` function |
| `#[Required]` | Constructor parameter |
| `services.yaml` | Explicit wiring in `main.go` |
| `$container->get(Foo::class)` | Direct instantiation |
| Interface binding | Accept interface in `New*` function |

## Testing

| PHPUnit | Go testing |
|---------|-----|
| `class FooTest extends TestCase` | `func TestFoo(t *testing.T)` |
| `$this->assertEquals($a, $b)` | `if a != b { t.Errorf(...) }` |
| `$this->assertTrue($x)` | `if !x { t.Error(...) }` |
| `$this->expectException(E::class)` | Check returned error |
| `@dataProvider` | Table-driven tests |
| `$this->createMock(Foo::class)` | Manual mock or mockgen |
| `setUp()` | Code before test or `t.Cleanup()` |
| `tearDown()` | `t.Cleanup()` or `defer` |

## Common Patterns

### Singleton (PHP) → Package Variable (Go)

```php
class Database {
    private static ?self $instance = null;
    public static function getInstance(): self {
        return self::$instance ??= new self();
    }
}
```

```go
var db *sql.DB
var once sync.Once

func GetDB() *sql.DB {
    once.Do(func() {
        db, _ = sql.Open("postgres", dsn)
    })
    return db
}
```

### Factory (PHP) → New* Function (Go)

```php
class UserFactory {
    public function create(string $name): User {
        return new User($name, new DateTime());
    }
}
```

```go
func NewUser(name string) *User {
    return &User{Name: name, CreatedAt: time.Now()}
}
```

### Builder (PHP) → Functional Options (Go)

```php
$user = (new UserBuilder())
    ->setName("Alice")
    ->setEmail("alice@example.com")
    ->build();
```

```go
type Option func(*User)

func WithEmail(e string) Option { return func(u *User) { u.Email = e } }

func NewUser(name string, opts ...Option) *User {
    u := &User{Name: name}
    for _, opt := range opts {
        opt(u)
    }
    return u
}

user := NewUser("Alice", WithEmail("alice@example.com"))
```

### Repository (PHP) → Interface + Struct (Go)

```php
interface UserRepositoryInterface {
    public function find(int $id): ?User;
}

class DoctrineUserRepository implements UserRepositoryInterface {
    public function find(int $id): ?User { }
}
```

```go
type UserRepository interface {
    Find(ctx context.Context, id int) (*User, error)
}

type PostgresUserRepository struct {
    db *sql.DB
}

func (r *PostgresUserRepository) Find(ctx context.Context, id int) (*User, error) {
    // Implementation
}
```
