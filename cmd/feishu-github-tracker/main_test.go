package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

func TestNewServerConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Server.Host = "127.0.0.1"
	cfg.Server.Server.Port = 0
	cfg.Server.Server.Timeout = 5

	h := http.NewServeMux()
	srv := NewServer(cfg, h)
	if srv == nil {
		t.Fatalf("expected server, got nil")
	}
	if srv.ReadTimeout == 0 || srv.WriteTimeout == 0 {
		t.Fatalf("expected non-zero timeouts")
	}
}

func TestInitializeConfigDirCopiesOnlyMissingFiles(t *testing.T) {
	defaultDir := t.TempDir()
	configDir := t.TempDir()

	writeTestFile(t, filepath.Join(defaultDir, "server.yaml"), "default server")
	writeTestFile(t, filepath.Join(defaultDir, "nested", "events.yaml"), "default events")
	writeTestFile(t, filepath.Join(configDir, "server.yaml"), "custom server")

	if err := initializeConfigDir(defaultDir, configDir, nil); err != nil {
		t.Fatalf("initializeConfigDir() error = %v", err)
	}

	assertTestFileContents(t, filepath.Join(configDir, "server.yaml"), "custom server")
	assertTestFileContents(t, filepath.Join(configDir, "nested", "events.yaml"), "default events")

	writeTestFile(t, filepath.Join(configDir, "nested", "events.yaml"), "custom events")
	if err := initializeConfigDir(defaultDir, configDir, nil); err != nil {
		t.Fatalf("second initializeConfigDir() error = %v", err)
	}
	assertTestFileContents(t, filepath.Join(configDir, "nested", "events.yaml"), "custom events")
}

// TestInitializeConfigDirSkipsLegacyShadows verifies that a legacy flat file in
// the live config dir keeps the corresponding example subdirectory from being
// seeded, so seeded defaults never shadow the user's active legacy config.
func TestInitializeConfigDirSkipsLegacyShadows(t *testing.T) {
	defaultDir := t.TempDir()
	configDir := t.TempDir()

	writeTestFile(t, filepath.Join(defaultDir, "patterns", "example.yaml"), "example pattern")
	writeTestFile(t, filepath.Join(defaultDir, "events", "definitions", "push.yaml"), "example event")
	writeTestFile(t, filepath.Join(defaultDir, "feishu-bots.yaml"), "example bots")
	// User still runs the legacy flat repos.yaml.
	writeTestFile(t, filepath.Join(configDir, "repos.yaml"), "user legacy repos")

	if err := initializeConfigDir(defaultDir, configDir, config.SeedingSkips(configDir)); err != nil {
		t.Fatalf("initializeConfigDir() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(configDir, "patterns")); !os.IsNotExist(err) {
		t.Error("patterns/ must not be seeded next to a legacy repos.yaml")
	}
	// No legacy events.yaml in configDir → events/ is seeded normally.
	assertTestFileContents(t, filepath.Join(configDir, "events", "definitions", "push.yaml"), "example event")
	assertTestFileContents(t, filepath.Join(configDir, "feishu-bots.yaml"), "example bots")
	assertTestFileContents(t, filepath.Join(configDir, "repos.yaml"), "user legacy repos")
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assertTestFileContents(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(got) != want {
		t.Errorf("ReadFile(%q) = %q, want %q", path, got, want)
	}
}
