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

// TestLoadRealTemplates tests loading the actual templates.jsonc and templates.cn.jsonc files
// This test is skipped by default as the real template files are very large and may have formatting issues

func TestLoadRealTemplates(t *testing.T) {
	// Use real templates from the project's configs directory.
	// This test requires the files to exist in the repository root.
	projectRoot := filepath.Join("..", "..", "configs")

	// Ensure templates.jsonc exists
	templatesPath := filepath.Join(projectRoot, "templates.jsonc")
	if _, err := os.Stat(templatesPath); os.IsNotExist(err) {
		t.Fatalf("Required file missing: %s", templatesPath)
	}

	// Ensure templates.cn.jsonc exists
	templatesCnPath := filepath.Join(projectRoot, "templates.cn.jsonc")
	if _, err := os.Stat(templatesCnPath); os.IsNotExist(err) {
		t.Fatalf("Required file missing: %s", templatesCnPath)
	}

	// Test loading templates.jsonc
	t.Run("LoadDefaultTemplates", func(t *testing.T) {
		var templates TemplatesConfig
		err := loadConfigFile(templatesPath, &templates)
		if err != nil {
			t.Fatalf("Failed to load templates.jsonc: %v", err)
		}

		// Check if ping template exists
		if _, ok := templates.Templates["ping"]; !ok {
			t.Error("Expected ping template in templates.jsonc")
		}

		// Check if other common templates exist
		commonTemplates := []string{"push", "pull_request", "issues", "issue_comment"}
		for _, tmpl := range commonTemplates {
			if _, ok := templates.Templates[tmpl]; !ok {
				t.Errorf("Expected %s template in templates.jsonc", tmpl)
			}
		}

		t.Logf("Successfully loaded %d templates from templates.jsonc", len(templates.Templates))
	})

	// Test loading templates.cn.jsonc if it exists
	t.Run("LoadChineseTemplates", func(t *testing.T) {
		templatesCnPath := filepath.Join(projectRoot, "templates.cn.jsonc")
		if _, err := os.Stat(templatesCnPath); os.IsNotExist(err) {
			t.Skip("Skipping test: templates.cn.jsonc not found")
		}

		var templates TemplatesConfig
		err := loadConfigFile(templatesCnPath, &templates)
		if err != nil {
			t.Fatalf("Failed to load templates.cn.jsonc: %v", err)
		}

		// Check if ping template exists
		if _, ok := templates.Templates["ping"]; !ok {
			t.Error("Expected ping template in templates.cn.jsonc")
		}

		// Check if other common templates exist
		commonTemplates := []string{"push", "pull_request", "issues", "issue_comment"}
		for _, tmpl := range commonTemplates {
			if _, ok := templates.Templates[tmpl]; !ok {
				t.Errorf("Expected %s template in templates.cn.jsonc", tmpl)
			}
		}

		t.Logf("Successfully loaded %d templates from templates.cn.jsonc", len(templates.Templates))
	})

	// Test ping template structure
	t.Run("ValidatePingTemplate", func(t *testing.T) {
		var templates TemplatesConfig
		err := loadConfigFile(templatesPath, &templates)
		if err != nil {
			t.Fatalf("Failed to load templates.jsonc: %v", err)
		}

		pingTemplate, ok := templates.Templates["ping"]
		if !ok {
			t.Fatal("ping template not found")
		}

		if len(pingTemplate.Payloads) == 0 {
			t.Fatal("ping template has no payloads")
		}

		// Validate first payload
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
	t.Run("LoadCompleteConfigWithRealTemplates", func(t *testing.T) {
		// Create temp dir with minimal config files
		tmpDir := t.TempDir()

		serverYAML := `
server:
  host: "127.0.0.1"
  port: 4594
  secret: "test_secret"
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
events:
  push:
`

		botsYAML := `
feishu_bots:
  - alias: "test-bot"
    url: "https://example.com/webhook"
`

		// Write minimal configs
		files := map[string]string{
			"server.yaml":      serverYAML,
			"repos.yaml":       reposYAML,
			"events.yaml":      eventsYAML,
			"feishu-bots.yaml": botsYAML,
		}

		for name, content := range files {
			path := filepath.Join(tmpDir, name)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write %s: %v", name, err)
			}
		}

		// Copy real templates
		realTemplatesPath := filepath.Join(projectRoot, "templates.jsonc")
		templatesData, err := os.ReadFile(realTemplatesPath)
		if err != nil {
			t.Fatalf("Failed to read real templates.jsonc: %v", err)
		}

		tmpTemplatesPath := filepath.Join(tmpDir, "templates.jsonc")
		if err := os.WriteFile(tmpTemplatesPath, templatesData, 0644); err != nil {
			t.Fatalf("Failed to write templates.jsonc to temp dir: %v", err)
		}

		// Load config
		cfg, err := Load(tmpDir)
		if err != nil {
			t.Fatalf("Failed to load config with real templates: %v", err)
		}

		// Validate
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
