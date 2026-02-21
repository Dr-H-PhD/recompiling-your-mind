# Chapter 11: Database Access

Doctrine is central to Symfony development—entities, repositories, the EntityManager, DQL, migrations. Go's database story is simpler but requires more manual work.

## `database/sql` Fundamentals

Go's `database/sql` is a thin abstraction over database drivers:

```go
import (
    "database/sql"
    _ "github.com/lib/pq"  // PostgreSQL driver
)

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/dbname?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Verify connection
    if err := db.Ping(); err != nil {
        log.Fatal(err)
    }

    // db is now ready for queries
}
```

The `sql.Open` doesn't connect—it prepares the connection pool. `Ping` verifies connectivity.

### Connection Pool Configuration

```go
db.SetMaxOpenConns(25)                 // Maximum open connections
db.SetMaxIdleConns(25)                 // Maximum idle connections
db.SetConnMaxLifetime(5 * time.Minute) // Maximum connection lifetime
```

Doctrine manages pooling via DBAL configuration. Go's pooling is built into `database/sql`.

### Basic Queries

```go
// Query multiple rows
rows, err := db.QueryContext(ctx, "SELECT id, name, email FROM users WHERE active = $1", true)
if err != nil {
    return nil, err
}
defer rows.Close()

var users []User
for rows.Next() {
    var u User
    if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
        return nil, err
    }
    users = append(users, u)
}
if err := rows.Err(); err != nil {
    return nil, err
}

// Query single row
var name string
err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = $1", id).Scan(&name)
if err == sql.ErrNoRows {
    return "", ErrNotFound
}
if err != nil {
    return "", err
}

// Execute (INSERT, UPDATE, DELETE)
result, err := db.ExecContext(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", name, email)
if err != nil {
    return 0, err
}
id, err := result.LastInsertId()  // Or result.RowsAffected()
```

### Always Use Context

```go
// Good: Use context for cancellation and timeouts
rows, err := db.QueryContext(ctx, "SELECT ...")

// Avoid: No context means no cancellation
rows, err := db.Query("SELECT ...")  // Don't use this
```

## Query Builders: SQLC vs Doctrine QueryBuilder

Doctrine QueryBuilder lets you build queries programmatically:

```php
$qb = $this->createQueryBuilder('u')
    ->where('u.active = :active')
    ->andWhere('u.createdAt > :since')
    ->orderBy('u.createdAt', 'DESC')
    ->setParameter('active', true)
    ->setParameter('since', $since);
```

Go has several approaches:

### 1. Raw SQL (Most Common)

```go
query := `
    SELECT id, name, email
    FROM users
    WHERE active = $1
      AND created_at > $2
    ORDER BY created_at DESC
`
rows, err := db.QueryContext(ctx, query, true, since)
```

Many Go developers prefer raw SQL—it's explicit, performant, and your DBA can read it.

### 2. squirrel (Query Builder)

```go
import sq "github.com/Masterminds/squirrel"

query, args, err := sq.Select("id", "name", "email").
    From("users").
    Where(sq.Eq{"active": true}).
    Where(sq.Gt{"created_at": since}).
    OrderBy("created_at DESC").
    PlaceholderFormat(sq.Dollar).
    ToSql()

rows, err := db.QueryContext(ctx, query, args...)
```

### 3. SQLC (Code Generation)

SQLC generates Go code from SQL:

```sql
-- queries.sql
-- name: GetUser :one
SELECT id, name, email FROM users WHERE id = $1;

-- name: ListActiveUsers :many
SELECT id, name, email FROM users WHERE active = true ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email;
```

Run `sqlc generate` to create type-safe Go code:

```go
// Generated code
func (q *Queries) GetUser(ctx context.Context, id int64) (User, error)
func (q *Queries) ListActiveUsers(ctx context.Context) ([]User, error)
func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error)

// Usage
user, err := queries.GetUser(ctx, 42)
```

SQLC provides type safety without runtime overhead—the SQL is still explicit.

## ORMs: GORM vs Doctrine ORM (And Why Many Skip Them)

Doctrine ORM is central to Symfony:

```php
#[Entity]
class User
{
    #[Id]
    #[GeneratedValue]
    #[Column]
    private int $id;

    #[Column]
    private string $name;

    #[OneToMany(targetEntity: Post::class, mappedBy: 'author')]
    private Collection $posts;
}

// Usage
$user = $em->find(User::class, $id);
$em->persist($newUser);
$em->flush();
```

