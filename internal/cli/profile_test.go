package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grantcarthew/webctl/internal/browser"
)

func TestStartupErrorHint(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		wantEmpty bool
		wantHas   string
	}{
		{
			name:    "system profile in use gets targeted hint, not orphan reaping",
			err:     fmt.Errorf("failed to start browser: %w", browser.ErrSystemProfileInUse),
			wantHas: "default profile",
		},
		{
			name:    "port conflict keeps the orphan-reaping hint",
			err:     fmt.Errorf("%w: 9222", browser.ErrPortInUse),
			wantHas: "stop --force",
		},
		{
			name:      "unrelated error gets no hint",
			err:       errors.New("some other failure"),
			wantEmpty: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := startupErrorHint(tc.err)
			if tc.wantEmpty {
				if got != "" {
					t.Errorf("expected no hint, got %q", got)
				}
				return
			}
			if got == "" {
				t.Fatalf("expected a hint containing %q, got empty", tc.wantHas)
			}
			if !strings.Contains(got, tc.wantHas) {
				t.Errorf("expected hint to contain %q, got %q", tc.wantHas, got)
			}
			// The system-profile hint must never suggest stop --force.
			if errors.Is(tc.err, browser.ErrSystemProfileInUse) && strings.Contains(got, "stop --force") {
				t.Errorf("system-profile hint must not mention stop --force, got %q", got)
			}
		})
	}
}

func TestResolveProfile_DefaultPersistent(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdg)

	got, err := resolveProfile(false, "", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(xdg, "webctl", "profile")
	if got != want {
		t.Errorf("expected default profile %q, got %q", want, got)
	}
	if got == "" || got == browser.UserDataDirDefault {
		t.Errorf("default profile must be a concrete path, got %q", got)
	}

	// The resolver must create the directory itself.
	if info, statErr := os.Stat(got); statErr != nil {
		t.Errorf("expected resolver to create %q: %v", got, statErr)
	} else if !info.IsDir() {
		t.Errorf("expected %q to be a directory", got)
	}
}

func TestResolveProfile_TempIsEmpty(t *testing.T) {
	got, err := resolveProfile(true, "", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("temp profile must resolve to empty string, got %q", got)
	}
}

func TestResolveProfile_SystemIsDefaultSentinel(t *testing.T) {
	got, err := resolveProfile(false, "", false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != browser.UserDataDirDefault {
		t.Errorf("system profile must resolve to %q, got %q", browser.UserDataDirDefault, got)
	}
}

func TestResolveProfile_UserDataDirIsAbsolute(t *testing.T) {
	got, err := resolveProfile(false, "relative/path", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("--user-data-dir must resolve to an absolute path, got %q", got)
	}
	if got == browser.UserDataDirDefault {
		t.Errorf("--user-data-dir must never alias the system sentinel")
	}
}

func TestResolveProfile_EmptyUserDataDirRejected(t *testing.T) {
	_, err := resolveProfile(false, "   ", true, false)
	if err == nil {
		t.Fatal("expected an error for an empty --user-data-dir value")
	}
}

func TestResolveProfile_MutuallyExclusive(t *testing.T) {
	cases := []struct {
		name           string
		temp           bool
		userDataDir    string
		userDataDirSet bool
		system         bool
	}{
		{"temp+system", true, "", false, true},
		{"temp+userdatadir", true, "/tmp/x", true, false},
		{"userdatadir+system", false, "/tmp/x", true, true},
		{"all three", true, "/tmp/x", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveProfile(tc.temp, tc.userDataDir, tc.userDataDirSet, tc.system)
			if err == nil {
				t.Errorf("expected mutual-exclusion error for %s", tc.name)
			}
		})
	}
}
