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
  match_all_rules: true
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
  - alias: "test-bot-cn"
    url: "https://example.com/webhook-cn"
    template: "cn"
`

	templatesConfig := `
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

	templatesCnConfig := `
{
	// templates.cn.jsonc
	"templates": {
		"push": {
			"payloads": [
				{
					"tags": ["default"],
					"payload": { "msg_type": "text", "content": { "text": "测试" } }
				}
			]
		}
	}
}
`

	// Write test config files
	files := map[string]string{
		"server.yaml":        serverYAML,
		"repos.yaml":         reposYAML,
		"events.yaml":        eventsYAML,
		"feishu-bots.yaml":   botsYAML,
		"templates.jsonc":    templatesConfig,
		"templates.cn.jsonc": templatesCnConfig,
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
	if !cfg.Server.Server.MatchAllRules {
		t.Error("Expected match_all_rules to be true")
	}

	if len(cfg.Repos.Repos) != 1 {
		t.Errorf("Expected 1 repo, got %d", len(cfg.Repos.Repos))
	}

	if len(cfg.FeishuBots.FeishuBots) != 2 {
		t.Errorf("Expected 2 bots, got %d", len(cfg.FeishuBots.FeishuBots))
	}

	// Test template loading
	if len(cfg.Templates) != 2 {
		t.Errorf("Expected 2 templates (default + cn), got %d", len(cfg.Templates))
	}

	if _, ok := cfg.Templates["default"]; !ok {
		t.Error("Expected default template to be loaded")
	}

	if _, ok := cfg.Templates["cn"]; !ok {
		t.Error("Expected cn template to be loaded")
	}

	// Test GetBotTemplate
	if tmpl := cfg.GetBotTemplate("test-bot"); tmpl != "default" {
		t.Errorf("Expected default template for test-bot, got %s", tmpl)
	}

	if tmpl := cfg.GetBotTemplate("test-bot-cn"); tmpl != "cn" {
		t.Errorf("Expected cn template for test-bot-cn, got %s", tmpl)
	}

	if tmpl := cfg.GetBotTemplate("non-existent"); tmpl != "default" {
		t.Errorf("Expected default template for non-existent bot, got %s", tmpl)
	}

	// Test GetTemplateConfig
	defaultTmpl := cfg.GetTemplateConfig("default")
	if _, ok := defaultTmpl.Templates["push"]; !ok {
		t.Error("Expected push template in default config")
	}

	cnTmpl := cfg.GetTemplateConfig("cn")
	if _, ok := cnTmpl.Templates["push"]; !ok {
		t.Error("Expected push template in cn config")
	}

	// Test fallback for non-existent template
	fallbackTmpl := cfg.GetTemplateConfig("non-existent")
	if _, ok := fallbackTmpl.Templates["push"]; !ok {
		t.Error("Expected fallback to default template")
	}
}

// TestLoadRealTemplates tests loading the actual example-configs directory
// using config.Load (which transparently handles both old flat and new split layouts).

func TestLoadRealTemplates(t *testing.T) {
	projectRoot := filepath.Join("..", "..", "example-configs")

	// Ensure the directory exists
	if _, err := os.Stat(projectRoot); os.IsNotExist(err) {
		t.Skipf("example-configs not found at %s", projectRoot)
	}

	// Use config.Load which handles both layouts
	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("Failed to load example-configs: %v", err)
	}

	// Test default templates
	t.Run("DefaultTemplates", func(t *testing.T) {
		defaultTmpl := cfg.GetTemplateConfig("default")
		if _, ok := defaultTmpl.Templates["ping"]; !ok {
			t.Error("Expected ping template in default")
		}
		commonTemplates := []string{"push", "pull_request", "issues", "issue_comment"}
		for _, tmpl := range commonTemplates {
			if _, ok := defaultTmpl.Templates[tmpl]; !ok {
				t.Errorf("Expected %s template in default", tmpl)
			}
		}
		t.Logf("Default: %d templates", len(defaultTmpl.Templates))
	})

	// Test CN templates if present
	t.Run("ChineseTemplates", func(t *testing.T) {
		if _, ok := cfg.Templates["cn"]; !ok {
			t.Skip("cn locale not found")
		}
		cnTmpl := cfg.GetTemplateConfig("cn")
		if _, ok := cnTmpl.Templates["ping"]; !ok {
			t.Error("Expected ping template in cn")
		}
		t.Logf("CN: %d templates", len(cnTmpl.Templates))
	})

	// Validate ping template structure
	t.Run("ValidatePingTemplate", func(t *testing.T) {
		defaultTmpl := cfg.GetTemplateConfig("default")
		pingTemplate, ok := defaultTmpl.Templates["ping"]
		if !ok {
			t.Fatal("ping template not found")
		}
		if len(pingTemplate.Payloads) == 0 {
			t.Fatal("ping template has no payloads")
		}
		firstPayload := pingTemplate.Payloads[0]
		if len(firstPayload.Tags) == 0 {
			t.Error("ping template payload has no tags")
		}

		if firstPayload.Payload == nil {
			t.Fatal("ping template payload is nil")
		}

		// Check for required fields in Feishu card
		if msgType, ok := firstPayload.Payload["msg_type"]; !ok || msgType != "interactive" {
			t.Error("ping template should have msg_type: interactive")
		}

		t.Logf("Ping template has %d payload(s)", len(pingTemplate.Payloads))
	})

	// Test loading complete config with real templates

	// LoadCompleteConfigWithRealTemplates: just use config.Load on example-configs
	t.Run("LoadCompleteConfigWithRealTemplates", func(t *testing.T) {
		cfg, err := Load(projectRoot)
		if err != nil {
			t.Fatalf("Failed to load config with real templates: %v", err)
		}

		if _, ok := cfg.Templates["default"]; !ok {
			t.Error("Expected default template to be loaded")
		}

		defaultTmpl := cfg.GetTemplateConfig("default")
		if _, ok := defaultTmpl.Templates["ping"]; !ok {
			t.Error("Expected ping template in default config")
		}

		t.Log("Successfully loaded complete config with real templates")
	})
}
