package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	dashboardTrendMonths    = 12
	dashboardListLimit      = 6
	dashboardAttentionLimit = 8
)

// DashboardSummary is the aggregated snapshot rendered by the dashboard page.
// Counts are computed in SQL so the browser never has to download account
// credentials just to display totals.
type DashboardSummary struct {
	GeneratedAt    string                `json:"generatedAt"`
	Accounts       DashboardAccountStats `json:"accounts"`
	Proxies        DashboardProxyStats   `json:"proxies"`
	Gmails         DashboardGmailStats   `json:"gmails"`
	Tasks          DashboardTaskStats    `json:"tasks"`
	Credits        DashboardCredits      `json:"credits"`
	Routing        DashboardRouting      `json:"routing"`
	Platforms      []DashboardPlatform   `json:"platforms"`
	Countries      []DashboardCountry    `json:"countries"`
	Trend          []DashboardTrendPoint `json:"trend"`
	TopProxies     []DashboardProxyLoad  `json:"topProxies"`
	RecentAccounts []DashboardRecent     `json:"recentAccounts"`
	RecentTasks    []DashboardRecentTask `json:"recentTasks"`
	Attention      []DashboardAttention  `json:"attention"`
}

type DashboardAccountStats struct {
	Total       int `json:"total"`
	Active      int `json:"active"`
	Expired     int `json:"expired"`
	Error       int `json:"error"`
	Revoked     int `json:"revoked"`
	Attention   int `json:"attention"`
	AddedLast30 int `json:"addedLast30"`
	AddedLast90 int `json:"addedLast90"`
}

type DashboardProxyStats struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
	Error    int `json:"error"`
	Issues   int `json:"issues"`
	Unused   int `json:"unused"`
}

type DashboardGmailStats struct {
	Total             int `json:"total"`
	Active            int `json:"active"`
	Inactive          int `json:"inactive"`
	Error             int `json:"error"`
	Issues            int `json:"issues"`
	Unlinked          int `json:"unlinked"`
	UnmanagedAccounts int `json:"unmanagedAccounts"`
}

type DashboardTaskStats struct {
	Total       int `json:"total"`
	Queued      int `json:"queued"`
	Running     int `json:"running"`
	Completed   int `json:"completed"`
	Failed      int `json:"failed"`
	Cancelled   int `json:"cancelled"`
	Active      int `json:"active"`
	Last24h     int `json:"last24h"`
	BusyAccounts int `json:"busyAccounts"`
}

type DashboardCredits struct {
	Balance   int    `json:"balance"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type DashboardRouting struct {
	Auto       int `json:"auto"`
	Manual     int `json:"manual"`
	None       int `json:"none"`
	Routed     int `json:"routed"`
	Unassigned int `json:"unassigned"`
}

type DashboardPlatform struct {
	Platform  string `json:"platform"`
	Total     int    `json:"total"`
	Active    int    `json:"active"`
	Attention int    `json:"attention"`
}

type DashboardCountry struct {
	Country  string `json:"country"`
	Proxies  int    `json:"proxies"`
	Accounts int    `json:"accounts"`
}

type DashboardTrendPoint struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

type DashboardProxyLoad struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Country      string `json:"country"`
	Status       string `json:"status"`
	AccountCount int    `json:"accountCount"`
}

type DashboardRecent struct {
	ID           int    `json:"id"`
	Platform     string `json:"platform"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Status       string `json:"status"`
	ProxyMode    string `json:"proxyMode"`
	ProxyName    string `json:"proxyName"`
	ProxyCountry string `json:"proxyCountry"`
	ConnectedBy  string `json:"connectedBy"`
	ConnectedAt  string `json:"connectedAt"`
}

