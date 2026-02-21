// Package main demonstrates basic HTTP server patterns in Go.
// For PHP developers: This replaces Symfony's HttpFoundation and routing components.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// Response represents a JSON response structure.
// PHP equivalent: Similar to a DTO or a Symfony Serializer normalizable object.
type Response struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// User represents a user entity.
// PHP equivalent: App\Entity\User in Doctrine.
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	// Create a new ServeMux (router)
	// PHP equivalent: Symfony's Router component
	mux := http.NewServeMux()

	// Register routes
	// PHP equivalent: routes.yaml or @Route annotations
	mux.HandleFunc("GET /", homeHandler)
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /users", listUsersHandler)
	mux.HandleFunc("GET /users/{id}", getUserHandler)
	mux.HandleFunc("POST /users", createUserHandler)

	// Wrap with middleware
	// PHP equivalent: Symfony's EventListener or Middleware pattern
	handler := loggingMiddleware(corsMiddleware(mux))

	// Get port from environment or default
	// PHP equivalent: $_ENV['PORT'] or getenv('PORT')
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create server with timeouts
	// PHP equivalent: No direct equivalent - PHP-FPM handles this
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on port %s", port)

	// Start server (blocks)
	// PHP equivalent: php-fpm or built-in server
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// homeHandler handles the root route.
// PHP equivalent: HomeController::index()
func homeHandler(w http.ResponseWriter, r *http.Request) {
	resp := Response{
		Message:   "Welcome to the Go HTTP server",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

// healthHandler provides a health check endpoint.
// PHP equivalent: Symfony's HealthCheck bundle
func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// Sample in-memory data store
// PHP equivalent: Doctrine repository with cached entities
var users = []User{
	{ID: 1, Name: "Alice", Email: "alice@example.com"},
	{ID: 2, Name: "Bob", Email: "bob@example.com"},
}

// listUsersHandler returns all users.
// PHP equivalent: UserController::list() with UserRepository::findAll()
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, users)
}

// getUserHandler returns a single user by ID.
// PHP equivalent: UserController::show($id) with ParamConverter
func getUserHandler(w http.ResponseWriter, r *http.Request) {
	// Go 1.22+ pattern matching
	// PHP equivalent: $request->attributes->get('id')
	id := r.PathValue("id")

	for _, user := range users {
		if string(rune(user.ID+'0')) == id { // Simple conversion for demo
			writeJSON(w, http.StatusOK, user)
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
}

// createUserHandler creates a new user.
// PHP equivalent: UserController::create() with form validation
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User

	// Decode JSON body
	// PHP equivalent: $serializer->deserialize($request->getContent(), User::class, 'json')
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	// Validate
	// PHP equivalent: $validator->validate($user)
	if user.Name == "" || user.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Name and email required"})
		return
	}

	// Assign ID and save
	// PHP equivalent: $entityManager->persist($user); $entityManager->flush();
	user.ID = len(users) + 1
	users = append(users, user)

	writeJSON(w, http.StatusCreated, user)
}

// writeJSON is a helper to write JSON responses.
// PHP equivalent: return $this->json($data, $status)
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// loggingMiddleware logs each request.
// PHP equivalent: Symfony's EventListener on kernel.request/kernel.response
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// corsMiddleware adds CORS headers.
// PHP equivalent: NelmioCorsBundle or manual headers in EventListener
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
