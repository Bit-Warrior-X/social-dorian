package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Proxy struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Country      string `json:"country"`
	Status       string `json:"status"`
	Notes        string `json:"notes"`
	AccountCount int    `json:"accountCount"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type ProxyInput struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Country  string `json:"country"`
	Status   string `json:"status"`
	Notes    string `json:"notes"`
}

var allowedProxyProtocols = map[string]bool{
	"http":   true,
	"https":  true,
	"socks5": true,
}

var allowedProxyStatuses = map[string]bool{
	"active":   true,
	"inactive": true,
	"error":    true,
}

type ProxyStore struct {
	db *sql.DB
}

func newProxyStore(db *sql.DB) *ProxyStore {
	return &ProxyStore{db: db}
}

func (s *ProxyStore) list() ([]Proxy, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.protocol, p.host, p.port, p.username, p.password, p.country,
		       p.status, p.notes, p.created_at, p.updated_at,
		       COUNT(a.id) AS account_count
		FROM proxies p
		LEFT JOIN accounts a ON a.proxy_id = p.id
		GROUP BY p.id, p.name, p.protocol, p.host, p.port, p.username, p.password, p.country,
		         p.status, p.notes, p.created_at, p.updated_at
		ORDER BY p.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Proxy, 0)
	for rows.Next() {
		proxy, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, proxy)
	}
	return out, rows.Err()
}

func (s *ProxyStore) get(id int) (Proxy, bool, error) {
	row := s.db.QueryRow(`
		SELECT p.id, p.name, p.protocol, p.host, p.port, p.username, p.password, p.country,
		       p.status, p.notes, p.created_at, p.updated_at,
		       COUNT(a.id) AS account_count
		FROM proxies p
		LEFT JOIN accounts a ON a.proxy_id = p.id
		WHERE p.id = ?
		GROUP BY p.id, p.name, p.protocol, p.host, p.port, p.username, p.password, p.country,
		         p.status, p.notes, p.created_at, p.updated_at`, id)

	proxy, err := scanProxy(row)
	if err == sql.ErrNoRows {
		return Proxy{}, false, nil
	}
	if err != nil {
		return Proxy{}, false, err
	}
	return proxy, true, nil
}

func (s *ProxyStore) create(input ProxyInput) (Proxy, error) {
	res, err := s.db.Exec(`
		INSERT INTO proxies (name, protocol, host, port, username, password, country, status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.Name,
		input.Protocol,
		input.Host,
		input.Port,
		input.Username,
		input.Password,
		input.Country,
		input.Status,
		input.Notes,
	)
	if err != nil {
		return Proxy{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Proxy{}, err
	}
	proxy, _, err := s.get(int(id))
	return proxy, err
}

func (s *ProxyStore) update(id int, input ProxyInput) (Proxy, bool, error) {
	if _, found, err := s.get(id); err != nil {
		return Proxy{}, false, err
	} else if !found {
		return Proxy{}, false, nil
	}

	_, err := s.db.Exec(`
		UPDATE proxies SET
			name = ?, protocol = ?, host = ?, port = ?, username = ?, password = ?,
			country = ?, status = ?, notes = ?
		WHERE id = ?`,
		input.Name,
		input.Protocol,
		input.Host,
		input.Port,
		input.Username,
		input.Password,
		input.Country,
		input.Status,
		input.Notes,
		id,
	)
	if err != nil {
		return Proxy{}, false, err
	}
	proxy, found, err := s.get(id)
	return proxy, found, err
}

func (s *ProxyStore) delete(id int) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM proxies WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

type proxyScanner interface {
	Scan(dest ...any) error
}

func scanProxy(row proxyScanner) (Proxy, error) {
	var (
		proxy     Proxy
		createdAt time.Time
		updatedAt time.Time
	)
	err := row.Scan(
		&proxy.ID,
		&proxy.Name,
		&proxy.Protocol,
		&proxy.Host,
		&proxy.Port,
		&proxy.Username,
		&proxy.Password,
		&proxy.Country,
		&proxy.Status,
		&proxy.Notes,
		&createdAt,
		&updatedAt,
		&proxy.AccountCount,
	)
	if err != nil {
		return Proxy{}, err
	}
	proxy.CreatedAt = formatDateTime(createdAt)
	proxy.UpdatedAt = formatDateTime(updatedAt)
	return proxy, nil
}