type DashboardRecentTask struct {
	ID           int    `json:"id"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	AccountCount int    `json:"accountCount"`
	DoneCount    int    `json:"doneCount"`
	SuccessCount int    `json:"successCount"`
	FailCount    int    `json:"failCount"`
	CreatedBy    string `json:"createdBy"`
	CreatedAt    string `json:"createdAt"`
	TargetURL    string `json:"targetUrl"`
}

// DashboardAttention carries raw status values plus a short detail string so the
// frontend can render labels with its own shared constants.
type DashboardAttention struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	EntityID int    `json:"entityId"`
	Tone     string `json:"tone"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Detail   string `json:"detail"`
}

type DashboardStore struct {
	db *sql.DB
}

func newDashboardStore(db *sql.DB) *DashboardStore {
	return &DashboardStore{db: db}
}

func (s *DashboardStore) summary() (DashboardSummary, error) {
	now := time.Now().UTC()
	out := DashboardSummary{GeneratedAt: now.Format(time.RFC3339)}

	accounts, routing, err := s.accountStats(now)
	if err != nil {
		return DashboardSummary{}, err
	}
	out.Accounts = accounts
	out.Routing = routing

	if out.Proxies, err = s.proxyStats(); err != nil {
		return DashboardSummary{}, err
	}
	if out.Gmails, err = s.gmailStats(); err != nil {
		return DashboardSummary{}, err
	}
	if out.Tasks, err = s.taskStats(); err != nil {
		return DashboardSummary{}, err
	}
	if out.Credits, err = s.credits(); err != nil {
		return DashboardSummary{}, err
	}
	if out.Platforms, err = s.platforms(); err != nil {
		return DashboardSummary{}, err
	}
	if out.Countries, err = s.countries(); err != nil {
		return DashboardSummary{}, err
	}
	if out.Trend, err = s.trend(now, dashboardTrendMonths); err != nil {
		return DashboardSummary{}, err
	}
	if out.TopProxies, err = s.topProxies(); err != nil {
		return DashboardSummary{}, err
	}
	if out.RecentAccounts, err = s.recentAccounts(); err != nil {
		return DashboardSummary{}, err
	}
	if out.RecentTasks, err = s.recentTasks(); err != nil {
		return DashboardSummary{}, err
	}
	if out.Attention, err = s.attention(); err != nil {
		return DashboardSummary{}, err
	}
	return out, nil
}

func (s *DashboardStore) accountStats(now time.Time) (DashboardAccountStats, DashboardRouting, error) {
	var (
		stats   DashboardAccountStats
		routing DashboardRouting
	)

	err := s.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(status = 'active'), 0),
			COALESCE(SUM(status = 'expired'), 0),
			COALESCE(SUM(status = 'error'), 0),
			COALESCE(SUM(status = 'revoked'), 0),
			COALESCE(SUM(proxy_mode = 'auto'), 0),
			COALESCE(SUM(proxy_mode = 'manual'), 0),
			COALESCE(SUM(proxy_mode = 'none'), 0),
			COALESCE(SUM(proxy_id IS NULL OR proxy_mode = 'none'), 0),
			COALESCE(SUM(connected_at >= ?), 0),
			COALESCE(SUM(connected_at >= ?), 0)
		FROM accounts`,
		now.AddDate(0, 0, -30).Format("2006-01-02"),
		now.AddDate(0, 0, -90).Format("2006-01-02"),
	).Scan(
		&stats.Total,
		&stats.Active,
		&stats.Expired,
		&stats.Error,
		&stats.Revoked,
		&routing.Auto,
		&routing.Manual,
		&routing.None,
		&routing.Unassigned,
		&stats.AddedLast30,
		&stats.AddedLast90,
	)
	if err != nil {
		return stats, routing, err
	}

	stats.Attention = stats.Expired + stats.Error
	routing.Routed = stats.Total - routing.Unassigned
	return stats, routing, nil
}

func (s *DashboardStore) proxyStats() (DashboardProxyStats, error) {
	var stats DashboardProxyStats

	err := s.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(status = 'active'), 0),
			COALESCE(SUM(status = 'inactive'), 0),
			COALESCE(SUM(status = 'error'), 0)
		FROM proxies`).Scan(&stats.Total, &stats.Active, &stats.Inactive, &stats.Error)
	if err != nil {
		return stats, err
	}
	stats.Issues = stats.Inactive + stats.Error

	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM proxies p
		LEFT JOIN accounts a ON a.proxy_id = p.id
		WHERE a.id IS NULL`).Scan(&stats.Unused)
	return stats, err
}

func (s *DashboardStore) gmailStats() (DashboardGmailStats, error) {
	var stats DashboardGmailStats

	err := s.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(status = 'active'), 0),
			COALESCE(SUM(status = 'inactive'), 0),
			COALESCE(SUM(status = 'error'), 0)
		FROM gmails`).Scan(&stats.Total, &stats.Active, &stats.Inactive, &stats.Error)
	if err != nil {
		return stats, err
	}
	stats.Issues = stats.Inactive + stats.Error

	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM gmails g
		LEFT JOIN accounts a ON LOWER(a.email) = LOWER(g.email)
		WHERE a.id IS NULL`).Scan(&stats.Unlinked)
	if err != nil {
		return stats, err
	}

	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM accounts a
		LEFT JOIN gmails g ON LOWER(g.email) = LOWER(a.email)
		WHERE g.id IS NULL`).Scan(&stats.UnmanagedAccounts)
	return stats, err
}

