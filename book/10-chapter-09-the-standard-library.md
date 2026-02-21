# Chapter 9: The Standard Library Is Your Framework

PHP developers reach for frameworks instinctively. Symfony, Laravel, Slim—these provide the foundation for most PHP applications. Go developers often don't use frameworks at all. The standard library is comprehensive enough for most needs.

## Why Go Doesn't Need Symfony

Symfony provides:
- HTTP handling (HttpFoundation, HttpKernel)
- Routing (Routing component)
- Dependency injection (DependencyInjection component)
- Configuration (Config, Yaml, Dotenv)
- Serialisation (Serializer)
- Validation (Validator)
- Database abstraction (Doctrine DBAL)
- Templating (Twig)
- Caching (Cache)
- Logging (Monolog integration)

Go's standard library provides equivalents for most of these:

| Symfony Component | Go Standard Library |
|-------------------|---------------------|
| HttpFoundation    | `net/http`          |
| Routing           | `net/http` (1.22+)  |
| Serializer        | `encoding/json`, `encoding/xml` |
| Validator         | (none—use patterns or packages) |
| Doctrine DBAL     | `database/sql`      |
| Twig              | `html/template`, `text/template` |
| Cache             | (none—use packages) |
| Monolog           | `log/slog`          |

The gaps are intentional. Go philosophy says: if it can't be done well generically, don't include it.

## `net/http` vs Symfony HttpFoundation

Symfony wraps PHP's superglobals in objects:

```php
use Symfony\Component\HttpFoundation\Request;
use Symfony\Component\HttpFoundation\Response;

$request = Request::createFromGlobals();
$name = $request->query->get('name', 'World');

$response = new Response(
    "Hello, $name!",
    Response::HTTP_OK,
    ['Content-Type' => 'text/plain']
);
$response->send();
```

Go's `net/http` provides similar abstractions:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    if name == "" {
        name = "World"
    }

    w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "Hello, %s!", name)
}

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":8080", nil)
}
```

### Request Object Comparison

```php
// Symfony Request
$request->getMethod();           // GET, POST, etc.
$request->getPathInfo();         // /users/123
$request->query->get('page');    // Query params
$request->request->get('name');  // POST body
$request->headers->get('Accept'); // Headers
$request->getContent();          // Raw body
```

```go
// Go http.Request
r.Method              // GET, POST, etc.
r.URL.Path            // /users/123
r.URL.Query().Get("page")  // Query params
r.FormValue("name")   // POST body (form-encoded)
r.Header.Get("Accept") // Headers
io.ReadAll(r.Body)    // Raw body
```

### Response Writing

Symfony builds a Response object, then sends it.

Go writes directly to the ResponseWriter:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Set headers before writing body
    w.Header().Set("Content-Type", "application/json")

    // WriteHeader sets the status (optional—defaults to 200)
    w.WriteHeader(http.StatusCreated)

    // Write body (implements io.Writer)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

The streaming model is different—you can't modify headers after writing body bytes.

## `encoding/json` vs Symfony Serializer

Symfony Serializer is powerful and complex:

```php
use Symfony\Component\Serializer\SerializerInterface;

class UserController
{
    public function show(User $user, SerializerInterface $serializer): Response
    {
        return new Response(
            $serializer->serialize($user, 'json', ['groups' => ['public']]),
            200,
            ['Content-Type' => 'application/json']
        );
    }
}
```

Features include:
- Serialisation groups
- Custom normalisers
- Multiple formats (JSON, XML, CSV)
- Object denormalisation
- Circular reference handling

Go's `encoding/json` is simpler:

```go
type User struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"-"`  // Excluded
}

func showUser(w http.ResponseWriter, r *http.Request) {
    user := getUserFromDB()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```

Struct tags control JSON encoding:
- `json:"name"` — field name in JSON
- `json:"-"` — exclude field
- `json:"name,omitempty"` — omit if zero value
- `json:",string"` — encode number as string

### What Go Lacks

- **Serialisation groups**: Use different structs or custom marshalling
- **Custom normalisers**: Implement `json.Marshaler` interface
- **Circular references**: Handle manually (or redesign)

```go
// Custom JSON marshalling
type User struct {
    BirthDate time.Time
}

func (u User) MarshalJSON() ([]byte, error) {
    type Alias User
    return json.Marshal(&struct {
        Alias
        BirthDate string `json:"birth_date"`
    }{
        Alias:     Alias(u),
        BirthDate: u.BirthDate.Format("2006-01-02"),
    })
}
```

## `database/sql` vs Doctrine DBAL

Doctrine DBAL provides:
- Query builders
- Schema abstraction
- Multiple database support
- Connection pooling
- Type mapping

Go's `database/sql` is lower-level:

