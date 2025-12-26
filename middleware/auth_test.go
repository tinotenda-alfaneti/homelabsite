package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestNewAuthMiddleware(t *testing.T) {
	am := NewAuthMiddleware("testuser", "testpass")

	if am.adminUser != "testuser" {
		t.Errorf("Expected adminUser to be 'testuser', got '%s'", am.adminUser)
	}

	// Verify password was hashed
	err := bcrypt.CompareHashAndPassword([]byte(am.passwordHash), []byte("testpass"))
	if err != nil {
		t.Errorf("Password hash verification failed: %v", err)
	}
}

func TestValidateCredentials(t *testing.T) {
	am := NewAuthMiddleware("admin", "password123")

	tests := []struct {
		name     string
		username string
		password string
		want     bool
	}{
		{"Valid credentials", "admin", "password123", true},
		{"Invalid username", "wrong", "password123", false},
		{"Invalid password", "admin", "wrong", false},
		{"Both invalid", "wrong", "wrong", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := am.ValidateCredentials(tt.username, tt.password)
			if got != tt.want {
				t.Errorf("ValidateCredentials(%s, %s) = %v, want %v", tt.username, tt.password, got, tt.want)
			}
		})
	}
}

func TestCreateSession(t *testing.T) {
	am := NewAuthMiddleware("admin", "password")

	token, expiry := am.CreateSession()

	if token == "" {
		t.Error("Expected non-empty session token")
	}

	if expiry.Before(time.Now()) {
		t.Error("Expected expiry to be in the future")
	}

	// Check session was stored
	am.sessionMu.RLock()
	_, exists := am.sessions[token]
	am.sessionMu.RUnlock()

	if !exists {
		t.Error("Expected session to be stored")
	}
}

func TestDeleteSession(t *testing.T) {
	am := NewAuthMiddleware("admin", "password")
	token, _ := am.CreateSession()

	// Verify session exists
	am.sessionMu.RLock()
	_, exists := am.sessions[token]
	am.sessionMu.RUnlock()
	if !exists {
		t.Fatal("Session should exist before deletion")
	}

	// Delete session
	am.DeleteSession(token)

	// Verify session was deleted
	am.sessionMu.RLock()
	_, exists = am.sessions[token]
	am.sessionMu.RUnlock()
	if exists {
		t.Error("Session should be deleted")
	}
}

func TestRequireAuth(t *testing.T) {
	am := NewAuthMiddleware("admin", "password")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("authenticated")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	})

	wrapped := am.RequireAuth(testHandler)

	tests := []struct {
		name           string
		setupSession   bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "No session cookie",
			setupSession:   false,
			expectedStatus: http.StatusFound, // Redirect
			expectedBody:   "",
		},
		{
			name:           "Valid session",
			setupSession:   true,
			expectedStatus: http.StatusOK,
			expectedBody:   "authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/admin", nil)
			if tt.setupSession {
				token, _ := am.CreateSession()
				req.AddCookie(&http.Cookie{
					Name:  "session_token",
					Value: token,
				})
			}

			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedBody != "" && rr.Body.String() != tt.expectedBody {
				t.Errorf("Expected body '%s', got '%s'", tt.expectedBody, rr.Body.String())
			}
		})
	}
}
