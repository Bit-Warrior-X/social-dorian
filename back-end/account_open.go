package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	exitIPURL         = "https://api.ipify.org"
	sessionTTL        = 2 * time.Hour
	sessionPathPrefix = "/api/sessions/"
	novncRoot         = "/usr/share/novnc"
	browserProfiles   = "/var/lib/dorian-browser/profiles"
	displayBase       = 40
	maxDisplays       = 40
)

var platformURLs = map[string]string{
	"twitter":   "https://x.com",
	"instagram": "https://www.instagram.com",
	"linkedin":  "https://www.linkedin.com",
	"facebook":  "https://www.facebook.com",
	"tiktok":    "https://www.tiktok.com",
	"youtube":   "https://www.youtube.com",
}

type AccountOpenResult struct {
	SessionURL   string            `json:"sessionUrl"`
	URL          string            `json:"url"`
	ExitIP       string            `json:"exitIp"`
	Proxy        AccountOpenProxy  `json:"proxy"`
	Platform     string            `json:"platform"`
	AccountName  string            `json:"accountName"`
	Email        string            `json:"email"`
	Password     string            `json:"password"`
	Mode         string            `json:"mode"`
	ProfileDir   string            `json:"profileDir"`
	Reused       bool              `json:"reused"`
	LoginStatus  string            `json:"loginStatus"`
	LoginMessage string            `json:"loginMessage"`
	Logs         []SessionLogEntry `json:"logs"`
}

type AccountOpenProxy struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Country  string `json:"country"`
}

type SessionLogEntry struct {
	ID      int    `json:"id"`
	Level   string `json:"level"`
	Message string `json:"message"`
	At      string `json:"at"`
}

type browseSession struct {
	Token        string
	Account      Account
	Proxy        Proxy
	StartURL     string
	ExitIP       string
	CreatedAt    time.Time
	Display      int
	ProfileDir   string
	Forwarder    *http.Server
	ForwardLn    net.Listener
	Cmds         []*exec.Cmd
	WSPort       int
	VNCPort      int
	DebugPort    int
	LoginStatus  string
	LoginMessage string
	Logs         []SessionLogEntry
	nextLogID    int
	loginMu      sync.Mutex
}

var (
	browseSessions   = map[string]*browseSession{}
	browseSessionsMu sync.Mutex
	displayMu        sync.Mutex
)

func (s *AccountStore) open(id int) (AccountOpenResult, error) {
	account, found, err := s.get(id)
	if err != nil {
		return AccountOpenResult{}, err
	}
	if !found {
		return AccountOpenResult{}, fmt.Errorf("account not found")
	}

	startURL, ok := platformURLs[account.Platform]
	if !ok || startURL == "" {
		return AccountOpenResult{}, fmt.Errorf("no page to open for this platform")
	}

	if account.ProxyMode == "none" || account.ProxyID < 1 {
		return AccountOpenResult{}, fmt.Errorf("assign a proxy to this account before opening it")
	}
	if s.proxies == nil {
		return AccountOpenResult{}, fmt.Errorf("proxy store is not available")
	}

	if existing := findBrowseSessionByAccount(account.ID); existing != nil {
		existing.appendLog("info", "Reattached to existing remote session")
		s.ensureLogin(existing)
		return openResultFromSession(existing, true), nil
	}

	proxy, found, err := s.proxies.get(account.ProxyID)
	if err != nil {
		return AccountOpenResult{}, err
	}
	if !found {
		return AccountOpenResult{}, fmt.Errorf("assigned proxy was not found")
	}

	if err := probeProxy(proxy); err != nil {
		emitActivity(ActivityInput{
			Source:    "account",
			Level:     "error",
			Message:   fmt.Sprintf("Open blocked — proxy %s unreachable: %v", proxy.Name, err),
			AccountID: account.ID,
			ProxyID:   proxy.ID,
		})
		return AccountOpenResult{}, fmt.Errorf("proxy %s is not reachable: %w", proxy.Name, err)
	}

	exitIP, err := lookupExitIP(proxy)
	if err != nil {
		log.Printf("open account %d: exit IP lookup failed: %v", account.ID, err)
		exitIP = proxy.Host
	}

	token, err := newSessionToken()
	if err != nil {
		return AccountOpenResult{}, err
	}

	session := &browseSession{
		Token:     token,
		Account:   account,
		Proxy:     proxy,
		StartURL:  startURL,
		ExitIP:    exitIP,
		CreatedAt: time.Now().UTC(),
	}
	session.appendLog("info", fmt.Sprintf("Opening %s via proxy %s", account.Platform, proxy.Name))
	if exitIP != "" {
		session.appendLog("info", "Exit IP: "+exitIP)
	}

	if err := startRemoteChromium(session); err != nil {
		session.appendLog("error", err.Error())
		stopBrowseSession(session)
		return AccountOpenResult{}, err
	}
	session.appendLog("success", "Remote Chromium is ready")

	putBrowseSession(session)
	s.ensureLogin(session)
	return openResultFromSession(session, false), nil
}

