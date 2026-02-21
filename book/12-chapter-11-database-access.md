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

## NoSQL Databases

Symfony developers often use Doctrine ODM for MongoDB or symfony/cache for Redis. Go's NoSQL story is simpler—direct client libraries without heavy abstractions.

### MongoDB

MongoDB is popular for document storage. PHP uses the MongoDB extension or Doctrine ODM:

```php
// Doctrine ODM
#[Document]
class Product
{
    #[Id]
    private string $id;

    #[Field]
    private string $name;

    #[EmbedMany(targetDocument: Review::class)]
    private Collection $reviews;
}

$product = $dm->find(Product::class, $id);
```

Go uses the official MongoDB driver directly:

```go
import "go.mongodb.org/mongo-driver/mongo"

type Product struct {
    ID      primitive.ObjectID `bson:"_id,omitempty"`
    Name    string             `bson:"name"`
    Reviews []Review           `bson:"reviews"`
}

func connectMongoDB(ctx context.Context) (*mongo.Client, error) {
    client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        return nil, err
    }

    // Verify connection
    if err := client.Ping(ctx, nil); err != nil {
        return nil, err
    }

    return client, nil
}

func findProduct(ctx context.Context, db *mongo.Database, id primitive.ObjectID) (*Product, error) {
    collection := db.Collection("products")

    var product Product
    err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&product)
    if err == mongo.ErrNoDocuments {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, err
    }

    return &product, nil
}

func insertProduct(ctx context.Context, db *mongo.Database, product *Product) error {
    collection := db.Collection("products")
    result, err := collection.InsertOne(ctx, product)
    if err != nil {
        return err
    }
    product.ID = result.InsertedID.(primitive.ObjectID)
    return nil
}
```

### Complex Queries

```go
// Find with filter, sort, and limit
func findActiveProducts(ctx context.Context, db *mongo.Database, category string, limit int64) ([]Product, error) {
    collection := db.Collection("products")

    filter := bson.M{
        "category": category,
        "active":   true,
    }
    opts := options.Find().
        SetSort(bson.D{{Key: "created_at", Value: -1}}).
        SetLimit(limit)

    cursor, err := collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var products []Product
    if err := cursor.All(ctx, &products); err != nil {
        return nil, err
    }

    return products, nil
}

// Aggregation pipeline
func getProductStats(ctx context.Context, db *mongo.Database) ([]CategoryStats, error) {
    collection := db.Collection("products")

    pipeline := mongo.Pipeline{
        {{Key: "$match", Value: bson.M{"active": true}}},
        {{Key: "$group", Value: bson.M{
            "_id":   "$category",
            "count": bson.M{"$sum": 1},
            "avg":   bson.M{"$avg": "$price"},
        }}},
        {{Key: "$sort", Value: bson.M{"count": -1}}},
    }

    cursor, err := collection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var stats []CategoryStats
    return stats, cursor.All(ctx, &stats)
}
```

### Redis

Symfony uses symfony/cache or predis for Redis:

```php
// Symfony Cache
$cache = new RedisAdapter($redis);
$item = $cache->getItem('user_'.$id);
if (!$item->isHit()) {
    $item->set($user);
    $item->expiresAfter(3600);
    $cache->save($item);
}
```

Go's go-redis library provides direct access:

```go
import "github.com/go-redis/redis/v8"

func newRedisClient() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
}

// Basic operations
func cacheUser(ctx context.Context, rdb *redis.Client, user *User) error {
    data, err := json.Marshal(user)
    if err != nil {
        return err
    }

    key := fmt.Sprintf("user:%d", user.ID)
    return rdb.Set(ctx, key, data, time.Hour).Err()
}

func getCachedUser(ctx context.Context, rdb *redis.Client, id int) (*User, error) {
    key := fmt.Sprintf("user:%d", id)
    data, err := rdb.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, ErrNotFound  // Cache miss
    }
    if err != nil {
        return nil, err
    }

    var user User
    return &user, json.Unmarshal(data, &user)
}

// Cache-aside pattern
func getUser(ctx context.Context, rdb *redis.Client, db *sql.DB, id int) (*User, error) {
    // Try cache first
    user, err := getCachedUser(ctx, rdb, id)
    if err == nil {
        return user, nil
    }
    if err != ErrNotFound {
        // Log cache error but continue to database
        log.Printf("cache error: %v", err)
    }

    // Fetch from database
    user, err = fetchUserFromDB(ctx, db, id)
    if err != nil {
        return nil, err
    }

    // Cache for next time (fire and forget)
    go cacheUser(context.Background(), rdb, user)

    return user, nil
}
```

