package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type PostStore struct {
	db *sql.DB
}

type TrackedPost struct {
	ID               int      `json:"id"`
	Platform         string   `json:"platform"`
	URL              string   `json:"url"`
	Title            string   `json:"title"`
	Author           string   `json:"author"`
	BodyText         string   `json:"bodyText"`
	Status           string   `json:"status"`
	Analysis         Analysis `json:"analysis"`
	SuggestedReplies []string `json:"suggestedReplies"`
	ReplyCount       int      `json:"replyCount"`
	PostCount        int      `json:"postCount"`
	CreatedBy        string   `json:"createdBy"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
}

type Analysis struct {
	Summary   string   `json:"summary"`
	Sentiment string   `json:"sentiment"`
	Tone      string   `json:"tone"`
	Topics    []string `json:"topics"`
	Risks     []string `json:"risks"`
	Hooks     []string `json:"hooks"`
	Language  string   `json:"language"`
	Engine    string   `json:"engine"`
}

type AccountEngagement struct {
	ID            int    `json:"id"`
	AccountID     int    `json:"accountId"`
	AccountName   string `json:"accountName"`
	AccountEmail  string `json:"accountEmail"`
	Platform      string `json:"platform"`
	TrackedPostID int    `json:"trackedPostId,omitempty"`
	Kind          string `json:"kind"`
	TargetURL     string `json:"targetUrl"`
	Content       string `json:"content"`
	TaskID        int    `json:"taskId,omitempty"`
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
}

type TrackedPostInput struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	BodyText string `json:"bodyText"`
	Status   string `json:"status"`
}

type EngagementInput struct {
	AccountID     int
	TrackedPostID int
	Kind          string
	TargetURL     string
	Content       string
	TaskID        int
	TaskItemID    int
	Status        string
}

func newPostStore(db *sql.DB) *PostStore {
	return &PostStore{db: db}
}

func (s *PostStore) ensureSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tracked_posts (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			platform VARCHAR(40) NOT NULL DEFAULT 'facebook',
			url TEXT NOT NULL,
			title VARCHAR(255) NOT NULL DEFAULT '',
			author VARCHAR(255) NOT NULL DEFAULT '',
			body_text MEDIUMTEXT NOT NULL,
			status ENUM('watching','archived') NOT NULL DEFAULT 'watching',
			analysis_json MEDIUMTEXT NULL,
			suggested_replies_json MEDIUMTEXT NULL,
			created_by VARCHAR(120) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY idx_tracked_posts_status (status),
			KEY idx_tracked_posts_platform (platform),
			KEY idx_tracked_posts_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS account_engagements (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			account_id INT UNSIGNED NOT NULL,
			tracked_post_id INT UNSIGNED NULL,
			kind ENUM('post','reply','like','report','browse','login') NOT NULL,
			target_url TEXT NULL,
			content MEDIUMTEXT NULL,
			task_id INT UNSIGNED NULL,
			task_item_id INT UNSIGNED NULL,
			status ENUM('success','failed') NOT NULL DEFAULT 'success',
			meta_json TEXT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY idx_engagements_account (account_id),
			KEY idx_engagements_kind (kind),
			KEY idx_engagements_post (tracked_post_id),
			KEY idx_engagements_task (task_id),
			KEY idx_engagements_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostStore) handlePosts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		rows, err := s.list(status)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load posts")
			return
		}
		writeJSON(w, http.StatusOK, rows)
	case http.MethodPost:
		var input TrackedPostInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		actor := ""
		if u := currentUser(r); u != nil {
			actor = u.Email
		}
		post, err := s.create(input, actor)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		emitActivity(ActivityInput{
			Source:  "system",
			Level:   "info",
			Message: fmt.Sprintf("Tracked post #%d added for analysis", post.ID),
			Actor:   actor,
		})
		writeJSON(w, http.StatusCreated, post)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *PostStore) handlePostByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/posts/")
	rest = strings.Trim(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if parts[0] == "suggest" {
		s.handleSuggest(w, r)
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			post, found, err := s.get(id)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to load post")
				return
			}
			if !found {
				writeError(w, http.StatusNotFound, "post not found")
				return
			}
			writeJSON(w, http.StatusOK, post)
		case http.MethodPut:
			var input TrackedPostInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
			post, err := s.update(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, post)
		case http.MethodDelete:
			if err := s.delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if parts[1] == "analyze" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		post, err := s.analyzeAndSave(id)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		emitActivity(ActivityInput{
			Source:  "system",
			Level:   "success",
			Message: fmt.Sprintf("Analyzed tracked post #%d (%s)", post.ID, post.Analysis.Sentiment),
		})
		writeJSON(w, http.StatusOK, post)
		return
	}
	writeError(w, http.StatusNotFound, "not found")
}

func (s *PostStore) handleEngagements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	accountID, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("accountId")))
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	rows, err := s.listEngagements(accountID, kind, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load engagements")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *PostStore) handleSuggest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input struct {
		BodyText string `json:"bodyText"`
		Platform string `json:"platform"`
		Tone     string `json:"tone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	analysis, replies := analyzePostContent(input.Platform, input.BodyText, input.Tone)
	if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" && strings.TrimSpace(input.BodyText) != "" {
		if aiAnalysis, aiReplies, err := analyzeWithOpenAI(key, input.Platform, input.BodyText); err == nil {
			analysis = aiAnalysis
			replies = aiReplies
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"analysis":         analysis,
		"suggestedReplies": replies,
	})
}

