package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

const (
	facebookLoginURL   = "https://www.facebook.com/login"
	gmailCodeScript    = "/home/social/script/gmail_fb_code.py"
	loginStatusSkipped = "skipped"
	loginStatusStart   = "starting"
	loginStatusSignIn  = "signing_in"
	loginStatusVerify  = "verifying"
	loginStatusOK      = "success"
	loginStatusCheck   = "checkpoint"
	loginStatusFailed  = "failed"
	loginStatusError   = "error"
)

func (s *browseSession) setLogin(status, message string) {
	s.loginMu.Lock()
	s.LoginStatus = status
	s.LoginMessage = message
	s.loginMu.Unlock()
}

func (s *browseSession) loginSnapshot() (status, message string) {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	return s.LoginStatus, s.LoginMessage
}

func (st *AccountStore) ensureLogin(session *browseSession) {
	if session == nil {
		return
	}
	if session.Account.Platform != "facebook" {
		session.setLogin(loginStatusSkipped, "Auto-login is only set up for Facebook")
		return
	}

	session.loginMu.Lock()
	switch session.LoginStatus {
	case loginStatusStart, loginStatusSignIn, loginStatusVerify, loginStatusOK:
		session.loginMu.Unlock()
		return
	}
	session.LoginStatus = loginStatusStart
	session.LoginMessage = "Connecting to the browser…"
	session.loginMu.Unlock()

	go st.runFacebookLogin(session)
}

func (st *AccountStore) runFacebookLogin(session *browseSession) {
	wsURL, err := waitForCDPPage(session.DebugPort, 20*time.Second)
	if err != nil {
		session.setLogin(loginStatusError, err.Error())
		log.Printf("account %d login: %v", session.Account.ID, err)
		return
	}

	client, err := dialCDP(wsURL)
	if err != nil {
		session.setLogin(loginStatusError, err.Error())
		log.Printf("account %d login: %v", session.Account.ID, err)
		return
	}
	defer client.Close()

	session.setLogin(loginStatusSignIn, "Opening Facebook login…")
	if err := client.navigate(facebookLoginURL); err != nil {
		session.setLogin(loginStatusError, "Could not open Facebook login")
		log.Printf("account %d login navigate: %v", session.Account.ID, err)
		return
	}
	time.Sleep(3 * time.Second)
	dismissCookies(client)
	time.Sleep(1 * time.Second)

	if handled := st.finishVerificationIfNeeded(session, client); handled {
		return
	}

	state := classifyFacebook(client)
	if state == loginStatusOK {
		dismissCookies(client)
		session.setLogin(loginStatusOK, "Already signed in")
		st.markAccountStatus(session.Account.ID, "active")
		return
	}

	email := strings.TrimSpace(session.Account.Email)
	password := session.Account.Password
	if email == "" || password == "" {
		session.setLogin(loginStatusFailed, "Account is missing email or password")
		return
	}

	session.setLogin(loginStatusSignIn, "Filling Facebook login…")
	form := pageFields(client)
	if !strings.Contains(form, "email") {
		if handled := st.finishVerificationIfNeeded(session, client); handled {
			return
		}
		state = classifyFacebook(client)
		if state == loginStatusOK {
			session.setLogin(loginStatusOK, "Signed in")
			st.markAccountStatus(session.Account.ID, "active")
			return
		}
		if state == loginStatusCheck {
			session.setLogin(loginStatusCheck, "Facebook asked for extra verification — finish it in the window")
			return
		}
		session.setLogin(loginStatusError, "Facebook login form was not found")
		return
	}

	if !strings.Contains(form, "pass") {
		_ = fillField(client, "email", email)
		time.Sleep(400 * time.Millisecond)
		clickByTexts(client, "Continue", "Next", "Log in", "Log In")
		time.Sleep(2 * time.Second)
		dismissCookies(client)
	}

	if err := fillField(client, "email", email); err != nil {
		log.Printf("account %d login email: %v", session.Account.ID, err)
	}
	time.Sleep(250 * time.Millisecond)
	if err := fillField(client, "pass", password); err != nil {
		session.setLogin(loginStatusError, "Could not fill the password field")
		return
	}
	time.Sleep(400 * time.Millisecond)
	if !clickLogin(client) {
		_, _ = client.evaluate(`(() => {
			const pass = document.querySelector('#pass, input[name="pass"], input[type="password"]');
			if (!pass) return 'no';
			if (pass.form) { pass.form.submit(); return 'form'; }
			pass.dispatchEvent(new KeyboardEvent('keydown', {key:'Enter', bubbles:true}));
			return 'enter';
		})()`)
	}

	session.setLogin(loginStatusSignIn, "Submitted login, waiting…")
	time.Sleep(5 * time.Second)
	dismissCookies(client)

	if handled := st.finishVerificationIfNeeded(session, client); handled {
		return
	}

	state = classifyFacebook(client)
	dismissCookies(client)
	switch state {
	case loginStatusOK:
		session.setLogin(loginStatusOK, "Signed in")
		st.markAccountStatus(session.Account.ID, "active")
	case loginStatusCheck:
		session.setLogin(loginStatusCheck, "Facebook asked for extra verification — finish it in the window")
	case loginStatusFailed:
		session.setLogin(loginStatusFailed, "Still on the login page — check the email and password")
		st.markAccountStatus(session.Account.ID, "error")
	default:
		session.setLogin(state, "Facebook needs attention in the window")
	}
}

