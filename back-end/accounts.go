package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type Account struct {
	ID            int    `json:"id"`
	Platform      string `json:"platform"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	Password      string `json:"password"`
	Email         string `json:"email"`
	EmailPassword string `json:"emailPassword"`
	Birthday      string `json:"birthday"`
	Gender        string `json:"gender"`
	ProxyMode     string `json:"proxyMode"`
	ProxyID       int    `json:"proxyId"`
	Status        string `json:"status"`
	ConnectedBy   string `json:"connectedBy"`
	ConnectedAt   string `json:"connectedAt"`
}

type AccountInput struct {
	Platform      string `json:"platform"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	Password      string `json:"password"`
	Email         string `json:"email"`
	EmailPassword string `json:"emailPassword"`
	Birthday      string `json:"birthday"`
	Gender        string `json:"gender"`
	ProxyMode     string `json:"proxyMode"`
	ProxyID       int    `json:"proxyId"`
	Status        string `json:"status"`
	ConnectedBy   string `json:"connectedBy"`
	ConnectedAt   string `json:"connectedAt"`
}

var allowedPlatforms = map[string]bool{
	"twitter":   true,
	"instagram": true,
	"linkedin":  true,
	"facebook":  true,
	"tiktok":    true,
	"youtube":   true,
}

var allowedGenders = map[string]bool{
	"male":              true,
	"female":            true,
	"other":             true,
	"prefer_not_to_say": true,
}

var allowedStatuses = map[string]bool{
	"active":  true,
	"expired": true,
	"error":   true,
	"revoked": true,
}

var allowedProxyModes = map[string]bool{
	"none":   true,
	"auto":   true,
	"manual": true,
}

type AccountStore struct {
	db *sql.DB
}

func newAccountStore(db *sql.DB) *AccountStore {
	return &AccountStore{db: db}
}