func (s *PostStore) list(status string) ([]TrackedPost, error) {
	query := `
		SELECT p.id, p.platform, p.url, p.title, p.author, p.body_text, p.status,
		       p.analysis_json, p.suggested_replies_json, p.created_by, p.created_at, p.updated_at,
		       COALESCE(SUM(e.kind = 'reply'), 0), COALESCE(SUM(e.kind = 'post'), 0)
		FROM tracked_posts p
		LEFT JOIN account_engagements e ON e.tracked_post_id = p.id AND e.status = 'success'`
	args := []any{}
	if status != "" {
		query += ` WHERE p.status = ?`
		args = append(args, status)
	}
	query += `
		GROUP BY p.id
		ORDER BY p.updated_at DESC, p.id DESC
		LIMIT 300`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TrackedPost, 0)
	for rows.Next() {
		post, err := scanTrackedPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, post)
	}
	return out, rows.Err()
}

func (s *PostStore) get(id int) (TrackedPost, bool, error) {
	row := s.db.QueryRow(`
		SELECT p.id, p.platform, p.url, p.title, p.author, p.body_text, p.status,
		       p.analysis_json, p.suggested_replies_json, p.created_by, p.created_at, p.updated_at,
		       COALESCE((SELECT COUNT(*) FROM account_engagements e WHERE e.tracked_post_id = p.id AND e.kind = 'reply' AND e.status = 'success'), 0),
		       COALESCE((SELECT COUNT(*) FROM account_engagements e WHERE e.tracked_post_id = p.id AND e.kind = 'post' AND e.status = 'success'), 0)
		FROM tracked_posts p
		WHERE p.id = ?`, id)
	post, err := scanTrackedPost(row)
	if err == sql.ErrNoRows {
		return TrackedPost{}, false, nil
	}
	if err != nil {
		return TrackedPost{}, false, err
	}
	return post, true, nil
}

func (s *PostStore) create(input TrackedPostInput, actor string) (TrackedPost, error) {
	input = normalizeTrackedPostInput(input)
	if input.URL == "" && input.BodyText == "" {
		return TrackedPost{}, fmt.Errorf("url or post text is required")
	}
	analysis, replies := analyzePostContent(input.Platform, input.BodyText, "")
	analysisJSON, _ := json.Marshal(analysis)
	repliesJSON, _ := json.Marshal(replies)
	res, err := s.db.Exec(`
		INSERT INTO tracked_posts (platform, url, title, author, body_text, status, analysis_json, suggested_replies_json, created_by)
		VALUES (?, ?, ?, ?, ?, 'watching', ?, ?, ?)`,
		input.Platform, input.URL, input.Title, input.Author, input.BodyText, string(analysisJSON), string(repliesJSON), actor,
	)
	if err != nil {
		return TrackedPost{}, err
	}
	id, _ := res.LastInsertId()
	post, _, err := s.get(int(id))
	return post, err
}

func (s *PostStore) update(id int, input TrackedPostInput) (TrackedPost, error) {
	existing, found, err := s.get(id)
	if err != nil {
		return TrackedPost{}, err
	}
	if !found {
		return TrackedPost{}, fmt.Errorf("post not found")
	}
	input = normalizeTrackedPostInput(input)
	if input.URL == "" {
		input.URL = existing.URL
	}
	if input.BodyText == "" {
		input.BodyText = existing.BodyText
	}
	if input.Title == "" {
		input.Title = existing.Title
	}
	if input.Author == "" {
		input.Author = existing.Author
	}
	if input.Platform == "" {
		input.Platform = existing.Platform
	}
	status := existing.Status
	if input.Status == "watching" || input.Status == "archived" {
		status = input.Status
	}
	_, err = s.db.Exec(`
		UPDATE tracked_posts
		SET platform = ?, url = ?, title = ?, author = ?, body_text = ?, status = ?
		WHERE id = ?`,
		input.Platform, input.URL, input.Title, input.Author, input.BodyText, status, id,
	)
	if err != nil {
		return TrackedPost{}, err
	}
	post, _, err := s.get(id)
	return post, err
}