### Redis Data Structures

```go
// Hash for structured data
func cacheUserHash(ctx context.Context, rdb *redis.Client, user *User) error {
    key := fmt.Sprintf("user:%d", user.ID)
    return rdb.HSet(ctx, key,
        "name", user.Name,
        "email", user.Email,
        "role", user.Role,
    ).Err()
}

// Sorted set for leaderboards
func updateScore(ctx context.Context, rdb *redis.Client, userID string, score float64) error {
    return rdb.ZAdd(ctx, "leaderboard", &redis.Z{
        Score:  score,
        Member: userID,
    }).Err()
}

func getTopUsers(ctx context.Context, rdb *redis.Client, limit int64) ([]string, error) {
    return rdb.ZRevRange(ctx, "leaderboard", 0, limit-1).Result()
}

// List for queues (simple alternative to Kafka)
func pushJob(ctx context.Context, rdb *redis.Client, job *Job) error {
    data, _ := json.Marshal(job)
    return rdb.LPush(ctx, "jobs", data).Err()
}

func popJob(ctx context.Context, rdb *redis.Client) (*Job, error) {
    // Block until job available
    result, err := rdb.BRPop(ctx, 0, "jobs").Result()
    if err != nil {
        return nil, err
    }

    var job Job
    return &job, json.Unmarshal([]byte(result[1]), &job)
}
```

## Data Streaming

PHP developers use Symfony Messenger for async messaging:

```php
// Symfony Messenger
$bus->dispatch(new OrderPlaced($orderId));

// Handler
#[AsMessageHandler]
class OrderPlacedHandler
{
    public function __invoke(OrderPlaced $message): void
    {
        // Process order
    }
}
```

Go's concurrency primitives handle in-process messaging, but for distributed streaming, you need tools like Kafka or Pulsar.

### Apache Kafka with Sarama

Kafka is the industry standard for event streaming. Sarama is Go's most popular Kafka client:

```go
import "github.com/Shopify/sarama"

// Producer
func newKafkaProducer(brokers []string) (sarama.SyncProducer, error) {
    config := sarama.NewConfig()
    config.Producer.RequiredAcks = sarama.WaitForAll
    config.Producer.Retry.Max = 5
    config.Producer.Return.Successes = true

    return sarama.NewSyncProducer(brokers, config)
}

func publishEvent(producer sarama.SyncProducer, topic string, event interface{}) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }

    msg := &sarama.ProducerMessage{
        Topic: topic,
        Value: sarama.ByteEncoder(data),
    }

    partition, offset, err := producer.SendMessage(msg)
    if err != nil {
        return err
    }

    log.Printf("Published to partition %d at offset %d", partition, offset)
    return nil
}

// Consumer
func consumeEvents(ctx context.Context, brokers []string, topic, groupID string, handler func([]byte) error) error {
    config := sarama.NewConfig()
    config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
    config.Consumer.Offsets.Initial = sarama.OffsetNewest

    group, err := sarama.NewConsumerGroup(brokers, groupID, config)
    if err != nil {
        return err
    }
    defer group.Close()

    consumer := &consumerHandler{handler: handler}

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := group.Consume(ctx, []string{topic}, consumer); err != nil {
                return err
            }
        }
    }
}

type consumerHandler struct {
    handler func([]byte) error
}

func (h *consumerHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    for msg := range claim.Messages() {
        if err := h.handler(msg.Value); err != nil {
            log.Printf("Error processing message: %v", err)
            continue
        }
        session.MarkMessage(msg, "")
    }
    return nil
}
```

### Event-Driven Architecture

