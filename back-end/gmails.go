package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

type Gmail struct {
	ID            int    `json:"id"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	AppPassword   string `json:"appPassword"`
	RecoveryEmail string `json:"recoveryEmail"`
	RecoveryPhone string `json:"recoveryPhone"`
	TwofaSecret   string `json:"twofaSecret"`
	BackupCodes   string `json:"backupCodes"`
	PinCode       string `json:"pinCode"`
	Label         string `json:"label"`
	Status        string `json:"status"`
	Notes         string `json:"notes"`
	AccountCount  int    `json:"accountCount"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

type GmailInput struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	AppPassword   string `json:"appPassword"`
	RecoveryEmail string `json:"recoveryEmail"`
	RecoveryPhone string `json:"recoveryPhone"`
	TwofaSecret   string `json:"twofaSecret"`
	BackupCodes   string `json:"backupCodes"`
	PinCode       string `json:"pinCode"`
	Label         string `json:"label"`
	Status        string `json:"status"`
	Notes         string `json:"notes"`
}

var allowedGmailStatuses = map[string]bool{
	"active":   true,
	"inactive": true,
	"error":    true,
}

type GmailStore struct {
	db *sql.DB
}

type PinCodeResponse struct {
	PinCode string `json:"pinCode"`
}

func newGmailStore(db *sql.DB) *GmailStore {
	return &GmailStore{db: db}
}

