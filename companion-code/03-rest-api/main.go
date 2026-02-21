// Package main demonstrates a complete REST API in Go.
// For PHP developers: This combines Symfony's Controller, Security, and Validator.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// --- Errors ---

var (
	ErrNotFound      = errors.New("resource not found")
	ErrUnauthorised  = errors.New("unauthorised")
	ErrValidation    = errors.New("validation failed")
	ErrInternalError = errors.New("internal server error")
)

// APIError represents a structured error response.
// PHP equivalent: Symfony's ApiProblem or custom error normaliser
type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// --- Models ---

// User represents a user in the system.
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Never serialise password
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest is the request body for creating a user.
// PHP equivalent: Form type or request DTO
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks the request is valid.
// PHP equivalent: Symfony Validator constraints on entity
func (r *CreateUserRequest) Validate() map[string]string {
	errs := make(map[string]string)

	if strings.TrimSpace(r.Name) == "" {
		errs["name"] = "Name is required"
	} else if len(r.Name) > 100 {
		errs["name"] = "Name must be 100 characters or less"
	}

	if strings.TrimSpace(r.Email) == "" {
		errs["email"] = "Email is required"
	} else if !strings.Contains(r.Email, "@") {
		errs["email"] = "Invalid email format"
	}

	if len(r.Password) < 8 {
		errs["password"] = "Password must be at least 8 characters"
	}

	return errs
}

// --- In-memory store (replace with database in production) ---

type UserStore struct {
	mu     sync.RWMutex
	users  map[int64]*User
	nextID int64
}

func NewUserStore() *UserStore {
	return &UserStore{
		users:  make(map[int64]*User),
		nextID: 1,
	}
}

func (s *UserStore) Create(user *User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user.ID = s.nextID
	user.CreatedAt = time.Now()
	s.users[user.ID] = user
	s.nextID++
}

func (s *UserStore) GetByID(id int64) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[id]
}

func (s *UserStore) GetAll() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return users
}

// --- Handlers ---

// API groups all handlers and dependencies.
// PHP equivalent: Controller with injected services
type API struct {
	users *UserStore
}

func NewAPI(users *UserStore) *API {
	return &API{users: users}
}

// ListUsers returns all users.
// PHP equivalent: UserController::index() with #[Route('/users', methods: ['GET'])]
func (api *API) ListUsers(w http.ResponseWriter, r *http.Request) {
	users := api.users.GetAll()
	writeJSON(w, http.StatusOK, users)
}

// CreateUser creates a new user.
// PHP equivalent: UserController::create() with form handling
func (api *API) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON in request body")
		return
	}

	// Validate request
	// PHP equivalent: $errors = $validator->validate($dto)
	if errs := req.Validate(); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	user := &User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password, // Would hash in production
	}

	api.users.Create(user)

	writeJSON(w, http.StatusCreated, user)
}

// --- Response helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, APIError{Code: code, Message: message})
}

func writeValidationError(w http.ResponseWriter, details map[string]string) {
	writeJSON(w, http.StatusUnprocessableEntity, APIError{
		Code:    "VALIDATION_ERROR",
		Message: "Validation failed",
		Details: details,
	})
}

// --- Middleware ---

// AuthMiddleware validates JWT tokens.
// PHP equivalent: Symfony Security firewall with JWT
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		// Skip auth for certain paths
		// PHP equivalent: access_control in security.yaml
		if r.URL.Path == "/health" || r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		// Check token
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid token")
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token != "valid-token" { // Would validate JWT in production
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware adds a request ID to each request.
// PHP equivalent: Symfony Uid component with EventListener
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateID() // Would use UUID in production
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func generateID() string {
	return time.Now().Format("20060102150405.000")
}

// RecoveryMiddleware catches panics and returns 500.
// PHP equivalent: Symfony's ExceptionListener
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --- Main ---

func main() {
	// Create dependencies
	users := NewUserStore()
	api := NewAPI(users)

	// Create router
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"service": "user-api", "version": "1.0.0"})
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})
	mux.HandleFunc("GET /users", api.ListUsers)
	mux.HandleFunc("POST /users", api.CreateUser)

	// Apply middleware (order matters - executed in reverse)
	handler := RecoveryMiddleware(RequestIDMiddleware(AuthMiddleware(mux)))

	// Create server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Graceful shutdown
	// PHP: No equivalent - PHP-FPM handles process management
	go func() {
		log.Printf("Server starting on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give active connections time to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
