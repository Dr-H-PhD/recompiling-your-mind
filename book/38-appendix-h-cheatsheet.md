# Appendix H: Go Cheat Sheet for PHP Developers

A quick reference card for common operations, comparing PHP and Go syntax.

---

## Variables and Types

| Operation | PHP | Go |
|-----------|-----|-----|
| Variable declaration | `$name = "Alice";` | `name := "Alice"` |
| Explicit type | `string $name = "Alice";` | `var name string = "Alice"` |
| Constants | `const MAX = 100;` | `const Max = 100` |
| Multiple declaration | `$a = $b = 0;` | `var a, b int` |
| Null/nil | `$ptr = null;` | `var ptr *int = nil` |

## Strings

| Operation | PHP | Go |
|-----------|-----|-----|
| Concatenation | `$s = $a . $b;` | `s := a + b` |
| Interpolation | `"Hello $name"` | `fmt.Sprintf("Hello %s", name)` |
| Length | `strlen($s)` | `len(s)` |
| Substring | `substr($s, 0, 5)` | `s[:5]` |
| Contains | `str_contains($s, "x")` | `strings.Contains(s, "x")` |
| Split | `explode(",", $s)` | `strings.Split(s, ",")` |
| Join | `implode(",", $arr)` | `strings.Join(arr, ",")` |
| Trim | `trim($s)` | `strings.TrimSpace(s)` |
| To upper | `strtoupper($s)` | `strings.ToUpper(s)` |
| Replace | `str_replace("a", "b", $s)` | `strings.ReplaceAll(s, "a", "b")` |

## Arrays/Slices

| Operation | PHP | Go |
|-----------|-----|-----|
| Create | `$arr = [1, 2, 3];` | `arr := []int{1, 2, 3}` |
| Append | `$arr[] = 4;` | `arr = append(arr, 4)` |
| Length | `count($arr)` | `len(arr)` |
| Access | `$arr[0]` | `arr[0]` |
| Slice | `array_slice($arr, 1, 2)` | `arr[1:3]` |
| Map/transform | `array_map(fn, $arr)` | Loop or generics |
| Filter | `array_filter($arr, fn)` | Loop or generics |
| In array | `in_array($x, $arr)` | `slices.Contains(arr, x)` |
| Reverse | `array_reverse($arr)` | `slices.Reverse(arr)` |

## Maps (Associative Arrays)

| Operation | PHP | Go |
|-----------|-----|-----|
| Create | `$m = ['a' => 1];` | `m := map[string]int{"a": 1}` |
| Set | `$m['b'] = 2;` | `m["b"] = 2` |
| Get | `$m['a']` | `m["a"]` |
| Get with default | `$m['x'] ?? 0` | `v, ok := m["x"]` |
| Delete | `unset($m['a']);` | `delete(m, "a")` |
| Key exists | `isset($m['a'])` | `_, ok := m["a"]` |
| Iterate | `foreach ($m as $k => $v)` | `for k, v := range m` |

## Control Flow

| Operation | PHP | Go |
|-----------|-----|-----|
| If | `if ($x > 0) { }` | `if x > 0 { }` |
| If-else | `if () { } else { }` | `if { } else { }` |
| Ternary | `$x > 0 ? "yes" : "no"` | No ternary (use if) |
| Switch | `switch ($x) { case 1: }` | `switch x { case 1: }` |
| For loop | `for ($i=0; $i<10; $i++)` | `for i := 0; i < 10; i++` |
| While | `while ($x) { }` | `for x { }` |
| Foreach | `foreach ($arr as $v)` | `for _, v := range arr` |
| Foreach key-value | `foreach ($arr as $k => $v)` | `for k, v := range arr` |

## Functions

| Operation | PHP | Go |
|-----------|-----|-----|
| Define | `function add($a, $b) { return $a + $b; }` | `func add(a, b int) int { return a + b }` |
| Multiple returns | Return array | `func f() (int, error)` |
| Variadic | `function f(...$args)` | `func f(args ...int)` |
| Anonymous | `$f = function($x) { };` | `f := func(x int) int { }` |
| Arrow function | `fn($x) => $x * 2` | `func(x int) int { return x * 2 }` |
| Closure | `function() use ($x) { }` | `func() { /* x captured */ }` |

## Error Handling

| Operation | PHP | Go |
|-----------|-----|-----|
| Return error | `throw new Exception()` | `return nil, errors.New("msg")` |
| Handle error | `try { } catch (E $e) { }` | `if err != nil { }` |
| Wrap error | `throw new E("msg", 0, $prev)` | `fmt.Errorf("ctx: %w", err)` |
| Check type | `catch (TypeError $e)` | `errors.As(err, &target)` |
| Check value | `$e->getMessage()` | `errors.Is(err, target)` |