```go
import (
    "database/sql"
    _ "github.com/lib/pq"  // PostgreSQL driver
)

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/dbname")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Query
    rows, err := db.QueryContext(ctx, "SELECT id, name FROM users WHERE active = $1", true)
    if err != nil {
        return err
    }
    defer rows.Close()

    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Name); err != nil {
            return err
        }
        users = append(users, u)
    }
}
```

### Key Differences

**Connection Pooling**: Built into `database/sql`—it manages a pool automatically.

**Type Safety**: You scan into Go types directly. No result arrays.

**No Query Builder**: Write SQL strings. Use libraries like `sqlx` or `squirrel` if needed.

**Prepared Statements**: Built-in via `db.Prepare()` or automatic with `db.Query()`.

```go
// Single row
var name string
err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = $1", id).Scan(&name)
if err == sql.ErrNoRows {
    // Not found
}

// Execute (INSERT, UPDATE, DELETE)
result, err := db.ExecContext(ctx, "UPDATE users SET active = $1 WHERE id = $2", true, id)
rowsAffected, _ := result.RowsAffected()
```

## `html/template` vs Twig

Twig provides:
```twig
{% extends 'base.html.twig' %}

{% block content %}
    <h1>{{ user.name }}</h1>
    {% for post in posts %}
        <article>{{ post.title | upper }}</article>
    {% endfor %}
{% endblock %}
```

Go's `html/template` is simpler:

```go
const tmpl = `
<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body>
    <h1>{{.User.Name}}</h1>
    {{range .Posts}}
        <article>{{.Title}}</article>
    {{end}}
</body>
</html>
`

func handler(w http.ResponseWriter, r *http.Request) {
    t := template.Must(template.New("page").Parse(tmpl))
    t.Execute(w, map[string]any{
        "Title": "My Page",
        "User":  user,
        "Posts": posts,
    })
}
```

### Key Differences

**Auto-escaping**: `html/template` automatically escapes HTML. Twig does too.

**Inheritance**: No built-in template inheritance. Use `template.ParseFiles()` with `define`/`template` blocks:

```go
// base.html
{{define "base"}}
<!DOCTYPE html>
<html>
<body>{{template "content" .}}</body>
</html>
{{end}}

// page.html
{{define "content"}}
<h1>{{.Title}}</h1>
{{end}}
```

**Filters**: Define functions:

```go
funcs := template.FuncMap{
    "upper": strings.ToUpper,
}
t := template.New("page").Funcs(funcs).Parse(`{{.Name | upper}}`)
```

## When to Reach for Third-Party Packages

The standard library covers 80% of needs. Reach for packages when:

### Routing Complexity

Go 1.22 added method-based routing to `http.ServeMux`, but for complex routing, consider:
- `chi` — lightweight, idiomatic
- `gorilla/mux` — feature-rich (deprecated but stable)
- `gin` — fast, full-featured

### Validation

No standard validation library. Consider:
- `go-playground/validator` — struct tag validation

```go
type User struct {
    Email string `validate:"required,email"`
    Age   int    `validate:"gte=0,lte=130"`
}
```

### Caching

No standard caching. Consider:
- `patrickmn/go-cache` — in-memory
- Redis clients for distributed caching

### Configuration

No standard config loading. Consider:
- `spf13/viper` — full-featured
- `kelseyhightower/envconfig` — environment variables

### Database

For more than raw SQL:
- `sqlx` — extensions to `database/sql`
- `GORM` — full ORM
- `sqlc` — generates Go from SQL

## Summary

- **Go's standard library** is comprehensive for HTTP, JSON, SQL, and templating
- **`net/http`** provides complete HTTP server/client functionality
- **`encoding/json`** handles serialisation with struct tags
- **`database/sql`** provides connection pooling and query execution
- **`html/template`** offers auto-escaped templating
- **Third-party packages** fill gaps (validation, caching, config)

---

## Exercises

1. **HTTP Server**: Build a simple REST API using only `net/http`. Implement GET, POST, PUT, DELETE for a resource. No external packages.

2. **JSON Customisation**: Create a struct with a `time.Time` field and custom JSON format. Implement `MarshalJSON` and `UnmarshalJSON`.

3. **Database CRUD**: Implement full CRUD operations using `database/sql`. Handle `sql.ErrNoRows` appropriately.

4. **Template Composition**: Build a multi-page site using `html/template` with a shared layout. Implement a custom template function.

5. **Middleware Stack**: Create logging and authentication middleware using only `net/http`. Chain them together.

6. **Symfony Replacement**: Take a simple Symfony controller. Rewrite it using only Go's standard library. List what you miss.

7. **Package Evaluation**: For validation, routing, and caching, evaluate two packages each. Recommend one per category with justification.

8. **Standard Library Limits**: Identify three things you commonly do in Symfony that require third-party Go packages. Evaluate whether the packages are worth it.
