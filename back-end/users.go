package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type ManagedUser struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ManagedUserInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

func requireAdmin(w http.ResponseWriter, r *http.Request) *AuthUser {
	user := currentUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return nil
	}
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin only")
		return nil
	}
	return user
}

func (s *AuthStore) handleUsers(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	switch r.Method {
	case http.MethodGet:
		users, err := s.listUsers()
		if err != nil {
			log.Printf("list users: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to list users")
			return
		}
		writeJSON(w, http.StatusOK, users)
	case http.MethodPost:
		input, ok := decodeManagedUser(w, r, true)
		if !ok {
			return
		}
		user, err := s.createUser(input)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, user)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *AuthStore) handleUserByID(w http.ResponseWriter, r *http.Request) {
	admin := requireAdmin(w, r)
	if admin == nil {
		return
	}
	id, err := strconv.Atoi(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	switch r.Method {
	case http.MethodPut:
		input, ok := decodeManagedUser(w, r, false)
		if !ok {
			return
		}
		user, found, err := s.updateUser(id, input, admin.ID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeJSON(w, http.StatusOK, user)
	case http.MethodDelete:
		if id == admin.ID {
			writeError(w, http.StatusBadRequest, "cannot delete your own account")
			return
		}
		found, err := s.deleteUser(id)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func decodeManagedUser(w http.ResponseWriter, r *http.Request, creating bool) (ManagedUserInput, bool) {
	var input ManagedUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return ManagedUserInput{}, false
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Name = strings.TrimSpace(input.Name)
	input.Role = strings.ToLower(strings.TrimSpace(input.Role))
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	if input.Role == "" {
		input.Role = "member"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if creating {
		if input.Email == "" || input.Password == "" {
			writeError(w, http.StatusBadRequest, "email and password are required")
			return ManagedUserInput{}, false
		}
		if _, err := mail.ParseAddress(input.Email); err != nil {
			writeError(w, http.StatusBadRequest, "invalid email")
			return ManagedUserInput{}, false
		}
		if len(input.Password) < 8 {
			writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
			return ManagedUserInput{}, false
		}
	}
	if input.Role != "admin" && input.Role != "member" {
		writeError(w, http.StatusBadRequest, "role must be admin or member")
		return ManagedUserInput{}, false
	}
	if input.Status != "active" && input.Status != "disabled" {
		writeError(w, http.StatusBadRequest, "status must be active or disabled")
		return ManagedUserInput{}, false
	}
	if input.Name == "" && input.Email != "" {
		input.Name = strings.Split(input.Email, "@")[0]
	}
	return input, true
}

func (s *AuthStore) listUsers() ([]ManagedUser, error) {
	rows, err := s.db.Query(`
		SELECT id, email, name, role, status, created_at, updated_at
		FROM users
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ManagedUser, 0)
	for rows.Next() {
		var (
			u                ManagedUser
			createdAt, updAt sql.NullTime
		)
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Status, &createdAt, &updAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			u.CreatedAt = formatDateTime(createdAt.Time)
		}
		if updAt.Valid {
			u.UpdatedAt = formatDateTime(updAt.Time)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *AuthStore) createUser(input ManagedUserInput) (ManagedUser, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return ManagedUser{}, err
	}
	res, err := s.db.Exec(`
		INSERT INTO users (email, name, password_hash, role, status)
		VALUES (?, ?, ?, ?, ?)`,
		input.Email, input.Name, string(hash), input.Role, input.Status,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return ManagedUser{}, errDuplicateEmail
		}
		return ManagedUser{}, err
	}
	id64, _ := res.LastInsertId()
	user, _, err := s.getManagedUser(int(id64))
	return user, err
}

var errDuplicateEmail = errString("email already registered")

type errString string

func (e errString) Error() string { return string(e) }

func (s *AuthStore) getManagedUser(id int) (ManagedUser, bool, error) {
	var (
		u                ManagedUser
		createdAt, updAt sql.NullTime
	)
	err := s.db.QueryRow(`
		SELECT id, email, name, role, status, created_at, updated_at
		FROM users WHERE id = ?`, id).Scan(
		&u.ID, &u.Email, &u.Name, &u.Role, &u.Status, &createdAt, &updAt,
	)
	if err == sql.ErrNoRows {
		return ManagedUser{}, false, nil
	}
	if err != nil {
		return ManagedUser{}, false, err
	}
	if createdAt.Valid {
		u.CreatedAt = formatDateTime(createdAt.Time)
	}
	if updAt.Valid {
		u.UpdatedAt = formatDateTime(updAt.Time)
	}
	return u, true, nil
}

func (s *AuthStore) updateUser(id int, input ManagedUserInput, actorID int) (ManagedUser, bool, error) {
	existing, found, err := s.getManagedUser(id)
	if err != nil || !found {
		return ManagedUser{}, found, err
	}
	email := existing.Email
	name := existing.Name
	role := existing.Role
	status := existing.Status
	if input.Email != "" {
		email = input.Email
	}
	if input.Name != "" {
		name = input.Name
	}
	if input.Role != "" {
		role = input.Role
	}
	if input.Status != "" {
		status = input.Status
	}
	if id == actorID && status == "disabled" {
		return ManagedUser{}, true, errString("cannot disable your own account")
	}
	if id == actorID && role != "admin" {
		return ManagedUser{}, true, errString("cannot remove your own admin role")
	}

	if input.Password != "" {
		if len(input.Password) < 8 {
			return ManagedUser{}, true, errString("password must be at least 8 characters")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return ManagedUser{}, true, err
		}
		_, err = s.db.Exec(`
			UPDATE users SET email = ?, name = ?, role = ?, status = ?, password_hash = ?
			WHERE id = ?`, email, name, role, status, string(hash), id)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return ManagedUser{}, true, errDuplicateEmail
			}
			return ManagedUser{}, true, err
		}
	} else {
		_, err = s.db.Exec(`
			UPDATE users SET email = ?, name = ?, role = ?, status = ?
			WHERE id = ?`, email, name, role, status, id)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return ManagedUser{}, true, errDuplicateEmail
			}
			return ManagedUser{}, true, err
		}
	}
	if status == "disabled" {
		_, _ = s.db.Exec(`DELETE FROM auth_sessions WHERE user_id = ?`, id)
	}
	return s.getManagedUser(id)
}

func (s *AuthStore) deleteUser(id int) (bool, error) {
	var role string
	err := s.db.QueryRow(`SELECT role FROM users WHERE id = ?`, id).Scan(&role)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if role == "admin" {
		var admins int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin' AND status = 'active'`).Scan(&admins); err != nil {
			return false, err
		}
		if admins <= 1 {
			return true, errString("cannot delete the last active admin")
		}
	}
	res, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