## Classes/Structs

| Operation | PHP | Go |
|-----------|-----|-----|
| Define | `class User { }` | `type User struct { }` |
| Properties | `public string $name;` | `Name string` (exported) |
| Private | `private $name;` | `name string` (unexported) |
| Constructor | `function __construct()` | `func NewUser() *User` |
| Method | `public function getName()` | `func (u *User) GetName()` |
| This | `$this->name` | `u.Name` (receiver) |
| Inheritance | `class A extends B` | Embedding: `type A struct { B }` |
| Interface | `class A implements I` | Implicit (no keyword) |

## JSON

| Operation | PHP | Go |
|-----------|-----|-----|
| Encode | `json_encode($data)` | `json.Marshal(data)` |
| Decode | `json_decode($s, true)` | `json.Unmarshal([]byte(s), &data)` |
| Decode to struct | N/A | Define struct with tags |
| Custom field | N/A | `` `json:"field_name"` `` |
| Omit empty | N/A | `` `json:"field,omitempty"` `` |
| Ignore field | N/A | `` `json:"-"` `` |

## HTTP

| Operation | PHP | Go |
|-----------|-----|-----|
| Handle route | Controller class | `http.HandleFunc("/", handler)` |
| Start server | Built-in/nginx | `http.ListenAndServe(":8080", nil)` |
| Get param | `$_GET['id']` | `r.URL.Query().Get("id")` |
| Post data | `$_POST['name']` | `r.FormValue("name")` |
| JSON body | `json_decode(file_get_contents('php://input'))` | `json.NewDecoder(r.Body).Decode(&v)` |
| Set header | `header('Content-Type: application/json')` | `w.Header().Set("Content-Type", "application/json")` |
| Status code | `http_response_code(404)` | `w.WriteHeader(http.StatusNotFound)` |
| Write response | `echo $data` | `w.Write([]byte(data))` |

## Database

| Operation | PHP | Go |
|-----------|-----|-----|
| Connect | `new PDO($dsn)` | `sql.Open("driver", dsn)` |
| Query rows | `$stmt->fetchAll()` | `db.Query(q)` + rows.Next() |
| Query one | `$stmt->fetch()` | `db.QueryRow(q).Scan(&v)` |
| Execute | `$stmt->execute()` | `db.Exec(q)` |
| Prepared | `$pdo->prepare($q)` | `db.Prepare(q)` |
| Transaction | `$pdo->beginTransaction()` | `db.Begin()` |
| Commit | `$pdo->commit()` | `tx.Commit()` |
| Rollback | `$pdo->rollBack()` | `tx.Rollback()` |

## Concurrency (Go only)

| Operation | Go |
|-----------|-----|
| Start goroutine | `go func() { }()` |
| Create channel | `ch := make(chan int)` |
| Buffered channel | `ch := make(chan int, 10)` |
| Send | `ch <- value` |
| Receive | `value := <-ch` |
| Close | `close(ch)` |
| Range over channel | `for v := range ch { }` |
| Select | `select { case <-ch: }` |
| Timeout | `case <-time.After(1 * time.Second):` |
| WaitGroup add | `wg.Add(1)` |
| WaitGroup done | `defer wg.Done()` |
| WaitGroup wait | `wg.Wait()` |
| Mutex lock | `mu.Lock(); defer mu.Unlock()` |

## Testing

| Operation | PHP | Go |
|-----------|-----|-----|
| Test file | `UserTest.php` | `user_test.go` |
| Test function | `public function testX()` | `func TestX(t *testing.T)` |
| Assert equal | `$this->assertEquals($a, $b)` | `if got != want { t.Errorf(...) }` |
| Setup | `setUp()` | Use `TestMain()` or subtests |
| Run tests | `phpunit` | `go test ./...` |
| Verbose | `phpunit -v` | `go test -v` |
| Coverage | `phpunit --coverage` | `go test -cover` |
| Benchmark | N/A | `func BenchmarkX(b *testing.B)` |

## Common Commands

| Task | PHP | Go |
|------|-----|-----|
| Run | `php script.php` | `go run main.go` |
| Build | N/A | `go build` |
| Install deps | `composer install` | `go mod download` |
| Add dep | `composer require pkg` | `go get pkg` |
| Update deps | `composer update` | `go get -u ./...` |
| Format | `php-cs-fixer fix` | `go fmt ./...` |
| Lint | `phpstan analyse` | `go vet ./...` |
| Test | `phpunit` | `go test ./...` |
| Docs | N/A | `go doc pkg` |

---

*Keep this cheat sheet handy during your first months with Go!*