func (s *ProxyStore) handleProxies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		proxies, err := s.list()
		if err != nil {
			log.Printf("list proxies: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to list proxies")
			return
		}
		writeJSON(w, http.StatusOK, proxies)
	case http.MethodPost:
		input, ok := decodeProxyInput(w, r)
		if !ok {
			return
		}
		proxy, err := s.create(input)
		if err != nil {
			log.Printf("create proxy: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to create proxy")
			return
		}
		writeJSON(w, http.StatusCreated, proxy)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *ProxyStore) handleProxyByID(w http.ResponseWriter, r *http.Request) {
	id, action, ok := parseProxyPath(w, r.URL.Path)
	if !ok {
		return
	}

	if action == "check" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		result, found, err := s.check(id)
		if err != nil {
			log.Printf("check proxy: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to check proxy")
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "proxy not found")
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}

	switch r.Method {
	case http.MethodGet:
		proxy, found, err := s.get(id)
		if err != nil {
			log.Printf("get proxy: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to get proxy")
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "proxy not found")
			return
		}
		writeJSON(w, http.StatusOK, proxy)
	case http.MethodPut:
		input, valid := decodeProxyInput(w, r)
		if !valid {
			return
		}
		proxy, found, err := s.update(id, input)
		if err != nil {
			log.Printf("update proxy: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to update proxy")
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "proxy not found")
			return
		}
		writeJSON(w, http.StatusOK, proxy)
	case http.MethodDelete:
		deleted, err := s.delete(id)
		if err != nil {
			log.Printf("delete proxy: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to delete proxy")
			return
		}
		if !deleted {
			writeError(w, http.StatusNotFound, "proxy not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func decodeProxyInput(w http.ResponseWriter, r *http.Request) (ProxyInput, bool) {
	var input ProxyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return ProxyInput{}, false
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Protocol = strings.ToLower(strings.TrimSpace(input.Protocol))
	input.Host = strings.TrimSpace(input.Host)
	input.Username = strings.TrimSpace(input.Username)
	input.Password = strings.TrimSpace(input.Password)
	input.Country = strings.TrimSpace(input.Country)
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	input.Notes = strings.TrimSpace(input.Notes)

	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return ProxyInput{}, false
	}
	if input.Protocol == "" {
		input.Protocol = "http"
	}
	if !allowedProxyProtocols[input.Protocol] {
		writeError(w, http.StatusBadRequest, "protocol must be http, https, or socks5")
		return ProxyInput{}, false
	}
	if input.Host == "" {
		writeError(w, http.StatusBadRequest, "host is required")
		return ProxyInput{}, false
	}
	if input.Port < 1 || input.Port > 65535 {
		writeError(w, http.StatusBadRequest, "port must be between 1 and 65535")
		return ProxyInput{}, false
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if !allowedProxyStatuses[input.Status] {
		writeError(w, http.StatusBadRequest, "status must be active, inactive, or error")
		return ProxyInput{}, false
	}

	return input, true
}

func parseProxyPath(w http.ResponseWriter, path string) (int, string, bool) {
	rest := strings.TrimPrefix(path, "/api/proxies/")
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
		if action != "check" {
			writeError(w, http.StatusNotFound, "not found")
			return 0, "", false
		}
	} else if len(parts) > 2 {
		writeError(w, http.StatusNotFound, "not found")
		return 0, "", false
	}

	return id, action, true
}

func parsePathID(w http.ResponseWriter, path, prefix string) (int, bool) {
	idStr := strings.TrimPrefix(path, prefix)
	idStr = strings.Trim(idStr, "/")
	if idStr == "" || strings.Contains(idStr, "/") {
		writeError(w, http.StatusNotFound, "not found")
		return 0, false
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}