GORM is Go's most popular ORM:

```go
type User struct {
    ID        uint   `gorm:"primaryKey"`
    Name      string
    Posts     []Post `gorm:"foreignKey:AuthorID"`
    CreatedAt time.Time
}

// Usage
var user User
db.First(&user, id)

db.Create(&User{Name: "Alice"})

db.Model(&user).Update("Name", "Bob")
```

### Why Many Go Developers Skip ORMs

Go culture is skeptical of ORMs:

1. **Hidden queries**: ORMs generate SQL you don't see
2. **N+1 problems**: Easy to create accidentally
3. **Learning curve**: Another abstraction to learn
4. **Performance**: Raw SQL is faster
5. **Go's philosophy**: Explicit over magic

The popular alternatives:
- **SQLC**: Type-safe, generated from SQL
- **sqlx**: Extensions to `database/sql` (named parameters, struct scanning)
- **Raw SQL**: Just write queries

### sqlx: A Happy Medium

```go
import "github.com/jmoiron/sqlx"

type User struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

// Scan into struct
var user User
err := db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)

// Scan into slice
var users []User
err := db.SelectContext(ctx, &users, "SELECT * FROM users WHERE active = $1", true)

// Named parameters
query := "INSERT INTO users (name, email) VALUES (:name, :email)"
result, err := db.NamedExecContext(ctx, query, user)
```

## Migrations: Goose vs Doctrine Migrations

Doctrine Migrations generates PHP files:

```php
public function up(Schema $schema): void
{
    $this->addSql('CREATE TABLE users (...)');
}

public function down(Schema $schema): void
{
    $this->addSql('DROP TABLE users');
}
```

Go has several migration tools. Goose is popular:

```sql
-- migrations/001_create_users.sql

-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE users;
```

```bash
goose postgres "postgres://user:pass@localhost/db" up
goose postgres "postgres://user:pass@localhost/db" down
```

Other options:
- **golang-migrate**: Another popular choice
- **atlas**: Schema-based migrations
- **sqlc**: Can manage schemas

## Connection Pooling (Built-In)

Doctrine DBAL configures pooling via environment:

```yaml
doctrine:
    dbal:
        connections:
            default:
                pooled: true
```

Go's `database/sql` pools automatically:

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(1 * time.Minute)
```

The pool:
- Opens connections on demand
- Reuses idle connections
- Closes connections past lifetime
- Blocks when pool is exhausted

## Transactions Without Doctrine's `flush()`

Doctrine batches changes and flushes:

```php
$em->persist($user);
$em->persist($order);
$em->flush();  // Single transaction with all changes
```

Go transactions are explicit:

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()  // Rollback if not committed

_, err = tx.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", name)
if err != nil {
    return err
}

_, err = tx.ExecContext(ctx, "INSERT INTO orders (user_id) VALUES ($1)", userID)
if err != nil {
    return err
}

return tx.Commit()  // Commit only if all succeeded
```

### Transaction Helper

```go
func withTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if err := fn(tx); err != nil {
        return err
    }

    return tx.Commit()
}

// Usage
err := withTx(ctx, db, func(tx *sql.Tx) error {
    // All operations use tx
    return nil
})
```

## Summary

- **`database/sql`** provides connection pooling and basic queries
- **SQLC** generates type-safe code from SQL
- **sqlx** adds struct scanning to `database/sql`
- **GORM** exists but many prefer explicit SQL
- **Migrations** use tools like Goose or golang-migrate
- **Transactions** are explicit with `BeginTx`, `Commit`, `Rollback`

---

## Exercises

1. **Repository Pattern**: Implement a User repository with `database/sql`. Include Find, FindAll, Create, Update, Delete.

2. **SQLC Setup**: Set up SQLC for a simple schema. Write queries and generate code. Compare to hand-written code.

3. **Transaction Handling**: Implement a function that creates a user and their initial preferences in a single transaction.

4. **Connection Pool Tuning**: Write a load test that stresses the connection pool. Experiment with pool settings.

5. **Migration Workflow**: Set up Goose for a project. Create up/down migrations. Practice rolling back.

6. **Query Builder Comparison**: Implement the same complex query using raw SQL, squirrel, and GORM. Compare readability and safety.

7. **N+1 Detection**: Write code that accidentally creates N+1 queries. Then fix it with a JOIN or batch query.

8. **Nullable Handling**: Handle nullable columns using `sql.NullString`, `sql.NullInt64`, etc. Then try with pointer types. Compare approaches.