func (s *DashboardStore) taskStats() (DashboardTaskStats, error) {
	var stats DashboardTaskStats
	err := s.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(status IN ('pending', 'queued')), 0),
			COALESCE(SUM(status = 'running'), 0),
			COALESCE(SUM(status = 'completed'), 0),
			COALESCE(SUM(status = 'failed'), 0),
			COALESCE(SUM(status = 'cancelled'), 0),
			COALESCE(SUM(created_at >= UTC_TIMESTAMP() - INTERVAL 1 DAY), 0)
		FROM tasks`).Scan(
		&stats.Total,
		&stats.Queued,
		&stats.Running,
		&stats.Completed,
		&stats.Failed,
		&stats.Cancelled,
		&stats.Last24h,
	)
	if err != nil {
		// Tasks table may not exist yet on older DBs mid-migration.
		if strings.Contains(err.Error(), "doesn't exist") {
			return DashboardTaskStats{}, nil
		}
		return stats, err
	}
	stats.Active = stats.Queued + stats.Running

	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT account_id)
		FROM task_items
		WHERE status IN ('pending', 'running')`).Scan(&stats.BusyAccounts)
	if err != nil {
		if strings.Contains(err.Error(), "doesn't exist") {
			return stats, nil
		}
		return stats, err
	}
	return stats, nil
}

func (s *DashboardStore) credits() (DashboardCredits, error) {
	var (
		out       DashboardCredits
		updatedAt sql.NullTime
	)
	err := s.db.QueryRow(`SELECT balance, updated_at FROM workspace_credits WHERE id = 1`).Scan(&out.Balance, &updatedAt)
	if err == sql.ErrNoRows {
		return DashboardCredits{Balance: 0}, nil
	}
	if err != nil {
		if strings.Contains(err.Error(), "doesn't exist") {
			return DashboardCredits{}, nil
		}
		return out, err
	}
	if updatedAt.Valid {
		out.UpdatedAt = formatDateTime(updatedAt.Time)
	}
	return out, nil
}