```go
// Domain events
type OrderPlaced struct {
    OrderID   string    `json:"order_id"`
    UserID    string    `json:"user_id"`
    Total     float64   `json:"total"`
    Timestamp time.Time `json:"timestamp"`
}

type OrderShipped struct {
    OrderID    string    `json:"order_id"`
    TrackingNo string    `json:"tracking_no"`
    Timestamp  time.Time `json:"timestamp"`
}

// Event publisher service
type EventPublisher struct {
    producer sarama.SyncProducer
}

func (p *EventPublisher) PublishOrderPlaced(order *Order) error {
    event := OrderPlaced{
        OrderID:   order.ID,
        UserID:    order.UserID,
        Total:     order.Total,
        Timestamp: time.Now(),
    }
    return publishEvent(p.producer, "orders.placed", event)
}

// Event consumer service
func startOrderProcessor(ctx context.Context, brokers []string) error {
    return consumeEvents(ctx, brokers, "orders.placed", "order-processor", func(data []byte) error {
        var event OrderPlaced
        if err := json.Unmarshal(data, &event); err != nil {
            return err
        }

        log.Printf("Processing order %s for user %s", event.OrderID, event.UserID)
        // Process the order...
        return nil
    })
}
```

### Redis Streams

For simpler streaming needs, Redis Streams provides a lightweight alternative:

```go
// Producer
func publishToStream(ctx context.Context, rdb *redis.Client, stream string, event interface{}) error {
    data, _ := json.Marshal(event)

    return rdb.XAdd(ctx, &redis.XAddArgs{
        Stream: stream,
        Values: map[string]interface{}{
            "data": data,
        },
    }).Err()
}

// Consumer with consumer groups
func consumeStream(ctx context.Context, rdb *redis.Client, stream, group, consumer string, handler func([]byte) error) error {
    // Create consumer group if not exists
    rdb.XGroupCreateMkStream(ctx, stream, group, "0")

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
                Group:    group,
                Consumer: consumer,
                Streams:  []string{stream, ">"},
                Count:    10,
                Block:    time.Second,
            }).Result()

            if err == redis.Nil {
                continue
            }
            if err != nil {
                return err
            }

            for _, stream := range streams {
                for _, msg := range stream.Messages {
                    data := []byte(msg.Values["data"].(string))
                    if err := handler(data); err != nil {
                        log.Printf("Error: %v", err)
                        continue
                    }
                    rdb.XAck(ctx, stream.Stream, group, msg.ID)
                }
            }
        }
    }
}
```

### Choosing a Streaming Solution

| Feature | Kafka | Redis Streams | Channels |
|---------|-------|---------------|----------|
| Persistence | Disk-based | Optional | None |
| Scalability | Massive | Moderate | Single process |
| Ordering | Per partition | Per stream | Per channel |
| Consumer groups | Yes | Yes | No |
| Complexity | High | Medium | Low |
| Use case | Large-scale events | Simple streaming | In-process only |

For PHP developers: Kafka replaces RabbitMQ/Symfony Messenger for high-throughput scenarios. Redis Streams is similar to Symfony Messenger with Redis transport. Channels are for goroutine coordination only.

## Summary

- **`database/sql`** provides connection pooling and basic queries
- **SQLC** generates type-safe code from SQL
- **sqlx** adds struct scanning to `database/sql`
- **GORM** exists but many prefer explicit SQL
- **Migrations** use tools like Goose or golang-migrate
- **Transactions** are explicit with `BeginTx`, `Commit`, `Rollback`
- **MongoDB** uses the official driver with BSON tags
- **Redis** serves as cache, session store, and simple queue
- **Kafka** handles distributed event streaming at scale
- **Redis Streams** provides lightweight streaming for simpler needs

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

9. **MongoDB CRUD**: Implement a product catalogue with MongoDB. Include embedded reviews and aggregation for statistics.

10. **Redis Cache-Aside**: Implement the cache-aside pattern for a user service. Handle cache misses, updates, and invalidation.

11. **Kafka Event System**: Build a simple order processing system with Kafka. Publish OrderPlaced events and consume them in a separate service.

12. **Redis Streams Worker**: Create a job queue using Redis Streams with consumer groups. Handle failures and acknowledgements.
