package main

import (
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// startTaskWatchSession prepares an Xvfb + noVNC display (no Chromium) so a
// Selenium worker can show a live window the same way as Account → Open.
func startTaskWatchSession(account Account) (*browseSession, string, error) {
	if existing := findBrowseSessionByAccount(account.ID); existing != nil {
		removeBrowseSession(existing.Token)
	}

	if _, err := exec.LookPath("Xvfb"); err != nil {
		return nil, "", fmt.Errorf("Xvfb is not installed on the server")
	}
	if _, err := exec.LookPath("x11vnc"); err != nil {
		return nil, "", fmt.Errorf("x11vnc is not installed on the server")
	}
	if _, err := exec.LookPath("websockify"); err != nil {
		return nil, "", fmt.Errorf("websockify is not installed on the server")
	}

	token, err := newSessionToken()
	if err != nil {
		return nil, "", err
	}

	display, err := allocateDisplay()
	if err != nil {
		return nil, "", err
	}

	session := &browseSession{
		Token:      token,
		Account:    account,
		CreatedAt:  time.Now().UTC(),
		Display:    display,
		ProfileDir: accountProfileDir(account.ID),
		StartURL:   platformURLs[account.Platform],
	}

	clearChromiumSingletonLocks(session.ProfileDir)

	xvfb := exec.Command("Xvfb", fmt.Sprintf(":%d", display), "-screen", "0", "1280x800x24", "-ac", "-nolisten", "tcp")
	xvfb.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := xvfb.Start(); err != nil {
		return nil, "", fmt.Errorf("start Xvfb: %w", err)
	}
	session.Cmds = append(session.Cmds, xvfb)
	time.Sleep(400 * time.Millisecond)

	vncLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		stopBrowseSession(session)
		return nil, "", err
	}
	vncPort := vncLn.Addr().(*net.TCPAddr).Port
	_ = vncLn.Close()
	session.VNCPort = vncPort

	wsLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		stopBrowseSession(session)
		return nil, "", err
	}
	wsPort := wsLn.Addr().(*net.TCPAddr).Port
	_ = wsLn.Close()
	session.WSPort = wsPort

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
		stopBrowseSession(session)
		return nil, "", fmt.Errorf("start x11vnc: %w", err)
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
		stopBrowseSession(session)
		return nil, "", fmt.Errorf("start websockify: %w", err)
	}
	session.Cmds = append(session.Cmds, websockify)

	putBrowseSession(session)

	wsPath := "api/sessions/" + session.Token + "/websockify"
	viewer := sessionPathPrefix + session.Token + "/vnc/vnc_lite.html" +
		"?autoconnect=1&reconnect=1&reconnect_delay=2000&scale=true&path=" + url.QueryEscape(wsPath)
	return session, viewer, nil
}
