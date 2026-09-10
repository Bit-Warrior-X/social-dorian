package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func main() {
	if err := loadDotEnv(".env"); err != nil {
		log.Fatalf("failed to load .env: %v", err)
	}

	db, err := openDB()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	authStore := newAuthStore(db)
	if err := authStore.ensureSchema(); err != nil {
		log.Fatalf("failed to prepare auth tables: %v", err)
	}
	if err := authStore.ensureAdmin(); err != nil {
		log.Fatalf("failed to prepare admin user: %v", err)
	}

	proxyStore := newProxyStore(db)
	accountStore := newAccountStore(db, proxyStore)
	gmailStore := newGmailStore(db)
	dashboardStore := newDashboardStore(db)
	taskStore := newTaskStore(db)
	if err := taskStore.ensureSchema(); err != nil {
		log.Fatalf("failed to prepare task tables: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/auth/", authStore.handleAuth)
	mux.HandleFunc("/api/geo/country", handleGeoCountry)
	mux.HandleFunc("/api/dashboard", dashboardStore.handleDashboard)
	mux.HandleFunc("/api/accounts", accountStore.handleAccounts)
	mux.HandleFunc("/api/accounts/", accountStore.handleAccountByID)
	mux.HandleFunc("/api/proxies/check-all", proxyStore.handleCheckAll)
	mux.HandleFunc("/api/proxies", proxyStore.handleProxies)
	mux.HandleFunc("/api/proxies/", proxyStore.handleProxyByID)
	mux.HandleFunc("/api/gmails", gmailStore.handleGmails)
	mux.HandleFunc("/api/gmails/", gmailStore.handleGmailByID)
	mux.HandleFunc("/api/sessions/", accountStore.handleBrowseSessions)
	mux.HandleFunc("/api/tasks", taskStore.handleTasks)
	mux.HandleFunc("/api/tasks/", taskStore.handleTaskByID)
	mux.HandleFunc("/api/credits", taskStore.handleCredits)
	mux.HandleFunc("/api/busy-accounts", taskStore.handleBusyAccounts)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Social Dorian API listening on :%s", port)
	if err := http.ListenAndServe(":"+port, corsMiddleware(authStore.wrap(mux))); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Service: "social-dorian-api",
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Add("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
