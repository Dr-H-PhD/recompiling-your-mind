package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHomeHandler tests the home endpoint.
// PHP equivalent: WebTestCase with $client->request('GET', '/')
func TestHomeHandler(t *testing.T) {
	// Create a request
	// PHP equivalent: Request::create('/', 'GET')
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	homeHandler(w, req)

	// Check status code
	// PHP equivalent: $this->assertEquals(200, $client->getResponse()->getStatusCode())
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse response
	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Message == "" {
		t.Error("Expected non-empty message")
	}
}

// TestHealthHandler tests the health check endpoint.
func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", resp["status"])
	}
}

// TestListUsersHandler tests listing all users.
func TestListUsersHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	listUsersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var users []User
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(users) < 2 {
		t.Errorf("Expected at least 2 users, got %d", len(users))
	}
}

// TestCreateUserHandler tests creating a new user.
func TestCreateUserHandler(t *testing.T) {
	// Table-driven tests - Go idiom
	// PHP equivalent: @dataProvider in PHPUnit
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid user",
			body:       `{"name": "Charlie", "email": "charlie@example.com"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing name",
			body:       `{"email": "test@example.com"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			createUserHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

// TestLoggingMiddleware tests that middleware wraps correctly.
func TestLoggingMiddleware(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrapped := loggingMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestCorsMiddleware tests CORS headers are set.
func TestCorsMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS header to be set")
	}
}

// TestCorsMiddlewarePreflight tests OPTIONS preflight handling.
func TestCorsMiddlewarePreflight(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS")
	})

	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
	}
}