func (s *PostStore) delete(id int) error {
	res, err := s.db.Exec(`DELETE FROM tracked_posts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("post not found")
	}
	return nil
}

func (s *PostStore) analyzeAndSave(id int) (TrackedPost, error) {
	post, found, err := s.get(id)
	if err != nil {
		return TrackedPost{}, err
	}
	if !found {
		return TrackedPost{}, fmt.Errorf("post not found")
	}
	analysis, replies := analyzePostContent(post.Platform, post.BodyText, "")
	if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" && strings.TrimSpace(post.BodyText) != "" {
		if aiAnalysis, aiReplies, aiErr := analyzeWithOpenAI(key, post.Platform, post.BodyText); aiErr != nil {
			log.Printf("openai analyze post %d: %v", id, aiErr)
			analysis.Engine = "heuristic"
			analysis.Risks = appendUnique(analysis.Risks, "AI analysis unavailable — used local heuristics")
		} else {
			analysis = aiAnalysis
			replies = aiReplies
		}
	}
	analysisJSON, _ := json.Marshal(analysis)
	repliesJSON, _ := json.Marshal(replies)
	_, err = s.db.Exec(`
		UPDATE tracked_posts
		SET analysis_json = ?, suggested_replies_json = ?
		WHERE id = ?`, string(analysisJSON), string(repliesJSON), id)
	if err != nil {
		return TrackedPost{}, err
	}
	updated, _, err := s.get(id)
	return updated, err
}

func (s *PostStore) listEngagements(accountID int, kind string, limit int) ([]AccountEngagement, error) {
	if limit <= 0 || limit > 500 {
		limit = 150
	}
	query := `
		SELECT e.id, e.account_id,
		       TRIM(CONCAT(COALESCE(a.first_name,''), ' ', COALESCE(a.last_name,''))),
		       COALESCE(a.email,''), COALESCE(a.platform,''),
		       COALESCE(e.tracked_post_id, 0), e.kind, COALESCE(e.target_url,''),
		       COALESCE(e.content,''), COALESCE(e.task_id, 0), e.status, e.created_at
		FROM account_engagements e
		LEFT JOIN accounts a ON a.id = e.account_id
		WHERE 1=1`
	args := []any{}
	if accountID > 0 {
		query += ` AND e.account_id = ?`
		args = append(args, accountID)
	}
	if kind != "" {
		query += ` AND e.kind = ?`
		args = append(args, kind)
	}
	query += ` ORDER BY e.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AccountEngagement, 0)
	for rows.Next() {
		var (
			entry     AccountEngagement
			createdAt time.Time
		)
		if err := rows.Scan(
			&entry.ID, &entry.AccountID, &entry.AccountName, &entry.AccountEmail, &entry.Platform,
			&entry.TrackedPostID, &entry.Kind, &entry.TargetURL, &entry.Content, &entry.TaskID,
			&entry.Status, &createdAt,
		); err != nil {
			return nil, err
		}
		entry.AccountName = strings.TrimSpace(entry.AccountName)
		if entry.AccountName == "" {
			entry.AccountName = entry.AccountEmail
		}
		entry.CreatedAt = formatDateTime(createdAt)
		out = append(out, entry)
	}
	return out, rows.Err()
}