func (s *AccountStore) list() ([]Account, error) {
	rows, err := s.db.Query(`
		SELECT id, platform, first_name, last_name, password, email, email_password,
		       birthday, gender, proxy_mode, proxy_id, status, connected_by, connected_at
		FROM accounts
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Account, 0)
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, account)
	}
	return out, rows.Err()
}

func (s *AccountStore) get(id int) (Account, bool, error) {
	row := s.db.QueryRow(`
		SELECT id, platform, first_name, last_name, password, email, email_password,
		       birthday, gender, proxy_mode, proxy_id, status, connected_by, connected_at
		FROM accounts
		WHERE id = ?`, id)

	account, err := scanAccount(row)
	if err == sql.ErrNoRows {
		return Account{}, false, nil
	}
	if err != nil {
		return Account{}, false, err
	}
	return account, true, nil
}

func (s *AccountStore) create(input AccountInput) (Account, error) {
	connectedAt := input.ConnectedAt
	if connectedAt == "" {
		connectedAt = time.Now().UTC().Format("2006-01-02")
	}

	res, err := s.db.Exec(`
		INSERT INTO accounts (
			platform, first_name, last_name, password, email, email_password,
			birthday, gender, proxy_mode, proxy_id, status, connected_by, connected_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.Platform,
		input.FirstName,
		input.LastName,
		input.Password,
		input.Email,
		input.EmailPassword,
		nullString(input.Birthday),
		input.Gender,
		input.ProxyMode,
		nullProxyID(input.ProxyMode, input.ProxyID),
		input.Status,
		input.ConnectedBy,
		connectedAt,
	)
	if err != nil {
		return Account{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Account{}, err
	}
	account, _, err := s.get(int(id))
	return account, err
}

func (s *AccountStore) update(id int, input AccountInput) (Account, bool, error) {
	if _, found, err := s.get(id); err != nil {
		return Account{}, false, err
	} else if !found {
		return Account{}, false, nil
	}

	_, err := s.db.Exec(`
		UPDATE accounts SET
			platform = ?, first_name = ?, last_name = ?, password = ?, email = ?, email_password = ?,
			birthday = ?, gender = ?, proxy_mode = ?, proxy_id = ?,
			status = COALESCE(NULLIF(?, ''), status)
		WHERE id = ?`,
		input.Platform,
		input.FirstName,
		input.LastName,
		input.Password,
		input.Email,
		input.EmailPassword,
		nullString(input.Birthday),
		input.Gender,
		input.ProxyMode,
		nullProxyID(input.ProxyMode, input.ProxyID),
		input.Status,
		id,
	)
	if err != nil {
		return Account{}, false, err
	}

	account, found, err := s.get(id)
	return account, found, err
}

func (s *AccountStore) delete(id int) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

type accountScanner interface {
	Scan(dest ...any) error
}

func scanAccount(row accountScanner) (Account, error) {
	var (
		account   Account
		birthday  sql.NullTime
		proxyID   sql.NullInt64
		connected sql.NullTime
	)
	err := row.Scan(
		&account.ID,
		&account.Platform,
		&account.FirstName,
		&account.LastName,
		&account.Password,
		&account.Email,
		&account.EmailPassword,
		&birthday,
		&account.Gender,
		&account.ProxyMode,
		&proxyID,
		&account.Status,
		&account.ConnectedBy,
		&connected,
	)
	if err != nil {
		return Account{}, err
	}
	account.Birthday = formatDate(birthday)
	account.ProxyID = scanProxyID(proxyID)
	account.ConnectedAt = formatDate(connected)
	return account, nil
}

func (s *AccountStore) handleAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		accounts, err := s.list()
		if err != nil {
			log.Printf("list accounts: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to list accounts")
			return
		}
		writeJSON(w, http.StatusOK, accounts)
	case http.MethodPost:
		input, ok := decodeAndValidateAccount(w, r, true)
		if !ok {
			return
		}
		account, err := s.create(input)
		if err != nil {
			log.Printf("create account: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to create account")
			return
		}
		writeJSON(w, http.StatusCreated, account)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *AccountStore) handleAccountByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r.URL.Path, "/api/accounts/")
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		account, found, err := s.get(id)
		if err != nil {
			log.Printf("get account: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to get account")
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}
		writeJSON(w, http.StatusOK, account)
	case http.MethodPut:
		input, valid := decodeAndValidateAccount(w, r, false)
		if !valid {
			return
		}
		account, found, err := s.update(id, input)
		if err != nil {
			log.Printf("update account: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to update account")
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}
		writeJSON(w, http.StatusOK, account)
	case http.MethodDelete:
		deleted, err := s.delete(id)
		if err != nil {
			log.Printf("delete account: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to delete account")
			return
		}
		if !deleted {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func decodeAndValidateAccount(w http.ResponseWriter, r *http.Request, creating bool) (AccountInput, bool) {
	var input AccountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return AccountInput{}, false
	}

	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Password = strings.TrimSpace(input.Password)
	input.Email = strings.TrimSpace(input.Email)
	input.EmailPassword = strings.TrimSpace(input.EmailPassword)
	input.Birthday = strings.TrimSpace(input.Birthday)
	input.Gender = strings.ToLower(strings.TrimSpace(input.Gender))
	input.ProxyMode = strings.ToLower(strings.TrimSpace(input.ProxyMode))
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	input.ConnectedBy = strings.TrimSpace(input.ConnectedBy)

	if input.Platform == "" {
		writeError(w, http.StatusBadRequest, "platform is required")
		return AccountInput{}, false
	}
	if !allowedPlatforms[input.Platform] {
		writeError(w, http.StatusBadRequest, "unsupported platform")
		return AccountInput{}, false
	}
	if input.FirstName == "" || input.LastName == "" {
		writeError(w, http.StatusBadRequest, "first name and last name are required")
		return AccountInput{}, false
	}
	if input.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return AccountInput{}, false
	}
	if input.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return AccountInput{}, false
	}
	if input.Gender == "" {
		input.Gender = "prefer_not_to_say"
	}
	if !allowedGenders[input.Gender] {
		writeError(w, http.StatusBadRequest, "invalid gender")
		return AccountInput{}, false
	}
	if input.ProxyMode == "" {
		input.ProxyMode = "none"
	}
	if !allowedProxyModes[input.ProxyMode] {
		writeError(w, http.StatusBadRequest, "proxyMode must be none, auto, or manual")
		return AccountInput{}, false
	}
	if input.ProxyMode == "manual" || input.ProxyMode == "auto" {
		if input.ProxyID < 1 {
			if input.ProxyMode == "manual" {
				writeError(w, http.StatusBadRequest, "select a registered proxy for manual mode")
			} else {
				writeError(w, http.StatusBadRequest, "auto select requires an available proxy")
			}
			return AccountInput{}, false
		}
	} else {
		input.ProxyID = 0
	}
	if creating {
		if input.Status == "" {
			input.Status = "active"
		}
		if input.ConnectedBy == "" {
			input.ConnectedBy = "You"
		}
	}
	if input.Status != "" && !allowedStatuses[input.Status] {
		writeError(w, http.StatusBadRequest, "status must be active, expired, error, or revoked")
		return AccountInput{}, false
	}

	return input, true
}
