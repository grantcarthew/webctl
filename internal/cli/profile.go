package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grantcarthew/webctl/internal/browser"
)

// resolveProfile maps the mutually exclusive profile flags to the single
// UserDataDir value passed down to the daemon and browser layers.
//
// The browser layer overloads UserDataDir: an empty string means "temp profile,
// created on start and deleted on stop" and browser.UserDataDirDefault means
// "system profile, no --user-data-dir flag". Mode resolution lives here so the
// browser stays a thin executor of a concrete value.
//
//   - no flag:        the persistent default profile under XDG data home
//   - temp:           empty string (browser creates and later deletes a temp dir)
//   - userDataDir:    an absolute path; never aliases the empty/"default" sentinels
//   - system:         browser.UserDataDirDefault
//
// userDataDirSet reports whether --user-data-dir was given, so an explicit empty
// value can be rejected rather than silently resolving to the temp behaviour.
func resolveProfile(temp bool, userDataDir string, userDataDirSet bool, system bool) (string, error) {
	selected := 0
	if temp {
		selected++
	}
	if userDataDirSet {
		selected++
	}
	if system {
		selected++
	}
	if selected > 1 {
		return "", errors.New("--temp-profile, --user-data-dir, and --system-profile are mutually exclusive")
	}

	switch {
	case temp:
		return "", nil
	case system:
		return browser.UserDataDirDefault, nil
	case userDataDirSet:
		if strings.TrimSpace(userDataDir) == "" {
			return "", errors.New("--user-data-dir requires a non-empty path")
		}
		abs, err := filepath.Abs(userDataDir)
		if err != nil {
			return "", fmt.Errorf("resolve --user-data-dir path: %w", err)
		}
		return abs, nil
	default:
		return defaultProfileDir()
	}
}

// defaultProfileDir returns the persistent default profile directory,
// $XDG_DATA_HOME/webctl/profile, falling back to ~/.local/share/webctl/profile
// when XDG_DATA_HOME is unset. It creates the directory with mode 0700 so the
// default does not depend on Chrome creating it and so the directory holding
// cookies and session state is not world-readable.
func defaultProfileDir() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine home directory for default profile: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}

	dir := filepath.Join(base, "webctl", "profile")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create default profile directory %s: %w", dir, err)
	}
	return dir, nil
}
