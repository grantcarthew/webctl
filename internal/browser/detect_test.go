package browser

import (
	"os"
	"runtime"
	"testing"
)

func TestChromePaths_ReturnsPathsForCurrentOS(t *testing.T) {
	t.Parallel()

	paths := chromePaths()

	switch runtime.GOOS {
	case "darwin", "linux":
		if len(paths) == 0 {
			t.Error("expected non-empty paths for supported OS")
		}
	default:
		if len(paths) != 0 {
			t.Errorf("expected empty paths for unsupported OS, got %d", len(paths))
		}
	}
}

func TestFindChrome_RespectsEnvVar(t *testing.T) {
	// Create a temp file to act as a fake chrome binary
	tmpFile, err := os.CreateTemp("", "fake-chrome-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Set env var
	original := os.Getenv("WEBCTL_CHROME")
	os.Setenv("WEBCTL_CHROME", tmpFile.Name())
	defer os.Setenv("WEBCTL_CHROME", original)

	path, err := FindChrome()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != tmpFile.Name() {
		t.Errorf("expected %s, got %s", tmpFile.Name(), path)
	}
}

func TestFindChrome_EnvVarInvalidPath(t *testing.T) {
	original := os.Getenv("WEBCTL_CHROME")
	os.Setenv("WEBCTL_CHROME", "/nonexistent/path/to/chrome")
	defer os.Setenv("WEBCTL_CHROME", original)

	_, err := FindChrome()
	if err != ErrChromeNotFound {
		t.Errorf("expected ErrChromeNotFound, got %v", err)
	}
}

func TestFindChrome_SearchesPaths(t *testing.T) {
	// Clear env var to test path search
	original := os.Getenv("WEBCTL_CHROME")
	os.Unsetenv("WEBCTL_CHROME")
	defer os.Setenv("WEBCTL_CHROME", original)

	// This test may pass or fail depending on whether Chrome is installed
	// We just verify it doesn't panic
	path, err := FindChrome()
	if err == nil {
		if path == "" {
			t.Error("found chrome but path is empty")
		}
		t.Logf("Found Chrome at: %s", path)
	} else if err != ErrChromeNotFound {
		t.Errorf("unexpected error type: %v", err)
	}
}
