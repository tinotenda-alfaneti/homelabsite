package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

type AuthMiddleware struct {
	sessions  map[string]time.Time
	sessionMu sync.RWMutex
	adminUser string
	adminPass string
}

func NewAuthMiddleware(adminUser, adminPass string) *AuthMiddleware {
	am := &AuthMiddleware{
		sessions:  make(map[string]time.Time),
		adminUser: adminUser,
		adminPass: adminPass,
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
	return username == am.adminUser && password == am.adminPass
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
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
