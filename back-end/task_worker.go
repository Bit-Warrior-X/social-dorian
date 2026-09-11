package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	creditCostPerAccount = 1
	workerScriptTimeout  = 12 * time.Minute
	browseDurationSec    = 90
)

type workerAccount struct {
	Account
	ProxyHost     string
	ProxyPort     int
	ProxyProtocol string
	ProxyUser     string
	ProxyPass     string
}

type workerResult struct {
	OK      bool
	Message string
}

func findScriptDir() string {
	if v := strings.TrimSpace(os.Getenv("SCRIPT_DIR")); v != "" {
		return v
	}
	candidates := []string{
		"/home/social/script",
		filepath.Join("..", "script"),
		"script",
	}
	for _, dir := range candidates {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			abs, err := filepath.Abs(dir)
			if err == nil {
				return abs
			}
			return dir
		}
	}
	return "/home/social/script"
}

func (s *TaskStore) loadWorkerAccount(accountID int) (workerAccount, error) {
	var (
		out                                                        workerAccount
		proxyID                                                    sql.NullInt64
		proxyHost, proxyProto, proxyUser, proxyPass                sql.NullString
		proxyPort                                                  sql.NullInt64
	)
	err := s.db.QueryRow(`
		SELECT a.id, a.platform, a.first_name, a.last_name, a.password, a.email, a.email_password,
		       a.proxy_mode, a.proxy_id,
		       p.host, p.port, p.protocol, p.username, p.password
		FROM accounts a
		LEFT JOIN proxies p ON p.id = a.proxy_id
		WHERE a.id = ?`, accountID).Scan(
		&out.ID, &out.Platform, &out.FirstName, &out.LastName, &out.Password, &out.Email, &out.EmailPassword,
		&out.ProxyMode, &proxyID,
		&proxyHost, &proxyPort, &proxyProto, &proxyUser, &proxyPass,
	)
	if err != nil {
		return workerAccount{}, err
	}
	if proxyID.Valid {
		out.ProxyID = int(proxyID.Int64)
	}
	if proxyHost.Valid {
		out.ProxyHost = proxyHost.String
	}
	if proxyPort.Valid {
		out.ProxyPort = int(proxyPort.Int64)
	}
	if proxyProto.Valid {
		out.ProxyProtocol = proxyProto.String
	}
	if proxyUser.Valid {
		out.ProxyUser = proxyUser.String
	}
	if proxyPass.Valid {
		out.ProxyPass = proxyPass.String
	}
	return out, nil
}

func (a workerAccount) proxyString(useProxy bool) string {
	if !useProxy || a.ProxyMode == "none" || a.ProxyID < 1 || a.ProxyHost == "" || a.ProxyPort < 1 {
		return ""
	}
	proto := strings.ToLower(strings.TrimSpace(a.ProxyProtocol))
	if proto == "" {
		proto = "http"
	}
	auth := ""
	if a.ProxyUser != "" {
		auth = a.ProxyUser
		if a.ProxyPass != "" {
			auth += ":" + a.ProxyPass
		}
		auth += "@"
	}
	return fmt.Sprintf("%s://%s%s:%d", proto, auth, a.ProxyHost, a.ProxyPort)
}

func writeAccountsCSV(path string, account workerAccount, proxy string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"id", "name", "password", "gmail", "gmail_password", "proxy"})
	name := strings.TrimSpace(account.FirstName + " " + account.LastName)
	if name == "" {
		name = account.Email
	}
	_ = w.Write([]string{
		fmt.Sprintf("%d", account.ID),
		name,
		account.Password,
		account.Email,
		account.EmailPassword,
		proxy,
	})
	w.Flush()
	return w.Error()
}

