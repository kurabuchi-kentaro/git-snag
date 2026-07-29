package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_versionFlagPrintsAndExitsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "git-snag") {
		t.Errorf("version output = %q, want it to name the tool", stdout.String())
	}
}

func TestRun_nonexistentScanRootFailsWithMessage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{filepath.Join(t.TempDir(), "missing")}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("nonexistent scan root should exit non-zero")
	}
	if !strings.Contains(stderr.String(), "scan root") {
		t.Errorf("stderr = %q, want a scan-root error message", stderr.String())
	}
}

func TestRun_missingGitBinaryFailsWithMessage(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty PATH: no git anywhere
	var stdout, stderr bytes.Buffer
	code := Run([]string{t.TempDir()}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("missing git should exit non-zero")
	}
	if !strings.Contains(stderr.String(), "git executable not found") {
		t.Errorf("stderr = %q, want a git-not-found message", stderr.String())
	}
}

func TestRun_unknownFlagFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--no-such-flag"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("unknown flag should exit non-zero")
	}
}

func TestRun_customConfigPathIsActuallyLoaded(t *testing.T) {
	// A broken YAML at the --config path must abort startup (REQ-A6),
	// which also proves the flag-specified file is the one being read.
	path := filepath.Join(t.TempDir(), "custom.yaml")
	if err := os.WriteFile(path, []byte("scan: [broken\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--config", path, t.TempDir()}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("broken config should exit non-zero")
	}
	if !strings.Contains(stderr.String(), "custom.yaml") {
		t.Errorf("stderr = %q, want it to reference the custom config", stderr.String())
	}
}
