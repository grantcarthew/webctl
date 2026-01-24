package browser

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestBuildArgs_DefaultPort(t *testing.T) {
	t.Parallel()

	opts := LaunchOptions{}
	args := buildArgs(opts)

	found := false
	for _, arg := range args {
		if strings.Contains(arg, "--remote-debugging-port=9222") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected default port 9222, args: %v", args)
	}
}

func TestBuildArgs_CustomPort(t *testing.T) {
	t.Parallel()

	opts := LaunchOptions{Port: 9333}
	args := buildArgs(opts)

	found := false
	for _, arg := range args {
		if strings.Contains(arg, "--remote-debugging-port=9333") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected port 9333, args: %v", args)
	}
}

func TestBuildArgs_Headless(t *testing.T) {
	t.Parallel()

	opts := LaunchOptions{Headless: true}
	args := buildArgs(opts)

	found := false
	for _, arg := range args {
		if arg == "--headless" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected --headless flag, args: %v", args)
	}
}

func TestBuildArgs_NotHeadless(t *testing.T) {
	t.Parallel()

	opts := LaunchOptions{Headless: false}
	args := buildArgs(opts)

	for _, arg := range args {
		if strings.Contains(arg, "headless") {
			t.Errorf("unexpected headless flag: %s", arg)
		}
	}
}

func TestBuildArgs_UserDataDir(t *testing.T) {
	t.Parallel()

	opts := LaunchOptions{UserDataDir: "/tmp/test-profile"}
	args := buildArgs(opts)

	found := false
	for _, arg := range args {
		if arg == "--user-data-dir=/tmp/test-profile" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected user-data-dir flag, args: %v", args)
	}
}

func TestBuildArgs_UserDataDirDefault(t *testing.T) {
	t.Parallel()

	// "default" should NOT add --user-data-dir flag
	opts := LaunchOptions{UserDataDir: UserDataDirDefault}
	args := buildArgs(opts)

	for _, arg := range args {
		if strings.Contains(arg, "--user-data-dir") {
			t.Errorf("unexpected user-data-dir flag with 'default': %v", args)
		}
	}
}

func TestBuildArgs_RequiredFlags(t *testing.T) {
	t.Parallel()

	opts := LaunchOptions{}
	args := buildArgs(opts)

	required := []string{
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-sync",
		"--disable-popup-blocking",
		"about:blank",
	}

	for _, req := range required {
		found := false
		for _, arg := range args {
			if arg == req {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing required arg %s, args: %v", req, args)
		}
	}
}

func TestBuildArgs_PlatformFlags(t *testing.T) {
	t.Parallel()

	opts := LaunchOptions{}
	args := buildArgs(opts)

	switch runtime.GOOS {
	case "darwin":
		found := false
		for _, arg := range args {
			if arg == "--use-mock-keychain" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected --use-mock-keychain on macOS, args: %v", args)
		}
	case "linux":
		found := false
		for _, arg := range args {
			if arg == "--password-store=basic" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected --password-store=basic on Linux, args: %v", args)
		}
	}
}

func TestCreateTempDataDir(t *testing.T) {
	t.Parallel()

	dir, err := createTempDataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	if dir == "" {
		t.Error("expected non-empty dir")
	}

	if !strings.Contains(dir, "webctl-chrome-") {
		t.Errorf("expected webctl-chrome- prefix, got %s", dir)
	}
}
