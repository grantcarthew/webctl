package browser

import (
	"errors"
	"runtime"
	"testing"
	"time"
)

// trueBinary returns a path to an executable that exits immediately with status
// 0 and never serves CDP, used to exercise the startup fail-fast path.
func trueBinary(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("no /bin/true on windows")
	}
	return "/bin/true"
}

func TestStartupFailFast_SystemProfile(t *testing.T) {
	bin := trueBinary(t)

	start := time.Now()
	_, err := StartWithBinary(bin, LaunchOptions{
		Port:        findTestPort(t),
		UserDataDir: UserDataDirDefault,
	})
	elapsed := time.Since(start)

	if !errors.Is(err, ErrSystemProfileInUse) {
		t.Fatalf("expected ErrSystemProfileInUse, got %v", err)
	}
	// Must fail well inside the 30s CDP timeout.
	if elapsed > 10*time.Second {
		t.Errorf("fail-fast took too long: %v", elapsed)
	}
}

func TestStartupFailFast_NonSystemProfileGenericError(t *testing.T) {
	bin := trueBinary(t)

	_, err := StartWithBinary(bin, LaunchOptions{
		Port:        findTestPort(t),
		UserDataDir: t.TempDir(),
	})

	if err == nil {
		t.Fatal("expected an error when the browser exits during startup")
	}
	if errors.Is(err, ErrSystemProfileInUse) {
		t.Errorf("non-system profile must not report ErrSystemProfileInUse, got %v", err)
	}
	if errors.Is(err, ErrStartTimeout) {
		t.Errorf("expected fail-fast, not a start timeout, got %v", err)
	}
}

// findTestPort returns an available port for a test launch.
func findTestPort(t *testing.T) int {
	t.Helper()
	port, err := findFreePort(DefaultPort + 1000)
	if err != nil {
		t.Fatalf("could not find a free port: %v", err)
	}
	return port
}