func (st *AccountStore) finishVerificationIfNeeded(session *browseSession, client *cdpClient) bool {
	if !needsEmailConfirmation(client) {
		return false
	}

	secret := st.mailboxSecret(session.Account.Email, session.Account.EmailPassword)
	if secret == "" {
		session.setLogin(loginStatusVerify, "Enter the email confirmation code in the window")
		return true
	}

	session.setLogin(loginStatusVerify, "Waiting for the Facebook email code…")
	code := fetchFacebookEmailCode(session.Account.Email, secret, 45, 0)
	if code == "" {
		session.setLogin(loginStatusVerify, "No code yet — requesting a resend…")
		if !resendConfirmationCode(client) {
			session.setLogin(loginStatusVerify, "Enter the email confirmation code in the window")
			return true
		}
		code = fetchFacebookEmailCode(session.Account.Email, secret, 45, time.Now().Unix()-5)
	}
	if code == "" {
		session.setLogin(loginStatusVerify, "No email code arrived — enter it in the window")
		return true
	}

	session.setLogin(loginStatusVerify, "Submitting email confirmation code…")
	if !submitConfirmationCode(client, code) {
		session.setLogin(loginStatusVerify, "Facebook rejected the code — enter it in the window")
		return true
	}
	time.Sleep(2 * time.Second)
	dismissCookies(client)
	if needsEmailConfirmation(client) {
		session.setLogin(loginStatusFailed, "Still on the confirmation page after submitting the code")
		return true
	}
	state := classifyFacebook(client)
	if state == loginStatusOK {
		session.setLogin(loginStatusOK, "Signed in after email confirmation")
		st.markAccountStatus(session.Account.ID, "active")
		return true
	}
	session.setLogin(state, "Email code submitted")
	return true
}

func (st *AccountStore) mailboxSecret(accountEmail, accountEmailPassword string) string {
	accountEmailPassword = strings.ReplaceAll(strings.TrimSpace(accountEmailPassword), " ", "")
	var appPass, gmailPass sql.NullString
	err := st.db.QueryRow(`
		SELECT g.app_password, g.password
		FROM gmails g
		WHERE LOWER(g.email) = LOWER(?)
		LIMIT 1`, strings.TrimSpace(accountEmail)).Scan(&appPass, &gmailPass)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("mailbox lookup for %s: %v", accountEmail, err)
	}
	candidates := []string{
		strings.ReplaceAll(strings.TrimSpace(appPass.String), " ", ""),
		accountEmailPassword,
		strings.ReplaceAll(strings.TrimSpace(gmailPass.String), " ", ""),
	}
	for _, secret := range candidates {
		if secret == "" {
			continue
		}
		switch strings.ToLower(secret) {
		case "gmailpass", "password", "apppassword":
			continue
		}
		return secret
	}
	return ""
}

func (st *AccountStore) markAccountStatus(id int, status string) {
	if _, err := st.db.Exec(`UPDATE accounts SET status = ? WHERE id = ?`, status, id); err != nil {
		log.Printf("update account %d status: %v", id, err)
	}
}

