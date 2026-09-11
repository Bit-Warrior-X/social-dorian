package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TaskContent struct {
	Text     string `json:"text,omitempty"`
	Headline string `json:"headline,omitempty"`
	LinkURL  string `json:"linkUrl,omitempty"`
	MediaURL string `json:"mediaUrl,omitempty"`
}

type Task struct {
	ID              int         `json:"id"`
	Type            string      `json:"type"`
	Title           string      `json:"title"`
	TargetURL       string      `json:"targetUrl"`
	Content         TaskContent `json:"content"`
	Status          string      `json:"status"`
	DelayMinSec     int         `json:"delayMinSec"`
	DelayMaxSec     int         `json:"delayMaxSec"`
	UseAccountProxy bool        `json:"useAccountProxy"`
	ShowBrowser     bool        `json:"showBrowser"`
	AccountIDs      []int       `json:"accountIds"`
	AccountCount    int         `json:"accountCount"`
	DoneCount       int         `json:"doneCount"`
	SuccessCount    int         `json:"successCount"`
	FailCount       int         `json:"failCount"`
	CreatedBy       string      `json:"createdBy"`
	CreatedAt       string      `json:"createdAt"`
	StartedAt       string      `json:"startedAt,omitempty"`
	FinishedAt      string      `json:"finishedAt,omitempty"`
	Items           []TaskItem  `json:"items,omitempty"`
	Logs            []TaskLog   `json:"logs,omitempty"`
}