func writePostFile(path string, content TaskContent) error {
	var b strings.Builder
	if strings.TrimSpace(content.Headline) != "" {
		b.WriteString("Title: ")
		b.WriteString(strings.TrimSpace(content.Headline))
		b.WriteString("\n\n")
	}
	b.WriteString(strings.TrimSpace(content.Text))
	if link := strings.TrimSpace(content.LinkURL); link != "" {
		b.WriteString("\n\n")
		b.WriteString(link)
	}
	// Local uploads are passed via --media; only append remote media URLs into the body.
	if media := strings.TrimSpace(content.MediaURL); media != "" && localMediaPath(media) == "" {
		b.WriteString("\n\n")
		b.WriteString(media)
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

func (s *TaskStore) executeItem(task Task, item TaskItem) workerResult {
	account, err := s.loadWorkerAccount(item.AccountID)
	if err != nil {
		return workerResult{OK: false, Message: fmt.Sprintf("account load failed: %v", err)}
	}
	if !strings.EqualFold(account.Platform, "facebook") {
		return workerResult{
			OK:      false,
			Message: fmt.Sprintf("%s automation is Facebook-only for now (got %s)", task.Type, account.Platform),
		}
	}
	if strings.TrimSpace(account.Email) == "" || strings.TrimSpace(account.Password) == "" {
		return workerResult{OK: false, Message: "account email/password missing"}
	}

	useProxy := task.UseAccountProxy
	proxy := account.proxyString(useProxy)
	if useProxy && proxy == "" {
		return workerResult{OK: false, Message: "useAccountProxy is on but account has no usable proxy"}
	}

	workDir, err := os.MkdirTemp("", fmt.Sprintf("dorian-task-%d-%d-*", task.ID, item.AccountID))
	if err != nil {
		return workerResult{OK: false, Message: fmt.Sprintf("temp dir: %v", err)}
	}
	defer os.RemoveAll(workDir)

	accountsFile := filepath.Join(workDir, "accounts.csv")
	if err := writeAccountsCSV(accountsFile, account, proxy); err != nil {
		return workerResult{OK: false, Message: fmt.Sprintf("accounts.csv: %v", err)}
	}

	profileDir := accountProfileDir(account.ID)
	_ = os.MkdirAll(profileDir, 0o755)

	var (
		watchSession *browseSession
		watchURL     string
	)
	if task.ShowBrowser {
		watchSession, watchURL, err = startTaskWatchSession(account.Account)
		if err != nil {
			return workerResult{OK: false, Message: fmt.Sprintf("could not open live browser view: %v", err)}
		}
		defer removeBrowseSession(watchSession.Token)
		s.addLog(task.ID, item.AccountID, "info", "Watch live: "+watchURL)
	}

	scriptDir := findScriptDir()
	args, err := buildWorkerArgs(task, account, workDir, profileDir, useProxy)
	if err != nil {
		return workerResult{OK: false, Message: err.Error()}
	}

	cmd := exec.Command("python3", args...)
	cmd.Dir = scriptDir
	cmd.Env = append(os.Environ(), "DORIAN_ACCOUNTS_FILE="+accountsFile)
	if watchSession != nil && watchSession.Display > 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("DISPLAY=:%d", watchSession.Display))
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return workerResult{OK: false, Message: fmt.Sprintf("stdout pipe: %v", err)}
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return workerResult{OK: false, Message: fmt.Sprintf("stderr pipe: %v", err)}
	}

	var outputMu sync.Mutex
	var outputBuf bytes.Buffer
	streamDone := make(chan struct{})
	go func() {
		defer close(streamDone)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.streamWorkerLines(task.ID, item.AccountID, stdoutPipe, "info", &outputMu, &outputBuf)
		}()
		go func() {
			defer wg.Done()
			s.streamWorkerLines(task.ID, item.AccountID, stderrPipe, "warn", &outputMu, &outputBuf)
		}()
		wg.Wait()
	}()

	if err := cmd.Start(); err != nil {
		return workerResult{OK: false, Message: fmt.Sprintf("start worker: %v", err)}
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var runErr error
	select {
	case runErr = <-done:
	case <-time.After(workerScriptTimeout):
		if cmd.Process != nil {
			// Kill the whole process group (python + chromedriver + chrome).
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		runErr = fmt.Errorf("timed out after %s", workerScriptTimeout)
		<-done
	}
	<-streamDone

	outputMu.Lock()
	outTail := trimWorkerOutput(outputBuf.String())
	outputMu.Unlock()
	if runErr != nil {
		msg := fmt.Sprintf("%s failed for %s: %v", task.Type, item.AccountName, runErr)
		if outTail != "" {
			msg += " — " + outTail
		}
		log.Printf("task %d item %d: %s", task.ID, item.ID, msg)
		return workerResult{OK: false, Message: msg}
	}

	msg := successMessage(task, item.AccountName)
	if watchURL != "" {
		msg += " · watched live"
	}
	if outTail != "" {
		msg += " — " + firstLine(outTail)
	}
	return workerResult{OK: true, Message: msg}
}

func (s *TaskStore) streamWorkerLines(taskID, accountID int, r io.Reader, defaultLevel string, mu *sync.Mutex, buf *bytes.Buffer) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		traceBuf   []string
		inTrace    bool
		traceLimit = 40
	)
	flushTrace := func() {
		if !inTrace || len(traceBuf) == 0 {
			return
		}
		joined := strings.Join(traceBuf, "\n")
		if len(joined) > 3500 {
			joined = joined[:3500] + "…"
		}
		s.addLog(taskID, accountID, "error", joined)
		traceBuf = nil
		inTrace = false
	}

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		mu.Lock()
		buf.WriteString(trimmed)
		buf.WriteByte('\n')
		mu.Unlock()

		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(trimmed, "Traceback ") || strings.HasPrefix(trimmed, "Traceback (") {
			flushTrace()
			inTrace = true
			traceBuf = []string{trimmed}
			continue
		}
		if inTrace {
			traceBuf = append(traceBuf, trimmed)
			isFrame := strings.HasPrefix(trimmed, "File ") || strings.HasPrefix(strings.TrimLeft(trimmed, " \t"), "File ")
			isException := strings.Contains(trimmed, "Error:") || strings.Contains(trimmed, "Exception:") ||
				strings.HasSuffix(trimmed, "Error") || strings.Contains(lower, "exception:")
			if (!isFrame && isException) || len(traceBuf) >= traceLimit {
				flushTrace()
			}
			continue
		}

		level, message := classifyWorkerLogLine(trimmed, defaultLevel)
		if message == "" {
			continue
		}
		s.addLog(taskID, accountID, level, message)
	}
	flushTrace()
}