func dismissCookies(client *cdpClient) {
	for i := 0; i < 3; i++ {
		clicked, err := client.evaluate(`(() => {
			const labels = [
				'Allow all cookies',
				'Allow essential and optional cookies',
				'Accept all',
				'Accept All',
				'Decline optional cookies'
			];
			const nodes = Array.from(document.querySelectorAll('button, [role="button"], div[aria-label], span'));
			for (const label of labels) {
				const node = nodes.find((el) => {
					const text = ((el.innerText || el.getAttribute('aria-label') || '') + '').replace(/\s+/g, ' ').trim();
					return text === label || text.includes(label);
				});
				if (!node) continue;
				const target = node.closest('button, [role="button"]') || node;
				target.click();
				return label;
			}
			return '';
		})()`)
		if err != nil {
			return
		}
		if strings.Trim(clicked, `"`) == "" {
			return
		}
		time.Sleep(1500 * time.Millisecond)
	}
}

func pageFields(client *cdpClient) string {
	raw, err := client.evaluate(`(() => {
		const visible = (el) => !!(el && el.offsetParent !== null);
		const email = document.querySelector('#email, input[name="email"], input[autocomplete="username"], input[type="email"]');
		const pass = document.querySelector('#pass, input[name="pass"], input[type="password"], input[autocomplete="current-password"]');
		const parts = [];
		if (visible(email)) parts.push('email');
		if (visible(pass)) parts.push('pass');
		return parts.join(',');
	})()`)
	if err != nil {
		return ""
	}
	return strings.Trim(raw, `"`)
}

func fillField(client *cdpClient, kind, value string) error {
	selector := `#email, input[name="email"], input[autocomplete="username"], input[type="email"]`
	if kind == "pass" {
		selector = `#pass, input[name="pass"], input[type="password"], input[autocomplete="current-password"]`
	}
	payload, _ := json.Marshal(value)
	script := fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) return 'missing';
		el.focus();
		el.click();
		const proto = el.tagName === 'TEXTAREA' ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
		const desc = Object.getOwnPropertyDescriptor(proto, 'value');
		if (desc && desc.set) desc.set.call(el, %s);
		else el.value = %s;
		el.dispatchEvent(new Event('input', { bubbles: true }));
		el.dispatchEvent(new Event('change', { bubbles: true }));
		return 'ok';
	})()`, selector, payload, payload)
	out, err := client.evaluate(script)
	if err != nil {
		return err
	}
	if strings.Trim(out, `"`) == "missing" {
		return fmt.Errorf("%s field not found", kind)
	}
	return nil
}

func clickLogin(client *cdpClient) bool {
	out, err := client.evaluate(`(() => {
		const nodes = Array.from(document.querySelectorAll('button[name="login"], #loginbutton, button[type="submit"], [role="button"]'));
		for (const node of nodes) {
			const text = ((node.innerText || node.getAttribute('aria-label') || '') + '').trim();
			if (node.getAttribute('name') === 'login' || node.id === 'loginbutton' || /^log in$/i.test(text)) {
				node.click();
				return 'ok';
			}
		}
		return '';
	})()`)
	return err == nil && strings.Trim(out, `"`) == "ok"
}

func clickByTexts(client *cdpClient, labels ...string) bool {
	payload, _ := json.Marshal(labels)
	out, err := client.evaluate(fmt.Sprintf(`(() => {
		const labels = %s;
		const nodes = Array.from(document.querySelectorAll('button, a, [role="button"], [role="menuitem"], [role="option"], [role="dialog"] *'));
		for (const label of labels) {
			const node = nodes.find((el) => {
				const text = ((el.innerText || el.getAttribute('aria-label') || '') + '').replace(/\s+/g, ' ').trim();
				return text === label || text.includes(label);
			});
			if (!node) continue;
			const target = node.closest('button, a, [role="button"], [role="menuitem"]') || node;
			target.click();
			return label;
		}
		return '';
	})()`, payload))
	return err == nil && strings.Trim(out, `"`) != ""
}

func needsEmailConfirmation(client *cdpClient) bool {
	out, err := client.evaluate(`(() => {
		const url = location.href.toLowerCase();
		if (url.includes('confirmemail') || url.includes('confirm_email')) return 'yes';
		const text = (document.body && document.body.innerText || '').toLowerCase();
		if (text.includes('enter the confirmation code') || text.includes('confirmation code') || text.includes('enter the code we sent')) return 'yes';
		return '';
	})()`)
	return err == nil && strings.Trim(out, `"`) == "yes"
}

