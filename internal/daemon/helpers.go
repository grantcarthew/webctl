package daemon

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// isBinaryMimeType returns true if the MIME type represents binary content.
func isBinaryMimeType(mimeType string) bool {
	mimeType = strings.ToLower(mimeType)

	// Check for binary prefixes
	binaryPrefixes := []string{
		"image/",
		"audio/",
		"video/",
		"font/",
	}
	for _, prefix := range binaryPrefixes {
		if strings.HasPrefix(mimeType, prefix) {
			return true
		}
	}

	// Check for specific binary types
	binaryTypes := []string{
		"application/octet-stream",
		"application/pdf",
		"application/zip",
		"application/gzip",
		"application/x-tar",
		"application/x-rar-compressed",
		"application/x-7z-compressed",
		"application/wasm",
	}
	for _, t := range binaryTypes {
		if mimeType == t {
			return true
		}
	}

	return false
}

// getBodiesDir returns the path to the bodies storage directory.
func getBodiesDir() string {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to temp directory if home cannot be determined
			return filepath.Join(os.TempDir(), "webctl-bodies")
		}
		stateHome = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateHome, "webctl", "bodies")
}

// saveBinaryBody saves binary body content to a file and returns the path.
func saveBinaryBody(requestID, url, mimeType, body string, isBase64 bool) (string, error) {
	// Create bodies directory
	bodiesDir := getBodiesDir()
	if err := os.MkdirAll(bodiesDir, 0700); err != nil {
		return "", err
	}

	// Generate filename
	ts := time.Now().Format("2006-01-02-150405")

	// Extract basename from URL
	basename := filepath.Base(url)
	if idx := strings.Index(basename, "?"); idx != -1 {
		basename = basename[:idx]
	}
	if basename == "" || basename == "/" {
		basename = "body"
	}

	// Ensure filename has extension based on MIME type
	ext := extensionFromMimeType(mimeType)
	if ext != "" && !strings.HasSuffix(basename, ext) {
		basename = basename + ext
	}

	// Sanitize request ID for filename (replace dots with dashes)
	safeRequestID := strings.ReplaceAll(requestID, ".", "-")

	filename := fmt.Sprintf("%s-%s-%s", ts, safeRequestID, basename)
	filePath := filepath.Join(bodiesDir, filename)

	// Decode body if base64
	var data []byte
	if isBase64 {
		var err error
		data, err = base64.StdEncoding.DecodeString(body)
		if err != nil {
			return "", err
		}
	} else {
		data = []byte(body)
	}

	// Write file
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", err
	}

	return filePath, nil
}

// extensionFromMimeType returns a file extension for the given MIME type.
func extensionFromMimeType(mimeType string) string {
	mimeType = strings.ToLower(mimeType)

	// Remove parameters (e.g., "text/html; charset=utf-8" -> "text/html")
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	extensions := map[string]string{
		"image/png":       ".png",
		"image/jpeg":      ".jpg",
		"image/gif":       ".gif",
		"image/webp":      ".webp",
		"image/svg+xml":   ".svg",
		"image/x-icon":    ".ico",
		"font/woff":       ".woff",
		"font/woff2":      ".woff2",
		"font/ttf":        ".ttf",
		"font/otf":        ".otf",
		"audio/mpeg":      ".mp3",
		"audio/ogg":       ".ogg",
		"audio/wav":       ".wav",
		"video/mp4":       ".mp4",
		"video/webm":      ".webm",
		"application/pdf": ".pdf",
		"application/zip": ".zip",
	}

	if ext, ok := extensions[mimeType]; ok {
		return ext
	}
	return ""
}

// clearBodiesDir removes all files in the bodies directory.
func clearBodiesDir() error {
	bodiesDir := getBodiesDir()
	entries, err := os.ReadDir(bodiesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			os.Remove(filepath.Join(bodiesDir, entry.Name()))
		}
	}
	return nil
}
