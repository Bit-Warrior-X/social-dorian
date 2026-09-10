package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type cdpClient struct {
	conn    *websocket.Conn
	nextID  atomic.Int64
	pending map[int64]chan cdpResponse
	mu      sync.Mutex
}

type cdpResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *cdpError       `json:"error"`
}

type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cdpTarget struct {
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	Title                string `json:"title"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func waitForCDPPage(port int, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/json", port)
	var lastErr error
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		res, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(300 * time.Millisecond)
			continue
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(300 * time.Millisecond)
			continue
		}
		var targets []cdpTarget
		if err := json.Unmarshal(body, &targets); err != nil {
			lastErr = err
			time.Sleep(300 * time.Millisecond)
			continue
		}
		for _, target := range targets {
			if target.Type == "page" && target.WebSocketDebuggerURL != "" {
				return target.WebSocketDebuggerURL, nil
			}
		}
		lastErr = fmt.Errorf("no page target yet")
		time.Sleep(300 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for DevTools")
	}
	return "", fmt.Errorf("chromium DevTools not ready: %w", lastErr)
}

func dialCDP(wsURL string) (*cdpClient, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 8 * time.Second}
	header := http.Header{}
	header.Set("Origin", "http://127.0.0.1")
	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("connect DevTools: %w", err)
	}
	client := &cdpClient{
		conn:    conn,
		pending: map[int64]chan cdpResponse{},
	}
	go client.readLoop()
	if _, err := client.call("Page.enable", nil); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := client.call("Runtime.enable", nil); err != nil {
		conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *cdpClient) Close() {
	if c == nil || c.conn == nil {
		return
	}
	_ = c.conn.Close()
}

func (c *cdpClient) readLoop() {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.failAll(err)
			return
		}
		var msg cdpResponse
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.ID == 0 {
			continue
		}
		c.mu.Lock()
		ch := c.pending[msg.ID]
		delete(c.pending, msg.ID)
		c.mu.Unlock()
		if ch != nil {
			ch <- msg
		}
	}
}

func (c *cdpClient) failAll(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.pending {
		ch <- cdpResponse{ID: id, Error: &cdpError{Message: err.Error()}}
		delete(c.pending, id)
	}
}

func (c *cdpClient) call(method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	payload := map[string]any{"id": id, "method": method}
	if params != nil {
		payload["params"] = params
	}
	ch := make(chan cdpResponse, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	if err := c.conn.WriteJSON(payload); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	timer := time.NewTimer(25 * time.Second)
	defer timer.Stop()
	select {
	case res := <-ch:
		if res.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, res.Error.Message)
		}
		return res.Result, nil
	case <-timer.C:
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("%s: timed out", method)
	}
}

func (c *cdpClient) navigate(url string) error {
	_, err := c.call("Page.navigate", map[string]any{"url": url})
	return err
}

func (c *cdpClient) evaluate(expression string) (string, error) {
	raw, err := c.call("Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err != nil {
		return "", err
	}
	var out struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.ExceptionDetails) > 0 && string(out.ExceptionDetails) != "null" {
		return "", fmt.Errorf("page script error: %s", string(out.ExceptionDetails))
	}
	if len(out.Result.Value) == 0 || string(out.Result.Value) == "null" {
		return "", nil
	}
	var text string
	if err := json.Unmarshal(out.Result.Value, &text); err == nil {
		return text, nil
	}
	return string(out.Result.Value), nil
}