func openResultFromSession(session *browseSession, reused bool) AccountOpenResult {
	wsPath := "api/sessions/" + session.Token + "/websockify"
	viewer := sessionPathPrefix + session.Token + "/vnc/vnc_lite.html" +
		"?autoconnect=1&reconnect=1&reconnect_delay=2000&scale=true&path=" + url.QueryEscape(wsPath)

	loginStatus, loginMessage, logs := session.loginSnapshot()
	return AccountOpenResult{
		SessionURL:   viewer,
		URL:          session.StartURL,
		ExitIP:       session.ExitIP,
		Platform:     session.Account.Platform,
		AccountName:  displayName(session.Account.FirstName, session.Account.LastName, session.Account.Email),
		Email:        session.Account.Email,
		Password:     session.Account.Password,
		Mode:         "chromium",
		ProfileDir:   session.ProfileDir,
		Reused:       reused,
		LoginStatus:  loginStatus,
		LoginMessage: loginMessage,
		Logs:         logs,
		Proxy: AccountOpenProxy{
			ID:       session.Proxy.ID,
			Name:     session.Proxy.Name,
			Protocol: session.Proxy.Protocol,
			Host:     session.Proxy.Host,
			Port:     session.Proxy.Port,
			Country:  session.Proxy.Country,
		},
	}
}

func lookupExitIP(p Proxy) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	client, err := proxyHTTPClient(p, 12*time.Second)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exitIPURL, nil)
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 400 {
		return "", fmt.Errorf("status %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 64))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", fmt.Errorf("empty exit IP")
	}
	return ip, nil
}

func startRemoteChromium(session *browseSession) error {
	if _, err := exec.LookPath("chromium"); err != nil {
		return fmt.Errorf("chromium is not installed on the server")
	}
	if _, err := exec.LookPath("Xvfb"); err != nil {
		return fmt.Errorf("Xvfb is not installed on the server")
	}
	if _, err := exec.LookPath("x11vnc"); err != nil {
		return fmt.Errorf("x11vnc is not installed on the server")
	}
	if _, err := exec.LookPath("websockify"); err != nil {
		return fmt.Errorf("websockify is not installed on the server")
	}
	if _, err := os.Stat(novncRoot); err != nil {
		return fmt.Errorf("noVNC assets not found at %s", novncRoot)
	}

	forwardLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	session.ForwardLn = forwardLn
	session.Forwarder = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleForwardedProxy(w, r, session.Proxy)
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := session.Forwarder.Serve(forwardLn); err != nil && err != http.ErrServerClosed {
			log.Printf("session %s forwarder: %v", session.Token, err)
		}
	}()

	display, err := allocateDisplay()
	if err != nil {
		return err
	}
	session.Display = display

	browserUser, err := lookupBrowserUser()
	if err != nil {
		return err
	}
	if err := ensureBrowserProfilesRoot(browserUser); err != nil {
		return fmt.Errorf("prepare browser profiles root: %w", err)
	}

	profileDir := accountProfileDir(session.Account.ID)
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return err
	}
	session.ProfileDir = profileDir

	clearChromiumSingletonLocks(profileDir)
	if err := chownPath(profileDir, browserUser); err != nil {
		return fmt.Errorf("prepare browser profile: %w", err)
	}

	session.appendLog("info", "Starting virtual display…")
	xvfb := exec.Command("Xvfb", fmt.Sprintf(":%d", display), "-screen", "0", "1280x800x24", "-ac", "-nolisten", "tcp")
	xvfb.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := xvfb.Start(); err != nil {
		return fmt.Errorf("start Xvfb: %w", err)
	}
	session.Cmds = append(session.Cmds, xvfb)
	time.Sleep(400 * time.Millisecond)

	vncLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	vncPort := vncLn.Addr().(*net.TCPAddr).Port
	_ = vncLn.Close()
	session.VNCPort = vncPort

	wsLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	wsPort := wsLn.Addr().(*net.TCPAddr).Port
	_ = wsLn.Close()
	session.WSPort = wsPort

	debugLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	debugPort := debugLn.Addr().(*net.TCPAddr).Port
	_ = debugLn.Close()
	session.DebugPort = debugPort

	displayEnv := fmt.Sprintf("DISPLAY=:%d", display)
	chromium := exec.Command(
		"chromium",
		"--disable-dev-shm-usage",
		"--disable-gpu",
		"--disable-software-rasterizer",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-sync",
		"--disable-features=TranslateUI",
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port="+strconv.Itoa(debugPort),
		"--remote-allow-origins=*",
		"--window-size=1280,800",
		"--window-position=0,0",
		"--user-data-dir="+profileDir,
		"--proxy-server=http://"+forwardLn.Addr().String(),
		session.StartURL,
	)
	chromium.Env = append(os.Environ(), displayEnv, "HOME="+browserUser.HomeDir)
	chromium.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Credential: &syscall.Credential{
			Uid: browserUser.Uid,
			Gid: browserUser.Gid,
		},
	}
	session.appendLog("info", "Launching Chromium with saved profile…")
	if err := chromium.Start(); err != nil {
		return fmt.Errorf("start chromium: %w", err)
	}
	session.Cmds = append(session.Cmds, chromium)

	session.appendLog("info", "Starting live view (VNC)…")
	x11vnc := exec.Command(
		"x11vnc",
		"-display", fmt.Sprintf(":%d", display),
		"-rfbport", strconv.Itoa(vncPort),
		"-localhost",
		"-forever",
		"-shared",
		"-nopw",
		"-xkb",
		"-ncache", "0",
		"-wait", "10",
		"-deferupdate", "10",
		"-quiet",
	)
	x11vnc.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := x11vnc.Start(); err != nil {
		return fmt.Errorf("start x11vnc: %w", err)
	}
	session.Cmds = append(session.Cmds, x11vnc)
	time.Sleep(300 * time.Millisecond)

	websockify := exec.Command(
		"websockify",
		fmt.Sprintf("127.0.0.1:%d", wsPort),
		fmt.Sprintf("127.0.0.1:%d", vncPort),
	)
	websockify.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := websockify.Start(); err != nil {
		return fmt.Errorf("start websockify: %w", err)
	}
	session.Cmds = append(session.Cmds, websockify)

	return nil
}

