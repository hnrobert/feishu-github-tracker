package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config directory for testing
	tmpDir := t.TempDir()

	// Create minimal test configs
	serverYAML := `
server:
  host: "127.0.0.1"
  port: 4594
  secret: "test_secret"
  log_level: "debug"
  max_payload_size: "5MB"
  timeout: 15
allowed_sources:
  - "github.com"
`

	reposYAML := `
repos:
  - pattern: "test/repo"
    events:
      push:
    notify_to:
      - test-bot
`

	eventsYAML := `
event_sets:
  basic:
    push:
events:
  push:
    branches:
      - "*"
`

	botsYAML := `
feishu_bots:
  - alias: "test-bot"
    url: "https://example.com/webhook"
`

	templatesYAML := `
{
	// templates.jsonc
	"templates": {
		"push": {
			"payloads": [
				{
					"tags": ["default"],
					"payload": { "msg_type": "text", "content": { "text": "test" } }
				}
			]
		}
	}
}
`

	// Write test config files
	files := map[string]string{
		"server.yaml":      serverYAML,
		"repos.yaml":       reposYAML,
		"events.yaml":      eventsYAML,
		"feishu-bots.yaml": botsYAML,
		"templates.jsonc":  templatesYAML,
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", name, err)
		}
	}

	// Test loading
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Basic validation
	if cfg.Server.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", cfg.Server.Server.Host)
	}

	if cfg.Server.Server.Port != 4594 {
		t.Errorf("Expected port 4594, got %d", cfg.Server.Server.Port)
	}

	if len(cfg.Repos.Repos) != 1 {
		t.Errorf("Expected 1 repo, got %d", len(cfg.Repos.Repos))
	}

	if len(cfg.FeishuBots.FeishuBots) != 1 {
		t.Errorf("Expected 1 bot, got %d", len(cfg.FeishuBots.FeishuBots))
	}
}