type TaskItem struct {
	ID           int    `json:"id"`
	TaskID       int    `json:"taskId"`
	AccountID    int    `json:"accountId"`
	AccountName  string `json:"accountName"`
	AccountEmail string `json:"accountEmail"`
	Platform     string `json:"platform"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	StartedAt    string `json:"startedAt,omitempty"`
	FinishedAt   string `json:"finishedAt,omitempty"`
}

type TaskLog struct {
	ID        int    `json:"id"`
	TaskID    int    `json:"taskId"`
	AccountID int    `json:"accountId,omitempty"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

type TaskInput struct {
	Type            string      `json:"type"`
	Title           string      `json:"title"`
	TargetURL       string      `json:"targetUrl"`
	Content         TaskContent `json:"content"`
	AccountIDs      []int       `json:"accountIds"`
	DelayMinSec     int         `json:"delayMinSec"`
	DelayMaxSec     int         `json:"delayMaxSec"`
	UseAccountProxy bool        `json:"useAccountProxy"`
	ShowBrowser     bool        `json:"showBrowser"`
}

type TaskStore struct {
	db      *sql.DB
	mu      sync.Mutex
	running map[int]bool
}

func newTaskStore(db *sql.DB) *TaskStore {
	return &TaskStore{
		db:      db,
		running: map[int]bool{},
	}
}

func (s *TaskStore) ensureSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tasks (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			type ENUM('report','post','browse','login_test','reply') NOT NULL,
			title VARCHAR(255) NOT NULL DEFAULT '',
			target_url TEXT NOT NULL,
			content TEXT NOT NULL,
			status ENUM('pending','queued','running','completed','failed','cancelled') NOT NULL DEFAULT 'pending',
			delay_min_sec INT UNSIGNED NOT NULL DEFAULT 5,
			delay_max_sec INT UNSIGNED NOT NULL DEFAULT 15,
			use_account_proxy TINYINT(1) NOT NULL DEFAULT 1,
			show_browser TINYINT(1) NOT NULL DEFAULT 0,
			created_by VARCHAR(120) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME NULL,
			finished_at DATETIME NULL,
			PRIMARY KEY (id),
			KEY idx_tasks_status (status),
			KEY idx_tasks_type (type),
			KEY idx_tasks_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS task_items (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			task_id INT UNSIGNED NOT NULL,
			account_id INT UNSIGNED NOT NULL,
			status ENUM('pending','running','success','failed','cancelled') NOT NULL DEFAULT 'pending',
			message TEXT NOT NULL,
			started_at DATETIME NULL,
			finished_at DATETIME NULL,
			PRIMARY KEY (id),
			KEY idx_task_items_task (task_id),
			KEY idx_task_items_account (account_id),
			KEY idx_task_items_status (status),
			CONSTRAINT fk_task_items_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
			CONSTRAINT fk_task_items_account FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS task_logs (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			task_id INT UNSIGNED NOT NULL,
			account_id INT UNSIGNED NULL,
			level ENUM('info','success','warn','error') NOT NULL DEFAULT 'info',
			message TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY idx_task_logs_task (task_id),
			CONSTRAINT fk_task_logs_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS workspace_credits (
			id INT UNSIGNED NOT NULL PRIMARY KEY,
			balance INT NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := s.ensureTasksContentColumn(); err != nil {
		return err
	}
	if err := s.ensureTasksReplyType(); err != nil {
		return err
	}
	if err := s.ensureTasksShowBrowserColumn(); err != nil {
		return err
	}
	if _, err := s.db.Exec(`INSERT IGNORE INTO workspace_credits (id, balance) VALUES (1, 1000)`); err != nil {
		return err
	}
	s.recoverStuckTasks()
	return nil
}

// recoverStuckTasks finalizes tasks left "running" after an API restart
// (in-memory workers are gone, so those rows would otherwise hang forever).
func (s *TaskStore) recoverStuckTasks() {
	res, err := s.db.Exec(`
		UPDATE task_items
		SET status = 'failed',
		    message = 'Worker interrupted (API restart)',
		    finished_at = UTC_TIMESTAMP()
		WHERE status IN ('pending', 'running')
		  AND task_id IN (
		    SELECT id FROM (
		      SELECT id FROM tasks WHERE status IN ('queued', 'running')
		    ) t
		  )`)
	if err != nil {
		log.Printf("recoverStuckTasks items: %v", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("recoverStuckTasks: marked %d orphaned item(s) failed", n)
	}

	res, err = s.db.Exec(`
		UPDATE tasks t
		SET status = CASE
			WHEN (
				SELECT COUNT(*) FROM task_items i
				WHERE i.task_id = t.id AND i.status = 'failed'
			) > 0
			AND (
				SELECT COUNT(*) FROM task_items i
				WHERE i.task_id = t.id AND i.status = 'success'
			) = 0
			THEN 'failed'
			ELSE 'completed'
		END,
		finished_at = UTC_TIMESTAMP()
		WHERE t.status IN ('queued', 'running')
		  AND NOT EXISTS (
			SELECT 1 FROM task_items i
			WHERE i.task_id = t.id AND i.status IN ('pending', 'running')
		  )`)
	if err != nil {
		log.Printf("recoverStuckTasks tasks: %v", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("recoverStuckTasks: finalized %d stuck task(s)", n)
	}
}

func (s *TaskStore) ensureTasksShowBrowserColumn() error {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'tasks'
		  AND COLUMN_NAME = 'show_browser'`).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE tasks ADD COLUMN show_browser TINYINT(1) NOT NULL DEFAULT 0`)
	return err
}

func (s *TaskStore) ensureTasksReplyType() error {
	_, err := s.db.Exec(`
		ALTER TABLE tasks
		MODIFY COLUMN type ENUM('report','post','browse','login_test','reply') NOT NULL`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		// Ignore if already applied or table missing during first boot before create
		if strings.Contains(strings.ToLower(err.Error()), "doesn't exist") {
			return nil
		}
		return err
	}
	return nil
}

func (s *TaskStore) ensureTasksContentColumn() error {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'tasks'
		  AND COLUMN_NAME = 'content'`).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE tasks ADD COLUMN content TEXT NOT NULL DEFAULT ''`)
	return err
}

func (s *TaskStore) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		tasks, err := s.list(status)
		if err != nil {
			log.Printf("list tasks: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to list tasks")
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	case http.MethodPost:
		input, ok := decodeTaskInput(w, r)
		if !ok {
			return
		}
		createdBy := "Admin"
		if user := currentUser(r); user != nil {
			createdBy = user.Name
			if createdBy == "" {
				createdBy = user.Email
			}
		}
		task, err := s.create(input, createdBy)
		if err != nil {
			log.Printf("create task: %v", err)
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *TaskStore) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	id, action, ok := parseTaskPath(w, r.URL.Path)
	if !ok {
		return
	}

	switch action {
	case "":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		task, found, err := s.get(id, true)
		if err != nil {
			log.Printf("get task: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to get task")
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeJSON(w, http.StatusOK, task)
	case "cancel":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		task, err := s.cancel(id)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, task)
	case "logs":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		logs, err := s.listLogs(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list logs")
			return
		}
		writeJSON(w, http.StatusOK, logs)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *TaskStore) handleCredits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var balance int
	var updatedAt time.Time
	err := s.db.QueryRow(`SELECT balance, updated_at FROM workspace_credits WHERE id = 1`).Scan(&balance, &updatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load credits")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"balance":          balance,
		"updatedAt":        formatDateTime(updatedAt),
		"costPerAccount":   creditCostPerAccount,
		"debitOnLaunch":    true,
	})
}

type MonitorLog struct {
	ID          int    `json:"id"`
	TaskID      int    `json:"taskId"`
	TaskType    string `json:"taskType"`
	TaskTitle   string `json:"taskTitle"`
	AccountID   int    `json:"accountId,omitempty"`
	Level       string `json:"level"`
	Message     string `json:"message"`
	CreatedAt   string `json:"createdAt"`
}

func (s *TaskStore) handleMonitorFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	afterID, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("after")))
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	if limit <= 0 || limit > 200 {
		limit = 80
	}

	query := `
		SELECT l.id, l.task_id, COALESCE(t.type, ''), COALESCE(t.title, ''),
		       COALESCE(l.account_id, 0), l.level, l.message, l.created_at
		FROM task_logs l
		LEFT JOIN tasks t ON t.id = l.task_id`
	args := []any{}
	if afterID > 0 {
		query += ` WHERE l.id > ?`
		args = append(args, afterID)
	}
	query += ` ORDER BY l.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load feed")
		return
	}
	defer rows.Close()

	out := make([]MonitorLog, 0)
	for rows.Next() {
		var (
			entry     MonitorLog
			createdAt time.Time
		)
		if err := rows.Scan(
			&entry.ID, &entry.TaskID, &entry.TaskType, &entry.TaskTitle,
			&entry.AccountID, &entry.Level, &entry.Message, &createdAt,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load feed")
			return
		}
		entry.CreatedAt = formatDateTime(createdAt)
		out = append(out, entry)
	}
	// Return chronological for the UI stream
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": out})
}

func (s *TaskStore) handleBusyAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	rows, err := s.db.Query(`
		SELECT DISTINCT account_id
		FROM task_items
		WHERE status IN ('pending', 'running')`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load busy accounts")
		return
	}
	defer rows.Close()
	out := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load busy accounts")
			return
		}
		out = append(out, id)
	}
	writeJSON(w, http.StatusOK, map[string]any{"accountIds": out})
}

func decodeTaskInput(w http.ResponseWriter, r *http.Request) (TaskInput, bool) {
	var input TaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return TaskInput{}, false
	}
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	input.Title = strings.TrimSpace(input.Title)
	input.TargetURL = strings.TrimSpace(input.TargetURL)
	input.Content.Text = strings.TrimSpace(input.Content.Text)
	input.Content.Headline = strings.TrimSpace(input.Content.Headline)
	input.Content.LinkURL = strings.TrimSpace(input.Content.LinkURL)
	input.Content.MediaURL = strings.TrimSpace(input.Content.MediaURL)
	switch input.Type {
	case "report", "post", "browse", "login_test", "reply":
	default:
		writeError(w, http.StatusBadRequest, "type must be report, post, browse, login_test, or reply")
		return TaskInput{}, false
	}
	if len(input.AccountIDs) == 0 {
		writeError(w, http.StatusBadRequest, "select at least one account")
		return TaskInput{}, false
	}
	switch input.Type {
	case "report":
		if input.TargetURL == "" {
			writeError(w, http.StatusBadRequest, "target URL is required")
			return TaskInput{}, false
		}
		input.Content = TaskContent{}
	case "reply":
		if input.TargetURL == "" {
			writeError(w, http.StatusBadRequest, "target URL is required")
			return TaskInput{}, false
		}
		if input.Content.Text == "" {
			writeError(w, http.StatusBadRequest, "reply text is required")
			return TaskInput{}, false
		}
		input.Content = TaskContent{Text: input.Content.Text}
	case "post":
		if input.Content.Text == "" {
			writeError(w, http.StatusBadRequest, "post text is required")
			return TaskInput{}, false
		}
		input.TargetURL = ""
	default:
		input.TargetURL = ""
		input.Content = TaskContent{}
	}
	if input.DelayMinSec <= 0 {
		input.DelayMinSec = 5
	}
	if input.DelayMaxSec < input.DelayMinSec {
		input.DelayMaxSec = input.DelayMinSec
	}
	if input.DelayMaxSec > 120 {
		input.DelayMaxSec = 120
	}
	if input.Title == "" {
		input.Title = defaultTaskTitle(input.Type)
	}
	return input, true
}

func defaultTaskTitle(taskType string) string {
	switch taskType {
	case "report":
		return "Report post"
	case "reply":
		return "Reply to post"
	case "post":
		return "Publish post"
	case "browse":
		return "Browse feed"
	case "login_test":
		return "Login test"
	default:
		return "Task"
	}
}

func (s *TaskStore) create(input TaskInput, createdBy string) (Task, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	needed := s.creditsNeeded(len(input.AccountIDs))
	var balance int
	if err := tx.QueryRow(`SELECT balance FROM workspace_credits WHERE id = 1 FOR UPDATE`).Scan(&balance); err != nil {
		return Task{}, fmt.Errorf("credits unavailable: %w", err)
	}
	if balance < needed {
		return Task{}, fmt.Errorf("insufficient credits: need %d, have %d", needed, balance)
	}

	for _, accountID := range input.AccountIDs {
		var exists int
		if err := tx.QueryRow(`SELECT 1 FROM accounts WHERE id = ?`, accountID).Scan(&exists); err != nil {
			if err == sql.ErrNoRows {
				return Task{}, fmt.Errorf("account %d was not found", accountID)
			}
			return Task{}, err
		}
	}

	contentJSON, err := json.Marshal(input.Content)
	if err != nil {
		return Task{}, err
	}

	res, err := tx.Exec(`
		INSERT INTO tasks (type, title, target_url, content, status, delay_min_sec, delay_max_sec, use_account_proxy, show_browser, created_by)
		VALUES (?, ?, ?, ?, 'queued', ?, ?, ?, ?, ?)`,
		input.Type, input.Title, input.TargetURL, string(contentJSON), input.DelayMinSec, input.DelayMaxSec, boolToInt(input.UseAccountProxy), boolToInt(input.ShowBrowser), createdBy,
	)
	if err != nil {
		return Task{}, err
	}
	taskID64, err := res.LastInsertId()
	if err != nil {
		return Task{}, err
	}
	taskID := int(taskID64)

	for _, accountID := range input.AccountIDs {
		if _, err := tx.Exec(`
			INSERT INTO task_items (task_id, account_id, status, message)
			VALUES (?, ?, 'pending', '')`, taskID, accountID); err != nil {
			return Task{}, err
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO task_logs (task_id, level, message)
		VALUES (?, 'info', ?)`, taskID, fmt.Sprintf("Task queued with %d account(s)", len(input.AccountIDs))); err != nil {
		return Task{}, err
	}

	if needed > 0 {
		if _, err := tx.Exec(`UPDATE workspace_credits SET balance = balance - ? WHERE id = 1`, needed); err != nil {
			return Task{}, err
		}
		if _, err := tx.Exec(`
			INSERT INTO task_logs (task_id, level, message)
			VALUES (?, 'info', ?)`, taskID, fmt.Sprintf("Debited %d credit(s)", needed)); err != nil {
			return Task{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Task{}, err
	}

	go s.runTask(taskID)

	task, _, err := s.get(taskID, true)
	return task, err
}

func (s *TaskStore) list(status string) ([]Task, error) {
	query := `
		SELECT t.id, t.type, t.title, t.target_url, t.content, t.status, t.delay_min_sec, t.delay_max_sec,
		       t.use_account_proxy, t.show_browser, t.created_by, t.created_at, t.started_at, t.finished_at,
		       COUNT(i.id) AS account_count,
		       SUM(CASE WHEN i.status IN ('success','failed','cancelled') THEN 1 ELSE 0 END) AS done_count,
		       SUM(CASE WHEN i.status = 'success' THEN 1 ELSE 0 END) AS success_count,
		       SUM(CASE WHEN i.status = 'failed' THEN 1 ELSE 0 END) AS fail_count
		FROM tasks t
		LEFT JOIN task_items i ON i.task_id = t.id`
	args := []any{}
	if status != "" {
		query += ` WHERE t.status = ?`
		args = append(args, status)
	}
	query += `
		GROUP BY t.id, t.type, t.title, t.target_url, t.content, t.status, t.delay_min_sec, t.delay_max_sec,
		         t.use_account_proxy, t.show_browser, t.created_by, t.created_at, t.started_at, t.finished_at
		ORDER BY t.id DESC
		LIMIT 200`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Task, 0)
	for rows.Next() {
		task, err := scanTaskSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

func (s *TaskStore) get(id int, withDetails bool) (Task, bool, error) {
	row := s.db.QueryRow(`
		SELECT t.id, t.type, t.title, t.target_url, t.content, t.status, t.delay_min_sec, t.delay_max_sec,
		       t.use_account_proxy, t.show_browser, t.created_by, t.created_at, t.started_at, t.finished_at,
		       COUNT(i.id) AS account_count,
		       SUM(CASE WHEN i.status IN ('success','failed','cancelled') THEN 1 ELSE 0 END) AS done_count,
		       SUM(CASE WHEN i.status = 'success' THEN 1 ELSE 0 END) AS success_count,
		       SUM(CASE WHEN i.status = 'failed' THEN 1 ELSE 0 END) AS fail_count
		FROM tasks t
		LEFT JOIN task_items i ON i.task_id = t.id
		WHERE t.id = ?
		GROUP BY t.id, t.type, t.title, t.target_url, t.content, t.status, t.delay_min_sec, t.delay_max_sec,
		         t.use_account_proxy, t.show_browser, t.created_by, t.created_at, t.started_at, t.finished_at`, id)
	task, err := scanTaskSummary(row)
	if err == sql.ErrNoRows {
		return Task{}, false, nil
	}
	if err != nil {
		return Task{}, false, err
	}

	ids, err := s.accountIDsForTask(id)
	if err != nil {
		return Task{}, false, err
	}
	task.AccountIDs = ids

	if withDetails {
		items, err := s.listItems(id)
		if err != nil {
			return Task{}, false, err
		}
		task.Items = items
		logs, err := s.listLogs(id)
		if err != nil {
			return Task{}, false, err
		}
		task.Logs = logs
	}
	return task, true, nil
}

func (s *TaskStore) accountIDsForTask(taskID int) ([]int, error) {
	rows, err := s.db.Query(`SELECT account_id FROM task_items WHERE task_id = ? ORDER BY id ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *TaskStore) listItems(taskID int) ([]TaskItem, error) {
	rows, err := s.db.Query(`
		SELECT i.id, i.task_id, i.account_id, a.first_name, a.last_name, a.email, a.platform,
		       i.status, i.message, i.started_at, i.finished_at
		FROM task_items i
		JOIN accounts a ON a.id = i.account_id
		WHERE i.task_id = ?
		ORDER BY i.id ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TaskItem, 0)
	for rows.Next() {
		var (
			item                 TaskItem
			first, last          string
			started, finished    sql.NullTime
		)
		if err := rows.Scan(
			&item.ID, &item.TaskID, &item.AccountID, &first, &last, &item.AccountEmail, &item.Platform,
			&item.Status, &item.Message, &started, &finished,
		); err != nil {
			return nil, err
		}
		item.AccountName = displayName(first, last, item.AccountEmail)
		if started.Valid {
			item.StartedAt = formatDateTime(started.Time)
		}
		if finished.Valid {
			item.FinishedAt = formatDateTime(finished.Time)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *TaskStore) listLogs(taskID int) ([]TaskLog, error) {
	rows, err := s.db.Query(`
		SELECT id, task_id, account_id, level, message, created_at
		FROM task_logs
		WHERE task_id = ?
		ORDER BY id ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TaskLog, 0)
	for rows.Next() {
		var (
			entry     TaskLog
			accountID sql.NullInt64
			createdAt time.Time
		)
		if err := rows.Scan(&entry.ID, &entry.TaskID, &accountID, &entry.Level, &entry.Message, &createdAt); err != nil {
			return nil, err
		}
		if accountID.Valid {
			entry.AccountID = int(accountID.Int64)
		}
		entry.CreatedAt = formatDateTime(createdAt)
		out = append(out, entry)
	}
	return out, rows.Err()
}

func (s *TaskStore) cancel(id int) (Task, error) {
	res, err := s.db.Exec(`
		UPDATE tasks
		SET status = 'cancelled', finished_at = UTC_TIMESTAMP()
		WHERE id = ? AND status IN ('pending', 'queued', 'running')`, id)
	if err != nil {
		return Task{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return Task{}, fmt.Errorf("task cannot be cancelled")
	}
	_, _ = s.db.Exec(`
		UPDATE task_items
		SET status = 'cancelled', message = 'Cancelled', finished_at = UTC_TIMESTAMP()
		WHERE task_id = ? AND status IN ('pending', 'running')`, id)
	_, _ = s.db.Exec(`INSERT INTO task_logs (task_id, level, message) VALUES (?, 'warn', 'Task cancelled')`, id)
	task, _, err := s.get(id, true)
	return task, err
}

func (s *TaskStore) runTask(taskID int) {
	s.mu.Lock()
	if s.running[taskID] {
		s.mu.Unlock()
		return
	}
	s.running[taskID] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.running, taskID)
		s.mu.Unlock()
	}()

	task, found, err := s.get(taskID, true)
	if err != nil || !found {
		return
	}
	if task.Status == "cancelled" {
		return
	}

	_, _ = s.db.Exec(`UPDATE tasks SET status = 'running', started_at = UTC_TIMESTAMP() WHERE id = ?`, taskID)
	s.addLog(taskID, 0, "info", "Worker started")
	if task.UseAccountProxy {
		s.addLog(taskID, 0, "info", "Using each account’s assigned proxy")
	} else {
		s.addLog(taskID, 0, "info", "Running without account proxies")
	}
	if task.ShowBrowser {
		s.addLog(taskID, 0, "info", "Show browser window enabled — live view links will appear per account")
	}

	for _, item := range task.Items {
		fresh, _, err := s.get(taskID, false)
		if err != nil || fresh.Status == "cancelled" {
			s.addLog(taskID, 0, "warn", "Worker stopped")
			return
		}

		_, _ = s.db.Exec(`
			UPDATE task_items SET status = 'running', started_at = UTC_TIMESTAMP(), message = ''
			WHERE id = ?`, item.ID)
		s.addLog(taskID, item.AccountID, "info", fmt.Sprintf("%s started %s", item.AccountName, task.Type))

		delay := task.DelayMinSec
		if task.DelayMaxSec > task.DelayMinSec {
			delay = task.DelayMinSec + int(time.Now().UnixNano()%int64(task.DelayMaxSec-task.DelayMinSec+1))
		}
		time.Sleep(time.Duration(delay) * time.Second)

		fresh, _, err = s.get(taskID, false)
		if err != nil || fresh.Status == "cancelled" {
			_, _ = s.db.Exec(`
				UPDATE task_items SET status = 'cancelled', message = 'Cancelled', finished_at = UTC_TIMESTAMP()
				WHERE id = ? AND status = 'running'`, item.ID)
			return
		}

		result := s.executeItem(task, item)
		if result.OK {
			_, _ = s.db.Exec(`
				UPDATE task_items SET status = 'success', message = ?, finished_at = UTC_TIMESTAMP()
				WHERE id = ?`, result.Message, item.ID)
			s.addLog(taskID, item.AccountID, "success", result.Message)
		} else {
			_, _ = s.db.Exec(`
				UPDATE task_items SET status = 'failed', message = ?, finished_at = UTC_TIMESTAMP()
				WHERE id = ?`, result.Message, item.ID)
			s.addLog(taskID, item.AccountID, "error", result.Message)
		}
	}

	task, _, _ = s.get(taskID, false)
	finalStatus := "completed"
	if task.FailCount > 0 && task.SuccessCount == 0 {
		finalStatus = "failed"
	} else if task.FailCount > 0 {
		finalStatus = "completed"
	}
	_, _ = s.db.Exec(`UPDATE tasks SET status = ?, finished_at = UTC_TIMESTAMP() WHERE id = ? AND status = 'running'`, finalStatus, taskID)
	s.addLog(taskID, 0, "info", fmt.Sprintf("Task %s: %d/%d success, %d failed", finalStatus, task.SuccessCount, task.AccountCount, task.FailCount))
}

func (s *TaskStore) addLog(taskID, accountID int, level, message string) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return
	}
	if accountID > 0 {
		_, _ = s.db.Exec(`INSERT INTO task_logs (task_id, account_id, level, message) VALUES (?, ?, ?, ?)`, taskID, accountID, level, msg)
	} else {
		_, _ = s.db.Exec(`INSERT INTO task_logs (task_id, level, message) VALUES (?, ?, ?)`, taskID, level, msg)
	}
	emitActivity(ActivityInput{
		Source:    "task",
		Level:     level,
		Message:   msg,
		TaskID:    taskID,
		AccountID: accountID,
	})
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTaskSummary(row taskScanner) (Task, error) {
	var (
		task                         Task
		contentRaw                   string
		useProxy, showBrowser        int
		createdAt                    time.Time
		startedAt, finishedAt        sql.NullTime
		accountCount, done, ok, fail sql.NullInt64
	)
	err := row.Scan(
		&task.ID, &task.Type, &task.Title, &task.TargetURL, &contentRaw, &task.Status,
		&task.DelayMinSec, &task.DelayMaxSec, &useProxy, &showBrowser, &task.CreatedBy,
		&createdAt, &startedAt, &finishedAt,
		&accountCount, &done, &ok, &fail,
	)
	if err != nil {
		return Task{}, err
	}
	task.Content = parseTaskContent(contentRaw)
	task.UseAccountProxy = useProxy == 1
	task.ShowBrowser = showBrowser == 1
	task.CreatedAt = formatDateTime(createdAt)
	if startedAt.Valid {
		task.StartedAt = formatDateTime(startedAt.Time)
	}
	if finishedAt.Valid {
		task.FinishedAt = formatDateTime(finishedAt.Time)
	}
	task.AccountCount = int(accountCount.Int64)
	task.DoneCount = int(done.Int64)
	task.SuccessCount = int(ok.Int64)
	task.FailCount = int(fail.Int64)
	task.AccountIDs = []int{}
	return task, nil
}

func parseTaskContent(raw string) TaskContent {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return TaskContent{}
	}
	var content TaskContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		return TaskContent{}
	}
	return content
}

func parseTaskPath(w http.ResponseWriter, path string) (int, string, bool) {
	rest := strings.TrimPrefix(path, "/api/tasks/")
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
		if action != "cancel" && action != "logs" {
			writeError(w, http.StatusNotFound, "not found")
			return 0, "", false
		}
	} else if len(parts) > 2 {
		writeError(w, http.StatusNotFound, "not found")
		return 0, "", false
	}
	return id, action, true
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