func (s *GmailStore) list() ([]Gmail, error) {
	rows, err := s.db.Query(`
		SELECT g.id, g.email, g.password, g.app_password, g.recovery_email, g.recovery_phone,
		       g.twofa_secret, g.backup_codes, g.pin_code, g.label,
		       g.status, g.notes, g.created_at, g.updated_at,
		       COUNT(a.id) AS account_count
		FROM gmails g
		LEFT JOIN accounts a ON LOWER(a.email) = LOWER(g.email)
		GROUP BY g.id, g.email, g.password, g.app_password, g.recovery_email, g.recovery_phone,
		         g.twofa_secret, g.backup_codes, g.pin_code, g.label,
		         g.status, g.notes, g.created_at, g.updated_at
		ORDER BY g.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Gmail, 0)
	for rows.Next() {
		item, err := scanGmail(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *GmailStore) get(id int) (Gmail, bool, error) {
	row := s.db.QueryRow(`
		SELECT g.id, g.email, g.password, g.app_password, g.recovery_email, g.recovery_phone,
		       g.twofa_secret, g.backup_codes, g.pin_code, g.label,
		       g.status, g.notes, g.created_at, g.updated_at,
		       COUNT(a.id) AS account_count
		FROM gmails g
		LEFT JOIN accounts a ON LOWER(a.email) = LOWER(g.email)
		WHERE g.id = ?
		GROUP BY g.id, g.email, g.password, g.app_password, g.recovery_email, g.recovery_phone,
		         g.twofa_secret, g.backup_codes, g.pin_code, g.label,
		         g.status, g.notes, g.created_at, g.updated_at`, id)

	item, err := scanGmail(row)
	if err == sql.ErrNoRows {
		return Gmail{}, false, nil
	}
	if err != nil {
		return Gmail{}, false, err
	}
	return item, true, nil
}

func (s *GmailStore) create(input GmailInput) (Gmail, error) {
	res, err := s.db.Exec(`
		INSERT INTO gmails (email, password, app_password, recovery_email, recovery_phone,
		  twofa_secret, backup_codes, pin_code, label, status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.Email,
		input.Password,
		input.AppPassword,
		input.RecoveryEmail,
		input.RecoveryPhone,
		input.TwofaSecret,
		input.BackupCodes,
		input.PinCode,
		input.Label,
		input.Status,
		input.Notes,
	)
	if err != nil {
		return Gmail{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Gmail{}, err
	}
	item, _, err := s.get(int(id))
	return item, err
}

func (s *GmailStore) update(id int, input GmailInput) (Gmail, bool, error) {
	if _, found, err := s.get(id); err != nil {
		return Gmail{}, false, err
	} else if !found {
		return Gmail{}, false, nil
	}

	_, err := s.db.Exec(`
		UPDATE gmails SET
			email = ?, password = ?, app_password = ?, recovery_email = ?, recovery_phone = ?,
			twofa_secret = ?, backup_codes = ?, pin_code = ?,
			label = ?, status = ?, notes = ?
		WHERE id = ?`,
		input.Email,
		input.Password,
		input.AppPassword,
		input.RecoveryEmail,
		input.RecoveryPhone,
		input.TwofaSecret,
		input.BackupCodes,
		input.PinCode,
		input.Label,
		input.Status,
		input.Notes,
		id,
	)
	if err != nil {
		return Gmail{}, false, err
	}
	item, found, err := s.get(id)
	return item, found, err
}

func (s *GmailStore) delete(id int) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM gmails WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

type gmailScanner interface {
	Scan(dest ...any) error
}

func scanGmail(row gmailScanner) (Gmail, error) {
	var (
		item      Gmail
		createdAt time.Time
		updatedAt time.Time
	)
	err := row.Scan(
		&item.ID,
		&item.Email,
		&item.Password,
		&item.AppPassword,
		&item.RecoveryEmail,
		&item.RecoveryPhone,
		&item.TwofaSecret,
		&item.BackupCodes,
		&item.PinCode,
		&item.Label,
		&item.Status,
		&item.Notes,
		&createdAt,
		&updatedAt,
		&item.AccountCount,
	)
	if err != nil {
		return Gmail{}, err
	}
	item.CreatedAt = formatDateTime(createdAt)
	item.UpdatedAt = formatDateTime(updatedAt)
	return item, nil
}

func (s *GmailStore) handleGmails(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.list()
		if err != nil {
			log.Printf("list gmails: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to list gmails")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		input, ok := decodeGmailInput(w, r)
		if !ok {
			return
		}
		item, err := s.create(input)
		if err != nil {
			if isDuplicateKey(err) {
				writeError(w, http.StatusConflict, "a Gmail with this address already exists")
				return
			}
			log.Printf("create gmail: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to create gmail")
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *GmailStore) handleGmailByID(w http.ResponseWriter, r *http.Request) {
	id, action, ok := parseGmailPath(w, r.URL.Path, "/api/gmails/")
	if !ok {
		return
	}

	if action == "pin-code" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		pin, err := generatePinCode(6)
		if err != nil {
			log.Printf("generate pin code: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to generate pin code")
			return
		}

		// Persist the latest pin for auditability / later retrieval.
		if _, err := s.db.Exec(`UPDATE gmails SET pin_code = ? WHERE id = ?`, pin, id); err != nil {
			log.Printf("update pin code: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to update pin code")
			return
		}

		writeJSON(w, http.StatusOK, PinCodeResponse{PinCode: pin})
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, found, err := s.get(id)
		if err != nil {
			log.Printf("get gmail: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to get gmail")
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "gmail not found")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		input, valid := decodeGmailInput(w, r)
		if !valid {
			return
		}
		item, found, err := s.update(id, input)
		if err != nil {
			if isDuplicateKey(err) {
				writeError(w, http.StatusConflict, "a Gmail with this address already exists")
				return
			}
			log.Printf("update gmail: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to update gmail")
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "gmail not found")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		deleted, err := s.delete(id)
		if err != nil {
			log.Printf("delete gmail: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to delete gmail")
			return
		}
		if !deleted {
			writeError(w, http.StatusNotFound, "gmail not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func decodeGmailInput(w http.ResponseWriter, r *http.Request) (GmailInput, bool) {
	var input GmailInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return GmailInput{}, false
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Password = strings.TrimSpace(input.Password)
	input.AppPassword = strings.TrimSpace(input.AppPassword)
	input.RecoveryEmail = strings.ToLower(strings.TrimSpace(input.RecoveryEmail))
	input.RecoveryPhone = strings.TrimSpace(input.RecoveryPhone)
	input.TwofaSecret = strings.TrimSpace(input.TwofaSecret)
	input.BackupCodes = strings.TrimSpace(input.BackupCodes)
	input.PinCode = strings.TrimSpace(input.PinCode)
	input.Label = strings.TrimSpace(input.Label)
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	input.Notes = strings.TrimSpace(input.Notes)

	if input.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return GmailInput{}, false
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid email address")
		return GmailInput{}, false
	}
	if input.AppPassword == "" {
		writeError(w, http.StatusBadRequest, "app password is required")
		return GmailInput{}, false
	}
	if input.RecoveryEmail != "" {
		if _, err := mail.ParseAddress(input.RecoveryEmail); err != nil {
			writeError(w, http.StatusBadRequest, "invalid recovery email")
			return GmailInput{}, false
		}
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if !allowedGmailStatuses[input.Status] {
		writeError(w, http.StatusBadRequest, "status must be active, inactive, or error")
		return GmailInput{}, false
	}
	if input.Label == "" {
		input.Label = input.Email
	}

	if input.PinCode != "" {
		if len(input.PinCode) < 4 || len(input.PinCode) > 12 {
			writeError(w, http.StatusBadRequest, "pinCode must be 4-12 digits")
			return GmailInput{}, false
		}
		for _, r := range input.PinCode {
			if r < '0' || r > '9' {
				writeError(w, http.StatusBadRequest, "pinCode must contain digits only")
				return GmailInput{}, false
			}
		}
	}

	return input, true
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}

func parseGmailPath(w http.ResponseWriter, path, prefix string) (int, string, bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		writeError(w, http.StatusNotFound, "not found")
		return 0, "", false
	}

	parts := strings.Split(rest, "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, "", false
	}

	action := ""
	if len(parts) == 2 {
		action = parts[1]
		if action != "pin-code" {
			writeError(w, http.StatusNotFound, "not found")
			return 0, "", false
		}
	} else if len(parts) > 2 {
		writeError(w, http.StatusNotFound, "not found")
		return 0, "", false
	}

	return id, action, true
}

func generatePinCode(length int) (string, error) {
	if length < 4 {
		return "", nil
	}
	if length > 12 {
		length = 12
	}

	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		buf[i] = byte('0' + n.Int64())
	}
	return string(buf), nil
}
