package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_missingFileYieldsDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	excludes, animation := Resolve(cfg, false)
	if !excludes["node_modules"] {
		t.Error("defaults should exclude node_modules")
	}
	if !animation {
		t.Error("animation should default to enabled")
	}
}

func TestLoad_extraExcludesMergeWithDefaults(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "scan:\n  excludes: [dist, generated]\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	excludes, _ := Resolve(cfg, false)
	for _, want := range []string{"dist", "generated", "node_modules"} {
		if !excludes[want] {
			t.Errorf("excludes missing %q: %v", want, excludes)
		}
	}
}

func TestLoad_animationDisabledInConfig(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "animation:\n  enabled: false\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, animation := Resolve(cfg, false); animation {
		t.Error("config should disable the animation")
	}
}

func TestResolve_noAnimationFlagBeatsConfig(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "animation:\n  enabled: true\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, animation := Resolve(cfg, true); animation {
		t.Error("--no-animation must win over the config")
	}
}

func TestDefaultPath_respectsXDGConfigHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	want := filepath.Join(xdg, "git-snag", "config.yaml")
	if got != want {
		t.Errorf("DefaultPath = %q, want %q", got, want)
	}
}

func TestResolveIcons_flagBeatsConfig(t *testing.T) {
	t.Parallel()
	var cfg Config
	cfg.UI.Icons = "unicode"
	if got := ResolveIcons(cfg, ""); got != "unicode" {
		t.Errorf("ResolveIcons(config only) = %q, want unicode", got)
	}
	if got := ResolveIcons(cfg, "nerd"); got != "nerd" {
		t.Errorf("ResolveIcons(flag set) = %q, want nerd (flag wins)", got)
	}
	if got := ResolveIcons(Config{}, ""); got != "" {
		t.Errorf("ResolveIcons(nothing set) = %q, want empty (UI default applies)", got)
	}
}

func TestLoad_uiIconsParsed(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "ui:\n  icons: unicode\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UI.Icons != "unicode" {
		t.Errorf("UI.Icons = %q, want unicode", cfg.UI.Icons)
	}
}

func TestLoad_brokenYamlFailsClearly(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "scan: [unclosed\n")
	if _, err := Load(path); err == nil {
		t.Fatal("broken YAML should fail, not fall back to defaults")
	}
}

func TestLoad_unreadableFileFailsClearly(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	path := writeConfig(t, "scan:\n  excludes: [x]\n")
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	_, err := Load(path)
	if err == nil {
		t.Fatal("unreadable config should fail, not fall back to defaults")
	}
	if !strings.Contains(err.Error(), "config") {
		t.Errorf("error should mention the config file: %v", err)
	}
}
