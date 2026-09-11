package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ActivityStore is the platform-wide operator log (tasks, accounts, proxies, system).
type ActivityStore struct {
	db *sql.DB
}

type ActivityEntry struct {
	ID          int64  `json:"id"`
	Source      string `json:"source"`
	Level       string `json:"level"`
	Message     string `json:"message"`
	TaskID      int    `json:"taskId,omitempty"`
	TaskType    string `json:"taskType,omitempty"`
	TaskTitle   string `json:"taskTitle,omitempty"`
	AccountID   int    `json:"accountId,omitempty"`
	AccountName string `json:"accountName,omitempty"`
	ProxyID     int    `json:"proxyId,omitempty"`
	ProxyName   string `json:"proxyName,omitempty"`
	Actor       string `json:"actor,omitempty"`
	Context     string `json:"context,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

type ActivityInput struct {
	Source    string
	Level     string
	Message   string
	TaskID    int
	AccountID int
	ProxyID   int
	Actor     string
	Context   string
}

var activityBus *ActivityStore

func newActivityStore(db *sql.DB) *ActivityStore {
	return &ActivityStore{db: db}
}

func (s *ActivityStore) ensureSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS activity_logs (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			source ENUM('task','account','proxy','system') NOT NULL DEFAULT 'system',
			level ENUM('info','success','warn','error') NOT NULL DEFAULT 'info',
			message TEXT NOT NULL,
			task_id INT UNSIGNED NULL,
			account_id INT UNSIGNED NULL,
			proxy_id INT UNSIGNED NULL,
			actor VARCHAR(120) NOT NULL DEFAULT '',
			context_json TEXT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY idx_activity_created (created_at),
			KEY idx_activity_level (level),
			KEY idx_activity_source (source),
			KEY idx_activity_task (task_id),
			KEY idx_activity_account (account_id),
			KEY idx_activity_proxy (proxy_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	return err
}

func setActivityBus(store *ActivityStore) {
	activityBus = store
}

func emitActivity(in ActivityInput) {
	if activityBus == nil {
		return
	}
	activityBus.Add(in)
}

func (s *ActivityStore) Add(in ActivityInput) {
	if s == nil || s.db == nil {
		return
	}
	msg := strings.TrimSpace(in.Message)
	if msg == "" {
		return
	}
	if len(msg) > 4000 {
		msg = msg[:4000] + "…"
	}
	level := normalizeActivityLevel(in.Level)
	source := normalizeActivitySource(in.Source)
	actor := strings.TrimSpace(in.Actor)
	ctx := strings.TrimSpace(in.Context)
	if len(ctx) > 4000 {
		ctx = ctx[:4000] + "…"
	}

	var (
		taskID    any
		accountID any
		proxyID   any
		context   any
	)
	if in.TaskID > 0 {
		taskID = in.TaskID
	}
	if in.AccountID > 0 {
		accountID = in.AccountID
	}
	if in.ProxyID > 0 {
		proxyID = in.ProxyID
	}
	if ctx != "" {
		context = ctx
	}

	_, err := s.db.Exec(`
		INSERT INTO activity_logs (source, level, message, task_id, account_id, proxy_id, actor, context_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		source, level, msg, taskID, accountID, proxyID, actor, context,
	)
	if err != nil {
		log.Printf("activity_logs insert failed: %v", err)
	}
}

func normalizeActivityLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "success":
		return "success"
	case "warn", "warning":
		return "warn"
	case "error", "fatal", "critical":
		return "error"
	default:
		return "info"
	}
}

func normalizeActivitySource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "task":
		return "task"
	case "account":
		return "account"
	case "proxy":
		return "proxy"
	default:
		return "system"
	}
}