func (s *AccountStore) handleBrowseSessions(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, sessionPathPrefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	parts := strings.SplitN(rest, "/", 2)
	token := parts[0]
	session := getBrowseSession(token)
	if session == nil {
		writeError(w, http.StatusNotFound, "session expired or not found")
		return
	}

	if len(parts) == 1 {
		if r.Method == http.MethodDelete {
			removeBrowseSession(token)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, sessionPathPrefix+token+"/vnc/vnc_lite.html", http.StatusFound)
		return
	}

	action := parts[1]
	switch {
	case action == "status":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, openResultFromSession(session, true))
	case action == "login":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		session.loginMu.Lock()
		if session.LoginStatus == loginStatusOK {
			session.loginMu.Unlock()
			writeJSON(w, http.StatusOK, openResultFromSession(session, true))
			return
		}
		if session.LoginStatus == loginStatusStart || session.LoginStatus == loginStatusSignIn || session.LoginStatus == loginStatusVerify {
			session.loginMu.Unlock()
			writeJSON(w, http.StatusOK, openResultFromSession(session, true))
			return
		}
		session.LoginStatus = ""
		session.loginMu.Unlock()
		s.ensureLogin(session)
		writeJSON(w, http.StatusOK, openResultFromSession(session, true))
	case action == "websockify" || strings.HasPrefix(action, "websockify/"):
		proxySessionWebsockify(w, r, session)
	case action == "vnc" || strings.HasPrefix(action, "vnc/"):
		serveSessionNoVNC(w, r, strings.TrimPrefix(action, "vnc"))
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func proxySessionWebsockify(w http.ResponseWriter, r *http.Request, session *browseSession) {
	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", session.WSPort))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid websocket target")
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = "/"
		req.URL.RawPath = "/"
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, err error) {
		log.Printf("session %s websockify proxy: %v", session.Token, err)
		http.Error(rw, "remote browser websocket unavailable", http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}

func serveSessionNoVNC(w http.ResponseWriter, r *http.Request, suffix string) {
	rel := strings.TrimPrefix(suffix, "/")
	if rel == "" {
		http.Redirect(w, r, strings.TrimSuffix(r.URL.Path, "/")+"/vnc_lite.html", http.StatusFound)
		return
	}

	root := http.Dir(novncRoot)
	fileServer := http.FileServer(root)
	r2 := r.Clone(r.Context())
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = "/" + rel
	r2.URL.RawPath = ""
	fileServer.ServeHTTP(w, r2)
}

func allocateDisplay() (int, error) {
	displayMu.Lock()
	defer displayMu.Unlock()

	for display := displayBase; display < displayBase+maxDisplays; display++ {
		lockPath := fmt.Sprintf("/tmp/.X%d-lock", display)
		if _, err := os.Stat(lockPath); err == nil {
			continue
		}
		socketPath := fmt.Sprintf("/tmp/.X11-unix/X%d", display)
		if _, err := os.Stat(socketPath); err == nil {
			continue
		}
		return display, nil
	}
	return 0, fmt.Errorf("no free X display available")
}

func putBrowseSession(session *browseSession) {
	browseSessionsMu.Lock()
	defer browseSessionsMu.Unlock()
	expireBrowseSessionsLocked()
	browseSessions[session.Token] = session
}

func getBrowseSession(token string) *browseSession {
	browseSessionsMu.Lock()
	defer browseSessionsMu.Unlock()
	expireBrowseSessionsLocked()
	return browseSessions[token]
}

func removeBrowseSession(token string) {
	browseSessionsMu.Lock()
	session := browseSessions[token]
	delete(browseSessions, token)
	browseSessionsMu.Unlock()
	if session != nil {
		stopBrowseSession(session)
	}
}

func expireBrowseSessionsLocked() {
	now := time.Now().UTC()
	for token, session := range browseSessions {
		if now.Sub(session.CreatedAt) > sessionTTL {
			delete(browseSessions, token)
			go stopBrowseSession(session)
		}
	}
}

func stopBrowseSession(session *browseSession) {
	if session == nil {
		return
	}
	for i := len(session.Cmds) - 1; i >= 0; i-- {
		cmd := session.Cmds[i]
		if cmd.Process == nil {
			continue
		}
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGTERM)
		} else {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
	}
	time.Sleep(800 * time.Millisecond)
	for i := len(session.Cmds) - 1; i >= 0; i-- {
		cmd := session.Cmds[i]
		if cmd.Process == nil {
			continue
		}
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
		_, _ = cmd.Process.Wait()
	}
	if session.Forwarder != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = session.Forwarder.Shutdown(ctx)
		cancel()
	}
	if session.ForwardLn != nil {
		_ = session.ForwardLn.Close()
	}
	// Keep ProfileDir on disk so cookies/local storage survive for the next open.
}

