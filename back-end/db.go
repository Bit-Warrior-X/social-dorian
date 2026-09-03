package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		// Do not override variables already set in the environment.
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func openDB() (*sql.DB, error) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		user := envOr("DB_USER", "root")
		pass := os.Getenv("DB_PASSWORD")
		host := envOr("DB_HOST", "127.0.0.1")
		port := envOr("DB_PORT", "3306")
		name := envOr("DB_NAME", "doriansocial")
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC&clientFoundRows=true",
			user, pass, host, port, name)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func nullString(v string) interface{} {
	if v == "" {
		return nil
	}
	return v
}

func nullProxyID(mode string, id int) interface{} {
	if mode == "none" || id < 1 {
		return nil
	}
	return id
}

func scanProxyID(n sql.NullInt64) int {
	if n.Valid {
		return int(n.Int64)
	}
	return 0
}

func formatDate(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format("2006-01-02")
}

func formatDateTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
