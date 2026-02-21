// Package main is the Go service in a PHP-to-Go migration.
// This service handles new endpoints while PHP handles legacy.
package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Config holds application configuration.
// PHP equivalent: .env + services.yaml parameters
type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	RedisURL    string
}

// LoadConfig loads configuration from environment.
// PHP equivalent: DotEnv + Symfony configuration
func LoadConfig() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "shared-secret-between-php-and-go"),
		RedisURL:    getEnv("REDIS_URL", "redis://redis:6379"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// User represents a user (same schema as PHP entity).
// IMPORTANT: Must match PHP's User entity exactly for shared DB
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// API handles HTTP requests.
type API struct {
	db  *sql.DB
	cfg Config
}

func NewAPI(db *sql.DB, cfg Config) *API {
	return &API{db: db, cfg: cfg}
}

// HealthHandler returns service health.
// This is the first endpoint to route to Go.
func (api *API) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database
	ctx := r.Context()
	if err := api.db.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status":   "unhealthy",
			"database": "down",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "healthy",
		"service":  "go-api",
		"database": "up",
	})
}

// ListUsersV2 is the new Go implementation of user listing.
// PHP equivalent: UserController::index() but with improvements
func (api *API) ListUsersV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Same query as PHP, same table
	query := `SELECT id, name, email, created_at FROM users ORDER BY created_at DESC LIMIT 100`

	rows, err := api.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Query error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Database error"})
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		users = append(users, u)
	}

	// Add Go-specific headers for debugging during migration
	w.Header().Set("X-Served-By", "go-service")
	w.Header().Set("X-Migration-Phase", "2")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  users,
		"count": len(users),
		"meta": map[string]string{
			"version": "v2",
			"engine":  "go",
		},
	})
}

// CreateUserV2 creates a user (Go implementation).
// PHP equivalent: UserController::create()
func (api *API) CreateUserV2(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	// Insert into shared database
	// PHP and Go both write to the same table
	ctx := r.Context()
	query := `INSERT INTO users (name, email, created_at) VALUES ($1, $2, $3) RETURNING id`

	var id int64
	err := api.db.QueryRowContext(ctx, query, req.Name, req.Email, time.Now()).Scan(&id)
	if err != nil {
		log.Printf("Insert error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
		return
	}

	w.Header().Set("X-Served-By", "go-service")
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      id,
		"message": "User created by Go service",
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func main() {
	cfg := LoadConfig()

	// Connect to shared database
	// Same database as PHP - critical for migration
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	log.Println("Connected to shared database")

	api := NewAPI(db, cfg)

	// Register routes
	mux := http.NewServeMux()

	// Health check - first endpoint migrated to Go
	mux.HandleFunc("GET /health", api.HealthHandler)

	// v2 API - new implementation in Go
	mux.HandleFunc("GET /api/v2/users", api.ListUsersV2)
	mux.HandleFunc("POST /api/v2/users", api.CreateUserV2)

	// Note: /api/v1/* routes are still handled by PHP via nginx

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("Go service starting on :%s", cfg.Port)
	log.Println("Routes: /health, /api/v2/*")
	log.Println("Legacy routes (/api/v1/*) handled by PHP")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