func accountProfileDir(accountID int) string {
	return filepath.Join(browserProfiles, fmt.Sprintf("account-%d", accountID))
}

func ensureBrowserProfilesRoot(u *browserUser) error {
	if err := os.MkdirAll(browserProfiles, 0o755); err != nil {
		return err
	}
	return os.Chown(browserProfiles, int(u.Uid), int(u.Gid))
}

func clearChromiumSingletonLocks(profileDir string) {
	for _, name := range []string{"SingletonLock", "SingletonCookie", "SingletonSocket"} {
		_ = os.Remove(filepath.Join(profileDir, name))
	}
}

func deleteAccountBrowserProfile(accountID int) {
	dir := accountProfileDir(accountID)
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("delete browser profile for account %d: %v", accountID, err)
	}
}

func findBrowseSessionByAccount(accountID int) *browseSession {
	browseSessionsMu.Lock()
	defer browseSessionsMu.Unlock()
	expireBrowseSessionsLocked()
	for _, session := range browseSessions {
		if session.Account.ID == accountID {
			return session
		}
	}
	return nil
}

func newSessionToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

type browserUser struct {
	Uid     uint32
	Gid     uint32
	HomeDir string
}

func lookupBrowserUser() (*browserUser, error) {
	u, err := user.Lookup("dorian-browser")
	if err != nil {
		return nil, fmt.Errorf("browser user dorian-browser is missing: %w", err)
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return nil, err
	}
	home := u.HomeDir
	if home == "" {
		home = "/var/lib/dorian-browser"
	}
	return &browserUser{Uid: uint32(uid), Gid: uint32(gid), HomeDir: home}, nil
}

func chownPath(path string, u *browserUser) error {
	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if err := os.Lchown(name, int(u.Uid), int(u.Gid)); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	})
}

func handleForwardedProxy(w http.ResponseWriter, r *http.Request, proxy Proxy) {
	if r.Method == http.MethodConnect {
		dest := r.Host
		if dest == "" {
			http.Error(w, "missing host", http.StatusBadRequest)
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack not supported", http.StatusInternalServerError)
			return
		}
		client, _, err := hj.Hijack()
		if err != nil {
			http.Error(w, "hijack failed", http.StatusInternalServerError)
			return
		}
		upstream, err := dialViaProxy(proxy, dest)
		if err != nil {
			_, _ = client.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			client.Close()
			return
		}
		_, _ = client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		go func() {
			defer client.Close()
			defer upstream.Close()
			_, _ = io.Copy(upstream, client)
		}()
		_, _ = io.Copy(client, upstream)
		client.Close()
		upstream.Close()
		return
	}

	client, err := proxyHTTPClient(proxy, 30*time.Second)
	if err != nil {
		http.Error(w, "proxy setup failed", http.StatusBadGateway)
		return
	}
	out, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	out.Header = r.Header.Clone()
	res, err := client.Do(out)
	if err != nil {
		http.Error(w, "upstream failed", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	for key, values := range res.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
}