func classifyWorkerLogLine(line, defaultLevel string) (level, message string) {
	msg := line
	level = defaultLevel

	// Typical: "2026-09-11 02:24:13,822 - INFO - Comment sent"
	if idx := strings.Index(line, " - "); idx >= 0 {
		rest := line[idx+3:]
		if j := strings.Index(rest, " - "); j >= 0 {
			lvl := strings.ToUpper(strings.TrimSpace(rest[:j]))
			body := strings.TrimSpace(rest[j+3:])
			if body != "" {
				msg = body
			}
			switch lvl {
			case "ERROR", "CRITICAL", "FATAL":
				level = "error"
			case "WARNING", "WARN":
				level = "warn"
			case "INFO", "DEBUG":
				level = "info"
			}
		}
	}

	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "traceback"),
		strings.Contains(lower, "exception"),
		strings.HasPrefix(lower, "error"):
		level = "error"
	case strings.Contains(lower, "warning"):
		if level != "error" {
			level = "warn"
		}
	case strings.Contains(lower, "success"),
		strings.Contains(lower, "confirmed"),
		strings.Contains(lower, "posted"),
		strings.Contains(lower, "comment sent"):
		if level == "info" {
			level = "success"
		}
	}

	if len(msg) > 2000 {
		msg = msg[:2000] + "…"
	}
	return level, msg
}

