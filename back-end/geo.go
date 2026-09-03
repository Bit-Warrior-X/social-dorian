package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type geoLookupRequest struct {
	Host string `json:"host"`
}

type geoLookupResponse struct {
	Host        string `json:"host"`
	IP          string `json:"ip"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
}

type ipAPIResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	Query       string `json:"query"`
}

func handleGeoCountry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var input geoLookupRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	host := strings.TrimSpace(input.Host)
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if i := strings.IndexAny(host, "/:"); i >= 0 {
		host = host[:i]
	}
	if host == "" {
		writeError(w, http.StatusBadRequest, "host is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	ip, err := resolveLookupHost(ctx, host)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	country, code, err := lookupCountryByIP(ctx, ip)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, geoLookupResponse{
		Host:        host,
		IP:          ip,
		Country:     country,
		CountryCode: code,
	})
}

func resolveLookupHost(ctx context.Context, host string) (string, error) {
	if parsed := net.ParseIP(host); parsed != nil {
		if parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() || parsed.IsUnspecified() {
			return "", fmt.Errorf("host %s is a private or local address and cannot be geolocated", host)
		}
		return parsed.String(), nil
	}

	resolver := &net.Resolver{}
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return "", fmt.Errorf("could not resolve host %s", host)
	}
	for _, addr := range addrs {
		ip := addr.IP
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			continue
		}
		// Prefer IPv4 for broader geo DB coverage.
		if v4 := ip.To4(); v4 != nil {
			return v4.String(), nil
		}
	}
	for _, addr := range addrs {
		if addr.IP != nil && !addr.IP.IsLoopback() && !addr.IP.IsPrivate() {
			return addr.IP.String(), nil
		}
	}
	return "", fmt.Errorf("no public IP found for host %s", host)
}

func lookupCountryByIP(ctx context.Context, ip string) (string, string, error) {
	endpoint := "http://ip-api.com/json/" + url.PathEscape(ip) + "?fields=status,message,country,countryCode,query"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}

	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("geolocation lookup failed")
	}
	defer res.Body.Close()

	var payload ipAPIResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", "", fmt.Errorf("invalid geolocation response")
	}
	if payload.Status != "success" {
		msg := strings.TrimSpace(payload.Message)
		if msg == "" {
			msg = "lookup failed"
		}
		return "", "", fmt.Errorf("geolocation failed: %s", msg)
	}
	if payload.CountryCode == "" {
		return "", "", fmt.Errorf("no country found for %s", ip)
	}
	return payload.Country, strings.ToUpper(payload.CountryCode), nil
}
