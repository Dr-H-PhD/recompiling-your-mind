// Package main demonstrates database access patterns in Go.
// For PHP developers: This replaces Doctrine DBAL and partially Doctrine ORM.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// User represents a user entity.
// PHP equivalent: App\Entity\User with Doctrine annotations
type User struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepository handles user database operations.
// PHP equivalent: App\Repository\UserRepository extends ServiceEntityRepository
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new repository with a database connection.
// PHP equivalent: Dependency injection via services.yaml
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindAll retrieves all users.
// PHP equivalent: $this->findAll() or $this->createQueryBuilder('u')->getQuery()->getResult()
func (r *UserRepository) FindAll(ctx context.Context) ([]User, error) {
	// Use context for cancellation
	// PHP: No equivalent - requests can't be cancelled mid-query
	query := `SELECT id, name, email, created_at, updated_at FROM users ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close() // Always close rows!

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		users = append(users, u)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return users, nil
}

// FindByID retrieves a user by ID.
// PHP equivalent: $this->find($id)
func (r *UserRepository) FindByID(ctx context.Context, id int64) (*User, error) {
	query := `SELECT id, name, email, created_at, updated_at FROM users WHERE id = $1`

	var u User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// PHP equivalent: return null from find()
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return &u, nil
}

// FindByEmail retrieves a user by email.
// PHP equivalent: $this->findOneBy(['email' => $email])
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, name, email, created_at, updated_at FROM users WHERE email = $1`

	var u User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return &u, nil
}

// Create inserts a new user.
// PHP equivalent: $em->persist($user); $em->flush();
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (name, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// RETURNING id gives us the generated ID
	// PHP equivalent: Doctrine sets the ID on the entity after flush()
	err := r.db.QueryRowContext(ctx, query, user.Name, user.Email, user.CreatedAt, user.UpdatedAt).
		Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}

	return nil
}

// Update modifies an existing user.
// PHP equivalent: $em->flush() after modifying entity
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, updated_at = $3
		WHERE id = $4
	`

	user.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query, user.Name, user.Email, user.UpdatedAt, user.ID)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	// Check if any row was actually updated
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("user not found: %d", user.ID)
	}

	return nil
}

// Delete removes a user by ID.
// PHP equivalent: $em->remove($user); $em->flush();
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("user not found: %d", id)
	}

	return nil
}

// WithTransaction executes operations within a transaction.
// PHP equivalent: $em->transactional(function() { ... })
func (r *UserRepository) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		// Rollback on error
		// PHP: Doctrine rolls back on exception
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original: %w)", rbErr, err)
		}
		return err
	}

	// Commit on success
	// PHP equivalent: $em->flush() at the end
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

func main() {
	// Get database URL from environment
	// PHP equivalent: DATABASE_URL in .env
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/demo?sslmode=disable"
	}

	// Open database connection
	// PHP equivalent: Doctrine DBAL connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	// PHP: PHP-FPM creates new connections per request (or uses persistent connections)
	// Go: Single pool shared across all goroutines
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Database connected successfully")

	// Create repository
	repo := NewUserRepository(db)

	// Example usage
	user := &User{
		Name:  "Test User",
		Email: "test@example.com",
	}

	if err := repo.Create(ctx, user); err != nil {
		log.Printf("Create user: %v", err)
	} else {
		log.Printf("Created user: %+v", user)
	}

	users, err := repo.FindAll(ctx)
	if err != nil {
		log.Printf("Find all: %v", err)
	} else {
		log.Printf("Found %d users", len(users))
	}
}