func (s *DashboardStore) recentTasks() ([]DashboardRecentTask, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.type, t.title, t.status, t.target_url, t.created_by, t.created_at,
		       COUNT(i.id) AS account_count,
		       SUM(CASE WHEN i.status IN ('success','failed','cancelled') THEN 1 ELSE 0 END) AS done_count,
		       SUM(CASE WHEN i.status = 'success' THEN 1 ELSE 0 END) AS success_count,
		       SUM(CASE WHEN i.status = 'failed' THEN 1 ELSE 0 END) AS fail_count
		FROM tasks t
		LEFT JOIN task_items i ON i.task_id = t.id
		GROUP BY t.id, t.type, t.title, t.status, t.target_url, t.created_by, t.created_at
		ORDER BY t.id DESC
		LIMIT ?`, dashboardListLimit)
	if err != nil {
		if strings.Contains(err.Error(), "doesn't exist") {
			return []DashboardRecentTask{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	out := make([]DashboardRecentTask, 0)
	for rows.Next() {
		var (
			item                         DashboardRecentTask
			createdAt                    time.Time
			accountCount, done, ok, fail sql.NullInt64
		)
		if err := rows.Scan(
			&item.ID, &item.Type, &item.Title, &item.Status, &item.TargetURL, &item.CreatedBy, &createdAt,
			&accountCount, &done, &ok, &fail,
		); err != nil {
			return nil, err
		}
		item.CreatedAt = formatDateTime(createdAt)
		item.AccountCount = int(accountCount.Int64)
		item.DoneCount = int(done.Int64)
		item.SuccessCount = int(ok.Int64)
		item.FailCount = int(fail.Int64)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *DashboardStore) platforms() ([]DashboardPlatform, error) {
	rows, err := s.db.Query(`
		SELECT
			platform,
			COUNT(*),
			COALESCE(SUM(status = 'active'), 0),
			COALESCE(SUM(status IN ('expired', 'error')), 0)
		FROM accounts
		GROUP BY platform
		ORDER BY COUNT(*) DESC, platform ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DashboardPlatform, 0)
	for rows.Next() {
		var item DashboardPlatform
		if err := rows.Scan(&item.Platform, &item.Total, &item.Active, &item.Attention); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *DashboardStore) countries() ([]DashboardCountry, error) {
	rows, err := s.db.Query(`
		SELECT p.country, COUNT(DISTINCT p.id), COUNT(a.id)
		FROM proxies p
		LEFT JOIN accounts a ON a.proxy_id = p.id
		WHERE p.country <> ''
		GROUP BY p.country
		ORDER BY COUNT(a.id) DESC, COUNT(DISTINCT p.id) DESC, p.country ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DashboardCountry, 0)
	for rows.Next() {
		var item DashboardCountry
		if err := rows.Scan(&item.Country, &item.Proxies, &item.Accounts); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// trend returns one point per month for the trailing window, including months
// with no connections so the chart keeps an even time axis.
func (s *DashboardStore) trend(now time.Time, months int) ([]DashboardTrendPoint, error) {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -(months - 1), 0)

	rows, err := s.db.Query(`
		SELECT DATE_FORMAT(connected_at, '%Y-%m'), COUNT(*)
		FROM accounts
		WHERE connected_at >= ?
		GROUP BY DATE_FORMAT(connected_at, '%Y-%m')`, start.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var (
			month string
			count int
		)
		if err := rows.Scan(&month, &count); err != nil {
			return nil, err
		}
		counts[month] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]DashboardTrendPoint, 0, months)
	for i := 0; i < months; i++ {
		month := start.AddDate(0, i, 0).Format("2006-01")
		out = append(out, DashboardTrendPoint{Month: month, Count: counts[month]})
	}
	return out, nil
}

func (s *DashboardStore) topProxies() ([]DashboardProxyLoad, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.protocol, p.host, p.port, p.country, p.status, COUNT(a.id)
		FROM proxies p
		LEFT JOIN accounts a ON a.proxy_id = p.id
		GROUP BY p.id, p.name, p.protocol, p.host, p.port, p.country, p.status
		ORDER BY COUNT(a.id) DESC, p.name ASC
		LIMIT ?`, dashboardListLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DashboardProxyLoad, 0)
	for rows.Next() {
		var item DashboardProxyLoad
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Protocol,
			&item.Host,
			&item.Port,
			&item.Country,
			&item.Status,
			&item.AccountCount,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *DashboardStore) recentAccounts() ([]DashboardRecent, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.platform, a.first_name, a.last_name, a.email, a.status, a.proxy_mode,
		       COALESCE(p.name, ''), COALESCE(p.country, ''), a.connected_by, a.connected_at
		FROM accounts a
		LEFT JOIN proxies p ON p.id = a.proxy_id
		ORDER BY a.connected_at DESC, a.id DESC
		LIMIT ?`, dashboardListLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DashboardRecent, 0)
	for rows.Next() {
		var (
			item        DashboardRecent
			firstName   string
			lastName    string
			connectedAt sql.NullTime
		)
		err := rows.Scan(
			&item.ID,
			&item.Platform,
			&firstName,
			&lastName,
			&item.Email,
			&item.Status,
			&item.ProxyMode,
			&item.ProxyName,
			&item.ProxyCountry,
			&item.ConnectedBy,
			&connectedAt,
		)
		if err != nil {
			return nil, err
		}
		item.Name = displayName(firstName, lastName, item.Email)
		item.ConnectedAt = formatDate(connectedAt)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *DashboardStore) attention() ([]DashboardAttention, error) {
	out := make([]DashboardAttention, 0)

	accountRows, err := s.db.Query(`
		SELECT id, platform, first_name, last_name, email, status
		FROM accounts
		WHERE status IN ('error', 'expired')
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer accountRows.Close()

	for accountRows.Next() {
		var (
			id                          int
			platform, first, last, mail string
			status                      string
		)
		if err := accountRows.Scan(&id, &platform, &first, &last, &mail, &status); err != nil {
			return nil, err
		}
		out = append(out, DashboardAttention{
			ID:       fmt.Sprintf("account-%d", id),
			Kind:     "account",
			EntityID: id,
			Tone:     attentionTone(status),
			Title:    displayName(first, last, mail),
			Status:   status,
			Detail:   platform,
		})
	}
	if err := accountRows.Err(); err != nil {
		return nil, err
	}

	proxyRows, err := s.db.Query(`
		SELECT id, name, host, port, status
		FROM proxies
		WHERE status IN ('error', 'inactive')
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer proxyRows.Close()

	for proxyRows.Next() {
		var (
			id     int
			name   string
			host   string
			port   int
			status string
		)
		if err := proxyRows.Scan(&id, &name, &host, &port, &status); err != nil {
			return nil, err
		}
		out = append(out, DashboardAttention{
			ID:       fmt.Sprintf("proxy-%d", id),
			Kind:     "proxy",
			EntityID: id,
			Tone:     attentionTone(status),
			Title:    name,
			Status:   status,
			Detail:   fmt.Sprintf("%s:%d", host, port),
		})
	}
	if err := proxyRows.Err(); err != nil {
		return nil, err
	}

	gmailRows, err := s.db.Query(`
		SELECT id, email, label, status
		FROM gmails
		WHERE status IN ('error', 'inactive')
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer gmailRows.Close()

	for gmailRows.Next() {
		var (
			id           int
			email, label string
			status       string
		)
		if err := gmailRows.Scan(&id, &email, &label, &status); err != nil {
			return nil, err
		}
		title := label
		if title == "" {
			title = email
		}
		out = append(out, DashboardAttention{
			ID:       fmt.Sprintf("gmail-%d", id),
			Kind:     "gmail",
			EntityID: id,
			Tone:     attentionTone(status),
			Title:    title,
			Status:   status,
			Detail:   email,
		})
	}
	if err := gmailRows.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Tone == "danger" && out[j].Tone != "danger"
	})
	if len(out) > dashboardAttentionLimit {
		out = out[:dashboardAttentionLimit]
	}
	return out, nil
}

func (s *DashboardStore) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	summary, err := s.summary()
	if err != nil {
		log.Printf("dashboard summary: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to build dashboard summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func attentionTone(status string) string {
	if status == "error" {
		return "danger"
	}
	return "warn"
}

func displayName(first, last, fallback string) string {
	name := strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
	if name != "" {
		return name
	}
	return fallback
}
