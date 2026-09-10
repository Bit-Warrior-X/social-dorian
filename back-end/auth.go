package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "dorian_session"
	authSessionTTL    = 7 * 24 * time.Hour
	loginMaxAttempts  = 8
	loginWindow       = 15 * time.Minute
)

type ctxKey int

const userContextKey ctxKey = 1

type AuthUser struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type AuthStore struct {
	db       *sql.DB
	attempts map[string][]time.Time
	mu       sync.Mutex
}

func newAuthStore(db *sql.DB) *AuthStore {
	return &AuthStore{
		db:       db,
		attempts: map[string][]time.Time{},
	}
}

func (s *AuthStore) ensureSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
		  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
		  email VARCHAR(255) NOT NULL,
		  name VARCHAR(120) NOT NULL DEFAULT '',
		  password_hash VARCHAR(255) NOT NULL,
		  role ENUM('admin', 'member') NOT NULL DEFAULT 'admin',
		  status ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
		  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		  PRIMARY KEY (id),
		  UNIQUE KEY uk_users_email (email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS auth_sessions (
		  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
		  user_id INT UNSIGNED NOT NULL,
		  token_hash CHAR(64) NOT NULL,
		  expires_at DATETIME NOT NULL,
		  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  PRIMARY KEY (id),
		  UNIQUE KEY uk_auth_sessions_token (token_hash),
		  KEY idx_auth_sessions_user (user_id),
		  KEY idx_auth_sessions_expires (expires_at),
		  CONSTRAINT fk_auth_sessions_user
		    FOREIGN KEY (user_id) REFERENCES users (id)
		    ON DELETE CASCADE
		    ON UPDATE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	return err
}

func (s *AuthStore) ensureAdmin() error {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	password := os.Getenv("ADMIN_PASSWORD")
	name := strings.TrimSpace(os.Getenv("ADMIN_NAME"))
	if name == "" {
		name = "Admin"
	}

	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}

	if email == "" || password == "" {
		if count == 0 {
			log.Print("no admin user found; set ADMIN_EMAIL and ADMIN_PASSWORD in .env")
		}
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO users (email, name, password_hash, role, status)
		VALUES (?, ?, ?, 'admin', 'active')
		ON DUPLICATE KEY UPDATE
		  name = VALUES(name),
		  password_hash = VALUES(password_hash),
		  role = 'admin',
		  status = 'active'`,
		email, name, string(hash),
	)
	return err
}

func (s *AuthStore) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || isPublicAPI(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		user, err := s.userFromRequest(r)
		if err != nil {
			log.Printf("auth lookup: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to check session")
			return
		}
		if user == nil {
			writeError(w, http.StatusUnauthorized, "sign in required")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicAPI(path string) bool {
	switch path {
	case "/api/health", "/api/auth/login":
		return true
	default:
		return false
	}
}

func (s *AuthStore) handleAuth(w http.ResponseWriter, r *http.Request) {
	switch strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/auth"), "/") {
	case "login":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleLogin(w, r)
	case "logout":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleLogout(w, r)
	case "me":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleMe(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *AuthStore) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.allowLogin(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many sign-in attempts — try again in a few minutes")
		return
	}

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	var user AuthUser
	var hash, status string
	err := s.db.QueryRow(`
		SELECT id, email, name, role, status, password_hash
		FROM users
		WHERE email = ?
		LIMIT 1`, email).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &status, &hash)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		log.Printf("login lookup: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to sign in")
		return
	}
	if status != "active" {
		writeError(w, http.StatusForbidden, "this account is disabled")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := newAuthToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	expires := time.Now().UTC().Add(authSessionTTL)
	if _, err := s.db.Exec(
		`INSERT INTO auth_sessions (user_id, token_hash, expires_at) VALUES (?, ?, ?)`,
		user.ID, hashToken(token), expires,
	); err != nil {
		log.Printf("create session: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	_, _ = s.db.Exec(`DELETE FROM auth_sessions WHERE expires_at < UTC_TIMESTAMP()`)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(authSessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, user)
}

func (s *AuthStore) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		_, _ = s.db.Exec(`DELETE FROM auth_sessions WHERE token_hash = ?`, hashToken(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *AuthStore) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := s.userFromRequest(r)
	if err != nil {
		log.Printf("me lookup: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to check session")
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *AuthStore) userFromRequest(r *http.Request) (*AuthUser, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil, nil
	}

	var user AuthUser
	var status string
	err = s.db.QueryRow(`
		SELECT u.id, u.email, u.name, u.role, u.status
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > UTC_TIMESTAMP()
		LIMIT 1`, hashToken(cookie.Value)).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if status != "active" {
		return nil, nil
	}
	return &user, nil
}

func (s *AuthStore) allowLogin(ip string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	kept := make([]time.Time, 0, len(s.attempts[ip]))
	for _, at := range s.attempts[ip] {
		if now.Sub(at) < loginWindow {
			kept = append(kept, at)
		}
	}
	if len(kept) >= loginMaxAttempts {
		s.attempts[ip] = kept
		return false
	}
	s.attempts[ip] = append(kept, now)
	return true
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newAuthToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	return proto == "https"
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func currentUser(r *http.Request) *AuthUser {
	user, _ := r.Context().Value(userContextKey).(*AuthUser)
	return user
}