func (s *PostStore) recordEngagement(in EngagementInput) {
	if s == nil || in.AccountID < 1 || in.Kind == "" {
		return
	}
	status := in.Status
	if status != "failed" {
		status = "success"
	}
	var tracked any
	if in.TrackedPostID > 0 {
		tracked = in.TrackedPostID
	} else if url := strings.TrimSpace(in.TargetURL); url != "" {
		var id int
		_ = s.db.QueryRow(`SELECT id FROM tracked_posts WHERE url = ? ORDER BY id DESC LIMIT 1`, url).Scan(&id)
		if id > 0 {
			tracked = id
		}
	}
	var taskID, itemID any
	if in.TaskID > 0 {
		taskID = in.TaskID
	}
	if in.TaskItemID > 0 {
		itemID = in.TaskItemID
	}
	_, err := s.db.Exec(`
		INSERT INTO account_engagements
		  (account_id, tracked_post_id, kind, target_url, content, task_id, task_item_id, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.AccountID, tracked, in.Kind, strings.TrimSpace(in.TargetURL), strings.TrimSpace(in.Content),
		taskID, itemID, status,
	)
	if err != nil {
		log.Printf("record engagement: %v", err)
	}
}

func normalizeTrackedPostInput(input TrackedPostInput) TrackedPostInput {
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	if input.Platform == "" {
		input.Platform = "facebook"
	}
	input.URL = strings.TrimSpace(input.URL)
	input.Title = strings.TrimSpace(input.Title)
	input.Author = strings.TrimSpace(input.Author)
	input.BodyText = strings.TrimSpace(input.BodyText)
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	if input.Title == "" && input.BodyText != "" {
		input.Title = truncateRunes(input.BodyText, 80)
	}
	return input
}

type trackedScanner interface {
	Scan(dest ...any) error
}

func scanTrackedPost(row trackedScanner) (TrackedPost, error) {
	var (
		post                         TrackedPost
		analysisRaw, repliesRaw      sql.NullString
		createdAt, updatedAt         time.Time
		replyCount, postCount        int
	)
	err := row.Scan(
		&post.ID, &post.Platform, &post.URL, &post.Title, &post.Author, &post.BodyText, &post.Status,
		&analysisRaw, &repliesRaw, &post.CreatedBy, &createdAt, &updatedAt,
		&replyCount, &postCount,
	)
	if err != nil {
		return TrackedPost{}, err
	}
	post.ReplyCount = replyCount
	post.PostCount = postCount
	post.CreatedAt = formatDateTime(createdAt)
	post.UpdatedAt = formatDateTime(updatedAt)
	post.SuggestedReplies = []string{}
	if analysisRaw.Valid && analysisRaw.String != "" {
		_ = json.Unmarshal([]byte(analysisRaw.String), &post.Analysis)
	}
	if repliesRaw.Valid && repliesRaw.String != "" {
		_ = json.Unmarshal([]byte(repliesRaw.String), &post.SuggestedReplies)
	}
	return post, nil
}

func analyzePostContent(platform, body, preferredTone string) (Analysis, []string) {
	text := strings.TrimSpace(body)
	analysis := Analysis{
		Summary:   "Paste the post text to unlock a fuller analysis.",
		Sentiment: "neutral",
		Tone:      "neutral",
		Topics:    []string{},
		Risks:     []string{},
		Hooks:     []string{},
		Language:  detectLanguageHint(text),
		Engine:    "heuristic",
	}
	if text == "" {
		replies := []string{
			"Interesting point — thanks for sharing.",
			"Curious to hear more about your experience with this.",
			"This resonates. What would you recommend as a next step?",
		}
		return analysis, replies
	}

	lower := strings.ToLower(text)
	words := strings.Fields(text)
	analysis.Summary = fmt.Sprintf("%d-word %s post. %s", len(words), firstNonEmpty(platform, "social"), truncateRunes(text, 160))

	pos := countHits(lower, []string{"great", "love", "amazing", " congrats", "proud", "happy", "excellent", "win", "success", "thank"})
	neg := countHits(lower, []string{"hate", "angry", "terrible", "worst", "scam", "fraud", "angry", "disappointed", "broken", "fail", "problem"})
	switch {
	case pos > neg+1:
		analysis.Sentiment = "positive"
		analysis.Tone = "supportive"
	case neg > pos+1:
		analysis.Sentiment = "negative"
		analysis.Tone = "empathetic"
	default:
		analysis.Sentiment = "neutral"
		analysis.Tone = "curious"
	}
	if preferredTone != "" {
		analysis.Tone = preferredTone
	}

	topicMap := map[string][]string{
		"product":   {"product", "launch", "feature", "app", "tool"},
		"politics":  {"election", "government", "president", "vote", "policy"},
		"business":  {"business", "startup", "revenue", "customer", "market"},
		"tech":      {"ai", "software", "code", "developer", "cloud"},
		"lifestyle": {"family", "travel", "food", "health", "fitness"},
	}
	for topic, keys := range topicMap {
		if countHits(lower, keys) > 0 {
			analysis.Topics = append(analysis.Topics, topic)
		}
	}
	if len(analysis.Topics) == 0 {
		analysis.Topics = []string{"general"}
	}

	if strings.Contains(lower, "http") {
		analysis.Hooks = append(analysis.Hooks, "Contains a link — replies that ask a specific question tend to perform better")
	}
	if strings.Contains(lower, "?") {
		analysis.Hooks = append(analysis.Hooks, "Author asked a question — answer it directly")
	}
	if len(words) < 12 {
		analysis.Hooks = append(analysis.Hooks, "Short post — keep replies concise")
	}
	if neg > 0 {
		analysis.Risks = append(analysis.Risks, "Negative tone detected — avoid sounding dismissive")
	}
	if countHits(lower, []string{"kill", "hate", "idiot", "stupid"}) > 0 {
		analysis.Risks = append(analysis.Risks, "Hostile language nearby — stay professional")
	}

	replies := suggestReplies(analysis, text)
	return analysis, replies
}

func suggestReplies(analysis Analysis, body string) []string {
	snippet := truncateRunes(strings.TrimSpace(body), 40)
	switch analysis.Tone {
	case "supportive":
		return []string{
			fmt.Sprintf("Love this — especially the part about “%s”. Thanks for sharing.", snippet),
			"This is encouraging. What helped you get here?",
			"Really well said. Bookmarking this for the team.",
		}
	case "empathetic":
		return []string{
			"Sorry you’re dealing with this — that sounds frustrating.",
			"Thanks for being honest about it. What would a better outcome look like?",
			"Hearing you. If useful, happy to compare notes on what worked for us.",
		}
	default:
		return []string{
			fmt.Sprintf("Interesting take on “%s”. What made you land there?", snippet),
			"Good point. Have you seen this play out differently in other cases?",
			"Appreciate you posting this — curious what you’d try next.",
		}
	}
}

func analyzeWithOpenAI(apiKey, platform, body string) (Analysis, []string, error) {
	payload := map[string]any{
		"model": firstNonEmpty(os.Getenv("OPENAI_MODEL"), "gpt-4o-mini"),
		"messages": []map[string]string{
			{
				"role": "system",
				"content": "You analyze social posts for operators. Return compact JSON only with keys: summary, sentiment, tone, topics (array), risks (array), hooks (array), language, suggestedReplies (array of 3 short natural replies). No markdown.",
			},
			{
				"role": "user",
				"content": fmt.Sprintf("Platform: %s\nPost:\n%s", firstNonEmpty(platform, "facebook"), body),
			},
		},
		"temperature": 0.4,
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return Analysis{}, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Analysis{}, nil, err
	}
	defer res.Body.Close()
	bodyBytes, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return Analysis{}, nil, fmt.Errorf("openai status %d: %s", res.StatusCode, truncateRunes(string(bodyBytes), 200))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return Analysis{}, nil, err
	}
	if len(parsed.Choices) == 0 {
		return Analysis{}, nil, fmt.Errorf("empty openai response")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	var out struct {
		Summary          string   `json:"summary"`
		Sentiment        string   `json:"sentiment"`
		Tone             string   `json:"tone"`
		Topics           []string `json:"topics"`
		Risks            []string `json:"risks"`
		Hooks            []string `json:"hooks"`
		Language         string   `json:"language"`
		SuggestedReplies []string `json:"suggestedReplies"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &out); err != nil {
		return Analysis{}, nil, err
	}
	analysis := Analysis{
		Summary:   out.Summary,
		Sentiment: firstNonEmpty(out.Sentiment, "neutral"),
		Tone:      firstNonEmpty(out.Tone, "curious"),
		Topics:    out.Topics,
		Risks:     out.Risks,
		Hooks:     out.Hooks,
		Language:  firstNonEmpty(out.Language, "unknown"),
		Engine:    "openai",
	}
	replies := out.SuggestedReplies
	if len(replies) == 0 {
		replies = suggestReplies(analysis, body)
	}
	return analysis, replies, nil
}

func countHits(haystack string, needles []string) int {
	n := 0
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			n++
		}
	}
	return n
}

func detectLanguageHint(text string) string {
	if text == "" {
		return "unknown"
	}
	nonASCII := 0
	for _, r := range text {
		if r > unicode.MaxASCII {
			nonASCII++
		}
	}
	if nonASCII > len([]rune(text))/4 {
		return "non-english"
	}
	return "english-like"
}

func truncateRunes(s string, max int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "…"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func appendUnique(list []string, item string) []string {
	item = strings.TrimSpace(item)
	if item == "" {
		return list
	}
	for _, existing := range list {
		if existing == item {
			return list
		}
	}
	return append(list, item)
}