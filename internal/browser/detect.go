// Package browser provides Chrome detection, launch, and target management.
package browser

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
)

// ErrChromeNotFound is returned when no Chrome binary can be located.
var ErrChromeNotFound = errors.New("chrome not found")

// chromePaths returns the list of paths to search for Chrome on the current platform.
func chromePaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
		}
	case "linux":
		return []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
		}
	default:
		return nil
	}
}

// FindChrome searches for a Chrome or Chromium binary on the system.
// It first checks the WEBCTL_CHROME environment variable, then searches
// common installation paths for the current platform.
// Returns the path to the executable or ErrChromeNotFound.
func FindChrome() (string, error) {
	// Check environment variable first
	if envPath := os.Getenv("WEBCTL_CHROME"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		// Env var set but path invalid - still return error with context
		return "", ErrChromeNotFound
	}

	// Search common paths
	for _, path := range chromePaths() {
		found, err := exec.LookPath(path)
		if err == nil {
			return found, nil
		}
	}

	return "", ErrChromeNotFound
}
