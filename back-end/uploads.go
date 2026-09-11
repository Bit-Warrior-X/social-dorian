package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	uploadRoot     = "/var/lib/dorian-browser/uploads"
	maxUploadBytes = 80 << 20 // 80 MiB
)

func ensureUploadRoot() error {
	return os.MkdirAll(uploadRoot, 0o755)
}

func handleUploads(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && (r.URL.Path == "/api/uploads" || r.URL.Path == "/api/uploads/"):
		handleUploadCreate(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/uploads/"):
		handleUploadGet(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleUploadCreate(w http.ResponseWriter, r *http.Request) {
	if err := ensureUploadRoot(); err != nil {
		writeError(w, http.StatusInternalServerError, "upload storage unavailable")
		return
	}
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
		".mp4": true, ".mov": true, ".webm": true,
	}
	if !allowed[ext] {
		writeError(w, http.StatusBadRequest, "unsupported file type (use image or video)")
		return
	}

	token, err := randomToken(16)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store upload")
		return
	}
	name := fmt.Sprintf("%s_%d%s", token, time.Now().Unix(), ext)
	dest := filepath.Join(uploadRoot, name)
	out, err := os.Create(dest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store upload")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, io.LimitReader(file, maxUploadBytes)); err != nil {
		_ = os.Remove(dest)
		writeError(w, http.StatusInternalServerError, "failed to store upload")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"url":      "/api/uploads/" + name,
		"path":     dest,
		"filename": header.Filename,
		"size":     header.Size,
	})
}

func handleUploadGet(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(strings.TrimPrefix(r.URL.Path, "/api/uploads/"))
	if name == "" || name == "." || strings.Contains(name, "..") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	path := filepath.Join(uploadRoot, name)
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	http.ServeFile(w, r, path)
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func localMediaPath(mediaURL string) string {
	mediaURL = strings.TrimSpace(mediaURL)
	if mediaURL == "" {
		return ""
	}
	if strings.HasPrefix(mediaURL, "/api/uploads/") {
		name := filepath.Base(mediaURL)
		path := filepath.Join(uploadRoot, name)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path
		}
	}
	if strings.HasPrefix(mediaURL, uploadRoot) {
		if st, err := os.Stat(mediaURL); err == nil && !st.IsDir() {
			return mediaURL
		}
	}
	return ""
}

func init() {
	if err := ensureUploadRoot(); err != nil {
		log.Printf("upload root: %v", err)
	}
}
