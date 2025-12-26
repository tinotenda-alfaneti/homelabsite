package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthMiddleware struct {
	sessions     map[string]time.Time
	sessionMu    sync.RWMutex
	adminUser    string
	passwordHash string
}

func NewAuthMiddleware(adminUser, adminPass string) *AuthMiddleware {
	// Hash the password on initialization
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	if err != nil {
		panic("Failed to hash admin password: " + err.Error())
	}

	am := &AuthMiddleware{
		sessions:     make(map[string]time.Time),
		adminUser:    adminUser,
		passwordHash: string(hashedPassword),
	}
	go am.cleanupSessions()
	return am
}

func (am *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		am.sessionMu.RLock()
		expiry, exists := am.sessions[cookie.Value]
		am.sessionMu.RUnlock()

		if !exists || time.Now().After(expiry) {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		// Extend session
		am.sessionMu.Lock()
		am.sessions[cookie.Value] = time.Now().Add(24 * time.Hour)
		am.sessionMu.Unlock()

		next(w, r)
	}
}

func (am *AuthMiddleware) ValidateCredentials(username, password string) bool {
	if username != am.adminUser {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(am.passwordHash), []byte(password))
	return err == nil
}

func (am *AuthMiddleware) CreateSession() (string, time.Time) {
	sessionToken := generateSessionToken()
	expiry := time.Now().Add(24 * time.Hour)

	am.sessionMu.Lock()
	am.sessions[sessionToken] = expiry
	am.sessionMu.Unlock()

	return sessionToken, expiry
}

func (am *AuthMiddleware) DeleteSession(token string) {
	am.sessionMu.Lock()
	delete(am.sessions, token)
	am.sessionMu.Unlock()
}

func (am *AuthMiddleware) cleanupSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		am.sessionMu.Lock()
		now := time.Now()
		for token, expiry := range am.sessions {
			if now.After(expiry) {
				delete(am.sessions, token)
			}
		}
		am.sessionMu.Unlock()
	}
}

func generateSessionToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("Error generating session token: %v", err)
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}