func buildWorkerArgs(task Task, account workerAccount, workDir, profileDir string, useProxy bool) ([]string, error) {
	scriptDir := findScriptDir()
	common := []string{
		"--email", account.Email,
		"--no-keep-open",
		"--profile-dir", profileDir,
	}
	if !task.ShowBrowser {
		common = append(common, "--headless")
	}
	if !useProxy {
		common = append(common, "--no-proxy")
	}

	switch task.Type {
	case "report":
		if strings.TrimSpace(task.TargetURL) == "" {
			return nil, fmt.Errorf("target URL is required for report")
		}
		args := []string{filepath.Join(scriptDir, "fb_report.py"), "--no-config", "--url", task.TargetURL, "--reason", "Spam"}
		return append(args, common...), nil
	case "reply":
		if strings.TrimSpace(task.TargetURL) == "" {
			return nil, fmt.Errorf("target URL is required for reply")
		}
		if strings.TrimSpace(task.Content.Text) == "" {
			return nil, fmt.Errorf("reply text is required")
		}
		replyFile := filepath.Join(workDir, "reply.txt")
		if err := os.WriteFile(replyFile, []byte(strings.TrimSpace(task.Content.Text)), 0o600); err != nil {
			return nil, fmt.Errorf("reply file: %w", err)
		}
		args := []string{
			filepath.Join(scriptDir, "fb_reply.py"),
			"--url", task.TargetURL,
			"--message-file", replyFile,
		}
		return append(args, common...), nil
	case "post":
		postFile := filepath.Join(workDir, "post.txt")
		if err := writePostFile(postFile, task.Content); err != nil {
			return nil, fmt.Errorf("post file: %w", err)
		}
		if strings.TrimSpace(task.Content.Text) == "" && strings.TrimSpace(task.Content.Headline) == "" {
			return nil, fmt.Errorf("post content is empty")
		}
		args := []string{filepath.Join(scriptDir, "fb_post.py"), "--message-file", postFile}
		if mediaPath := localMediaPath(task.Content.MediaURL); mediaPath != "" {
			args = append(args, "--media", mediaPath)
		}
		return append(args, common...), nil
	case "browse":
		args := []string{
			filepath.Join(scriptDir, "fb_browse.py"),
			"--duration", fmt.Sprintf("%d", browseDurationSec),
			"--max-likes", "2",
		}
		return append(args, common...), nil
	case "login_test":
		args := []string{
			filepath.Join(scriptDir, "fb_login.py"),
			"--no-profile-info",
		}
		return append(args, common...), nil
	default:
		return nil, fmt.Errorf("unsupported task type %q", task.Type)
	}
}

func successMessage(task Task, accountName string) string {
	switch task.Type {
	case "report":
		return fmt.Sprintf("%s reported %s", accountName, task.TargetURL)
	case "reply":
		preview := task.Content.Text
		if len(preview) > 48 {
			preview = preview[:45] + "..."
		}
		if preview == "" {
			return fmt.Sprintf("%s replied on post", accountName)
		}
		return fmt.Sprintf("%s replied “%s”", accountName, preview)
	case "post":
		preview := task.Content.Headline
		if preview == "" {
			preview = task.Content.Text
		}
		if len(preview) > 48 {
			preview = preview[:45] + "..."
		}
		if preview == "" {
			return fmt.Sprintf("%s published post", accountName)
		}
		return fmt.Sprintf("%s published “%s”", accountName, preview)
	case "browse":
		return fmt.Sprintf("%s browsed feed (~%ds)", accountName, browseDurationSec)
	case "login_test":
		return fmt.Sprintf("%s login check succeeded", accountName)
	default:
		return fmt.Sprintf("%s finished", accountName)
	}
}

func trimWorkerOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	kept := make([]string, 0, 6)
	for i := len(lines) - 1; i >= 0 && len(kept) < 6; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		kept = append([]string{line}, kept...)
	}
	out := strings.Join(kept, " | ")
	if len(out) > 400 {
		return out[len(out)-400:]
	}
	return out
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '|'); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}

func (s *TaskStore) creditsNeeded(accountCount int) int {
	if accountCount < 1 {
		return 0
	}
	return accountCount * creditCostPerAccount
}

func (s *TaskStore) getCreditBalance() (int, error) {
	var balance int
	err := s.db.QueryRow(`SELECT balance FROM workspace_credits WHERE id = 1`).Scan(&balance)
	return balance, err
}

func (s *TaskStore) debitCredits(amount int, reason string) error {
	if amount <= 0 {
		return nil
	}
	res, err := s.db.Exec(`
		UPDATE workspace_credits
		SET balance = balance - ?
		WHERE id = 1 AND balance >= ?`, amount, amount)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("insufficient credits")
	}
	log.Printf("credits: debited %d (%s)", amount, reason)
	return nil
}
