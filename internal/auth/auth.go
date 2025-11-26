package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Session represents a user session
type Session struct {
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionManager manages user sessions
type SessionManager struct {
	sessions      map[string]*Session
	mu            sync.RWMutex
	timeout       time.Duration
	username      string
	passwordHash  string
	secretKey     string
}

// NewSessionManager creates a new session manager
func NewSessionManager(username, passwordHash, secretKey string, timeoutMinutes int) *SessionManager {
	if timeoutMinutes <= 0 {
		timeoutMinutes = 60 // default 1 hour
	}

	sm := &SessionManager{
		sessions:     make(map[string]*Session),
		timeout:      time.Duration(timeoutMinutes) * time.Minute,
		username:     username,
		passwordHash: passwordHash,
		secretKey:    secretKey,
	}

	// Start cleanup goroutine
	go sm.cleanupExpiredSessions()

	return sm
}

// Authenticate checks if username and password are valid
func (sm *SessionManager) Authenticate(username, password string) bool {
	if username != sm.username {
		return false
	}

	// Compare password with bcrypt hash
	err := bcrypt.CompareHashAndPassword([]byte(sm.passwordHash), []byte(password))
	return err == nil
}

// CreateSession creates a new session and returns session ID
func (sm *SessionManager) CreateSession(username string) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Generate random session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return "", err
	}

	// Create session
	session := &Session{
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sm.timeout),
	}

	sm.sessions[sessionID] = session
	return sessionID, nil
}

// ValidateSession checks if session is valid
func (sm *SessionManager) ValidateSession(sessionID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		return false
	}

	return true
}

// RefreshSession extends the session expiry time
func (sm *SessionManager) RefreshSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if exists {
		session.ExpiresAt = time.Now().Add(sm.timeout)
	}
}

// DestroySession removes a session
func (sm *SessionManager) DestroySession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, sessionID)
}

// cleanupExpiredSessions periodically removes expired sessions
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for id, session := range sm.sessions {
			if now.After(session.ExpiresAt) {
				delete(sm.sessions, id)
			}
		}
		sm.mu.Unlock()
	}
}

// generateSessionID generates a random session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// AuthMiddleware returns a middleware that checks authentication
func (sm *SessionManager) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			// No cookie, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Validate session
		if !sm.ValidateSession(cookie.Value) {
			// Invalid session, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Refresh session on each request
		sm.RefreshSession(cookie.Value)

		// Session valid, continue
		next(w, r)
	}
}
