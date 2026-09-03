package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	xproxy "golang.org/x/net/proxy"
)

const proxyCheckURL = "https://www.gstatic.com/generate_204"

type ProxyCheckResult struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latencyMs"`
	Message   string `json:"message"`
	Proxy     Proxy  `json:"proxy"`
}

func (s *ProxyStore) setStatus(id int, status string) error {
	_, err := s.db.Exec(`UPDATE proxies SET status = ? WHERE id = ?`, status, id)
	return err
}

func (s *ProxyStore) check(id int) (ProxyCheckResult, bool, error) {
	proxy, found, err := s.get(id)
	if err != nil || !found {
		return ProxyCheckResult{}, found, err
	}

	started := time.Now()
	checkErr := probeProxy(proxy)
	latency := time.Since(started).Milliseconds()

	result := ProxyCheckResult{
		LatencyMs: latency,
		Proxy:     proxy,
	}

	if checkErr != nil {
		result.OK = false
		result.Message = checkErr.Error()
		if err := s.setStatus(id, "error"); err != nil {
			return result, true, err
		}
	} else {
		result.OK = true
		result.Message = "Proxy is reachable"
		if err := s.setStatus(id, "active"); err != nil {
			return result, true, err
		}
	}

	updated, _, err := s.get(id)
	if err != nil {
		return result, true, err
	}
	result.Proxy = updated
	return result, true, nil
}

type ProxyCheckAllResult struct {
	Total   int                `json:"total"`
	Active  int                `json:"active"`
	Failed  int                `json:"failed"`
	Results []ProxyCheckResult `json:"results"`
}

func (s *ProxyStore) checkAll() (ProxyCheckAllResult, error) {
	proxies, err := s.list()
	if err != nil {
		return ProxyCheckAllResult{}, err
	}

	type jobResult struct {
		index  int
		result ProxyCheckResult
		err    error
	}

	results := make([]ProxyCheckResult, len(proxies))
	jobs := make(chan int, len(proxies))
	out := make(chan jobResult, len(proxies))

	workers := 4
	if len(proxies) < workers {
		workers = len(proxies)
	}
	if workers < 1 {
		return ProxyCheckAllResult{Results: []ProxyCheckResult{}}, nil
	}

	for w := 0; w < workers; w++ {
		go func() {
			for i := range jobs {
				result, _, err := s.check(proxies[i].ID)
				out <- jobResult{index: i, result: result, err: err}
			}
		}()
	}

	for i := range proxies {
		jobs <- i
	}
	close(jobs)

	summary := ProxyCheckAllResult{
		Total:   len(proxies),
		Results: results,
	}
	for range proxies {
		item := <-out
		if item.err != nil {
			return ProxyCheckAllResult{}, item.err
		}
		results[item.index] = item.result
		if item.result.OK {
			summary.Active++
		} else {
			summary.Failed++
		}
	}
	summary.Results = results
	return summary, nil
}

func (s *ProxyStore) handleCheckAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result, err := s.checkAll()
	if err != nil {
		log.Printf("check all proxies: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update proxy statuses")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func probeProxy(p Proxy) error {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	transport := &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          1,
		IdleConnTimeout:       5 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	addr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port))

	switch p.Protocol {
	case "http", "https":
		proxyURL := &url.URL{
			Scheme: p.Protocol,
			Host:   addr,
		}
		if p.Username != "" {
			proxyURL.User = url.UserPassword(p.Username, p.Password)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.DialContext = (&net.Dialer{Timeout: 8 * time.Second}).DialContext
	case "socks5":
		var auth *xproxy.Auth
		if p.Username != "" {
			auth = &xproxy.Auth{User: p.Username, Password: p.Password}
		}
		dialer, err := xproxy.SOCKS5("tcp", addr, auth, &net.Dialer{Timeout: 8 * time.Second})
		if err != nil {
			return fmt.Errorf("socks5 setup failed: %w", err)
		}
		contextDialer, ok := dialer.(xproxy.ContextDialer)
		if !ok {
			return fmt.Errorf("socks5 dialer does not support deadlines")
		}
		transport.DialContext = contextDialer.DialContext
	default:
		return fmt.Errorf("unsupported protocol %q", p.Protocol)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   12 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyCheckURL, nil)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1024))

	// Any HTTP response from the probe URL means the proxy path worked.
	if res.StatusCode >= 200 && res.StatusCode < 500 {
		return nil
	}
	return fmt.Errorf("unexpected status %d", res.StatusCode)
}