func classifyFacebook(client *cdpClient) string {
	out, err := client.evaluate(`(() => {
		const url = location.href.toLowerCase();
		const text = (document.body && document.body.innerText || '').toLowerCase();
		if (url.includes('confirmemail') || url.includes('confirm_email') || text.includes('enter the confirmation code')) return 'verifying';
		if (url.includes('checkpoint') || url.includes('auth_platform') || url.includes('two_step') || url.includes('approvals')) return 'checkpoint';
		if (url.includes('login') && (url.includes('login.php') || url.replace(/\/$/, '').endsWith('/login'))) return 'failed';
		if (url.includes('facebook.com') && !url.includes('login') && !url.includes('confirm')) return 'success';
		return 'checkpoint';
	})()`)
	if err != nil {
		return loginStatusError
	}
	switch strings.Trim(out, `"`) {
	case loginStatusOK, loginStatusCheck, loginStatusFailed, loginStatusVerify:
		return strings.Trim(out, `"`)
	default:
		return loginStatusCheck
	}
}

func resendConfirmationCode(client *cdpClient) bool {
	if !clickByTexts(client, "I didn't get the code", "I didn’t get the code", "Didn't get the code") {
		return false
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		out, _ := client.evaluate(`(() => {
			const text = (document.body && document.body.innerText || '');
			if (text.includes("Haven't received the code") || text.includes("Haven’t received the code") || text.includes('Resend confirmation code')) return 'yes';
			return '';
		})()`)
		if strings.Trim(out, `"`) == "yes" {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	return clickByTexts(client, "Resend confirmation code", "Resend code")
}

func submitConfirmationCode(client *cdpClient, code string) bool {
	payload, _ := json.Marshal(code)
	out, err := client.evaluate(fmt.Sprintf(`(() => {
		const nodes = Array.from(document.querySelectorAll('input'));
		const el = nodes.find((input) => {
			if (input.offsetParent === null) return false;
			const hint = ((input.placeholder || '') + ' ' + (input.getAttribute('aria-label') || '') + ' ' + (input.name || '') + ' ' + (input.id || '')).toLowerCase();
			return hint.includes('confirmation') || hint.includes('code') || input.autocomplete === 'one-time-code' || input.name === 'approvals_code';
		}) || nodes.find((input) => input.offsetParent !== null && (input.type === 'text' || input.type === 'tel' || input.type === 'number'));
		if (!el) return 'missing';
		el.focus();
		el.click();
		const proto = HTMLInputElement.prototype;
		const desc = Object.getOwnPropertyDescriptor(proto, 'value');
		if (desc && desc.set) desc.set.call(el, %s);
		else el.value = %s;
		el.dispatchEvent(new Event('input', { bubbles: true }));
		el.dispatchEvent(new Event('change', { bubbles: true }));
		return 'ok';
	})()`, payload, payload))
	if err != nil || strings.Trim(out, `"`) != "ok" {
		return false
	}
	time.Sleep(400 * time.Millisecond)
	if !clickByTexts(client, "Continue", "Confirm", "Submit") {
		_, _ = client.evaluate(`(() => {
			const el = document.querySelector('input[autocomplete="one-time-code"], input[name="code"], input[name="approvals_code"]');
			if (el && el.form) el.form.submit();
			return 'ok';
		})()`)
	}
	time.Sleep(4 * time.Second)
	return !needsEmailConfirmation(client)
}

func fetchFacebookEmailCode(email, password string, timeout int, since int64) string {
	args := []string{
		gmailCodeScript,
		"--email", email,
		"--password", password,
		"--timeout", fmt.Sprintf("%d", timeout),
	}
	if since > 0 {
		args = append(args, "--since", fmt.Sprintf("%d", since))
	}
	cmd := exec.Command("python3", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("gmail code fetch: %v %s", err, strings.TrimSpace(stderr.String()))
		return ""
	}
	var out struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		log.Printf("gmail code parse: %v (%s)", err, strings.TrimSpace(stdout.String()))
		return ""
	}
	return strings.TrimSpace(out.Code)
}