func (s *ActivityStore) handleFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	afterID, _ := strconv.ParseInt(strings.TrimSpace(q.Get("after")), 10, 64)
	limit, _ := strconv.Atoi(strings.TrimSpace(q.Get("limit")))
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	level := normalizeActivityLevel(q.Get("level"))
	if strings.TrimSpace(q.Get("level")) == "" {
		level = ""
	}
	source := normalizeActivitySource(q.Get("source"))
	if strings.TrimSpace(q.Get("source")) == "" {
		source = ""
	}
	taskID, _ := strconv.Atoi(strings.TrimSpace(q.Get("taskId")))
	accountID, _ := strconv.Atoi(strings.TrimSpace(q.Get("accountId")))
	proxyID, _ := strconv.Atoi(strings.TrimSpace(q.Get("proxyId")))
	search := strings.TrimSpace(q.Get("q"))
	errorsOnly := strings.EqualFold(strings.TrimSpace(q.Get("errors")), "1") ||
		strings.EqualFold(strings.TrimSpace(q.Get("errors")), "true")

	query := `
		SELECT a.id, a.source, a.level, a.message,
		       COALESCE(a.task_id, 0), COALESCE(t.type, ''), COALESCE(t.title, ''),
		       COALESCE(a.account_id, 0),
		       TRIM(CONCAT(COALESCE(acc.first_name, ''), ' ', COALESCE(acc.last_name, ''))),
		       COALESCE(acc.email, ''),
		       COALESCE(a.proxy_id, 0), COALESCE(p.name, ''),
		       a.actor, COALESCE(a.context_json, ''), a.created_at
		FROM activity_logs a
		LEFT JOIN tasks t ON t.id = a.task_id
		LEFT JOIN accounts acc ON acc.id = a.account_id
		LEFT JOIN proxies p ON p.id = a.proxy_id
		WHERE 1=1`
	args := []any{}

	if afterID > 0 {
		query += ` AND a.id > ?`
		args = append(args, afterID)
	}
	if errorsOnly {
		query += ` AND a.level IN ('error','warn')`
	} else if level != "" {
		query += ` AND a.level = ?`
		args = append(args, level)
	}
	if source != "" {
		query += ` AND a.source = ?`
		args = append(args, source)
	}
	if taskID > 0 {
		query += ` AND a.task_id = ?`
		args = append(args, taskID)
	}
	if accountID > 0 {
		query += ` AND a.account_id = ?`
		args = append(args, accountID)
	}
	if proxyID > 0 {
		query += ` AND a.proxy_id = ?`
		args = append(args, proxyID)
	}
	if search != "" {
		query += ` AND (a.message LIKE ? OR a.context_json LIKE ? OR t.title LIKE ? OR acc.email LIKE ? OR p.name LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like, like, like)
	}

	query += ` ORDER BY a.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load activity feed")
		return
	}
	defer rows.Close()

	out := make([]ActivityEntry, 0)
	for rows.Next() {
		var (
			entry                 ActivityEntry
			accountName, email    string
			createdAt             time.Time
		)
		if err := rows.Scan(
			&entry.ID, &entry.Source, &entry.Level, &entry.Message,
			&entry.TaskID, &entry.TaskType, &entry.TaskTitle,
			&entry.AccountID, &accountName, &email,
			&entry.ProxyID, &entry.ProxyName,
			&entry.Actor, &entry.Context, &createdAt,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load activity feed")
			return
		}
		name := strings.TrimSpace(accountName)
		if name == "" {
			name = strings.TrimSpace(email)
		}
		entry.AccountName = name
		entry.CreatedAt = formatDateTime(createdAt)
		out = append(out, entry)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"logs": out,
		"meta": map[string]any{
			"count": len(out),
			"limit": limit,
		},
	})
}

func (s *ActivityStore) backfillFromTaskLogs() {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM activity_logs`).Scan(&count); err != nil || count > 0 {
		return
	}
	res, err := s.db.Exec(`
		INSERT INTO activity_logs (source, level, message, task_id, account_id, created_at)
		SELECT 'task', level, message, task_id, account_id, created_at
		FROM task_logs
		ORDER BY id ASC
		LIMIT 2000`)
	if err != nil {
		log.Printf("activity backfill skipped: %v", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("activity_logs: backfilled %d rows from task_logs", n)
		emitActivity(ActivityInput{
			Source:  "system",
			Level:   "info",
			Message: fmt.Sprintf("Backfilled %d historical task log lines into activity feed", n),
		})
	}
}
