package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

// countWeightLines counts how many `weight:` keys appear at the start of a line.
func countWeightLines(data []byte) int {
	count := strings.Count(string(data), "\nweight:")
	if strings.HasPrefix(string(data), "weight:") {
		count++
	}
	return count
}

// writeLegacyFixture seeds a config dir with legacy flat files (repos.yaml,
// events.yaml, templates.jsonc) plus a server.yaml.
func writeLegacyFixture(t *testing.T, dir, serverYAML string) {
	t.Helper()
	files := map[string]string{
		"repos.yaml": `
repos:
  # exact rule first, catch-all last — order must survive migration as weights
  - pattern: "AIMEtherCAT/*"
    events:
      basic:
    notify_to:
      - aim-ecat
  - pattern: "org/repo"
    events:
      push:
    notify_to:
      - dev-team
  - pattern: "*"
    events:
      default:
    notify_to:
      - all
`,
		"events.yaml": `
event_sets:
  basic:
    push:
events:
  push:
    branches:
      - main
`,
		"templates.jsonc": `
{
  // legacy templates
  "templates": {
    "push": {
      "payloads": [
        {"tags": ["default"], "payload": {"msg_type": "text"}}
      ]
    }
  }
}
`,
	}
	if serverYAML != "" {
		files["server.yaml"] = serverYAML
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestMigrateAllExplicit covers the opt-in migration: nothing happens until
// MigrateAll is called, one call migrates everything (weights injected per
// original order, originals backed up), and a second call is a no-op.
func TestMigrateAllExplicit(t *testing.T) {
	tmpDir := t.TempDir()
	writeLegacyFixture(t, tmpDir, "")

	// Before: legacy detected, no split dirs.
	if st := DetectLegacy(tmpDir); !st.Repos || !st.Events || !st.Templates || len(st.Files) != 3 {
		t.Fatalf("DetectLegacy = %+v, want all three types", st)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "patterns")); !os.IsNotExist(err) {
		t.Fatal("patterns/ must not exist before an explicit migration")
	}

	status, err := MigrateAll(tmpDir)
	if err != nil {
		t.Fatalf("MigrateAll failed: %v", err)
	}
	if !status.Any() {
		t.Fatal("MigrateAll reported nothing migrated")
	}

	// All three legacy files moved to legacy/, split dirs populated.
	for _, name := range []string{"repos.yaml", "events.yaml", "templates.jsonc"} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); !os.IsNotExist(err) {
			t.Errorf("%s still present after migration", name)
		}
		if _, err := os.Stat(filepath.Join(tmpDir, "legacy", name)); err != nil {
			t.Errorf("legacy/%s missing: %v", name, err)
		}
	}

	files, err := os.ReadDir(filepath.Join(tmpDir, "patterns"))
	if err != nil || len(files) != 3 {
		t.Fatalf("expected 3 pattern files, got %d (err %v)", len(files), err)
	}
	// Weight order preserved: first rule highest, catch-all zero.
	for _, f := range files {
		data, _ := os.ReadFile(filepath.Join(tmpDir, "patterns", f.Name()))
		if c := countWeightLines(data); c != 1 {
			t.Errorf("%s: expected exactly 1 weight line, got %d\n%s", f.Name(), c, data)
		}
	}
	catchAll, _ := os.ReadFile(filepath.Join(tmpDir, "patterns", "all.yaml"))
	if !strings.Contains(string(catchAll), "weight: 0") {
		t.Errorf("catch-all should carry weight 0:\n%s", catchAll)
	}

	// Second call: nothing left to migrate.
	status, err = MigrateAll(tmpDir)
	if err != nil {
		t.Fatalf("second MigrateAll failed: %v", err)
	}
	if status.Any() {
		t.Errorf("second MigrateAll migrated again: %+v", status)
	}
}

// TestMigrateIfRequested covers the server.yaml trigger: migrate_config: true
// migrates once and comments the option back out; a second startup is a no-op.
func TestMigrateIfRequested(t *testing.T) {
	tmpDir := t.TempDir()
	writeLegacyFixture(t, tmpDir, `server:
  host: "0.0.0.0"
  port: 4594
  migrate_config: true
  timeout: 15
`)

	ran, status, err := MigrateIfRequested(tmpDir)
	if err != nil {
		t.Fatalf("MigrateIfRequested failed: %v", err)
	}
	if !ran || !status.Repos {
		t.Fatalf("expected migration to run, ran=%v status=%+v", ran, status)
	}

	// The option must now be commented out in server.yaml, other keys intact.
	data, err := os.ReadFile(filepath.Join(tmpDir, "server.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "\n  migrate_config: true") {
		t.Errorf("migrate_config still active after migration:\n%s", text)
	}
	if !strings.Contains(text, "# migrate_config: true") {
		t.Errorf("migrate_config was not left as a comment:\n%s", text)
	}
	if !strings.Contains(text, "port: 4594") || !strings.Contains(text, "timeout: 15") {
		t.Errorf("neighboring keys lost:\n%s", text)
	}

	// Second startup: option is commented → no-op.
	ran2, status2, err := MigrateIfRequested(tmpDir)
	if err != nil {
		t.Fatalf("second MigrateIfRequested failed: %v", err)
	}
	if ran2 || status2.Any() {
		t.Errorf("second run should be a no-op, ran=%v status=%+v", ran2, status2)
	}
}

// TestMigrateIfRequestedNoOption verifies the default: without migrate_config
// (absent or false), NOTHING migrates — legacy files stay in place.
func TestMigrateIfRequestedNoOption(t *testing.T) {
	for name, serverYAML := range map[string]string{
		"absent": "server:\n  port: 4594\n",
		"false":  "server:\n  migrate_config: false\n",
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			writeLegacyFixture(t, tmpDir, serverYAML)
			ran, _, err := MigrateIfRequested(tmpDir)
			if err != nil {
				t.Fatal(err)
			}
			if ran {
				t.Error("migration ran without the explicit option")
			}
			if _, err := os.Stat(filepath.Join(tmpDir, "repos.yaml")); err != nil {
				t.Error("repos.yaml must stay untouched by default")
			}
			if !DetectLegacy(tmpDir).Repos {
				t.Error("legacy state must remain detectable")
			}
		})
	}
}

// TestSeedingSkips verifies that a legacy flat file keeps its split counterpart
// from being seeded (so the seeded examples never shadow user config).
func TestSeedingSkips(t *testing.T) {
	tmpDir := t.TempDir()
	writeLegacyFixture(t, tmpDir, "")

	skips := SeedingSkips(tmpDir)
	if !skips["patterns"] || !skips["events"] || !skips["templates"] {
		t.Fatalf("SeedingSkips = %v, want patterns/events/templates", skips)
	}

	// After migration, nothing is skipped anymore.
	if _, err := MigrateAll(tmpDir); err != nil {
		t.Fatal(err)
	}
	if skips := SeedingSkips(tmpDir); len(skips) != 0 {
		t.Fatalf("SeedingSkips after migration = %v, want empty", skips)
	}
}

// TestInjectWeightIdempotent verifies the defense-in-depth layer: even if
// injectWeight is called twice on the same node, it updates in place rather
// than stacking a second weight key.
func TestInjectWeightIdempotent(t *testing.T) {
	var root yaml.Node
	src := `
pattern: foo
events:
  push:
`
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		t.Fatal(err)
	}
	m := topMappingNode(&root)
	if m == nil {
		t.Fatal("topMappingNode returned nil")
	}
	injectWeight(m, 5)
	injectWeight(m, 3) // must update in place, not stack

	out, err := marshalYAMLNode(m)
	if err != nil {
		t.Fatal(err)
	}
	if c := countWeightLines(out); c != 1 {
		t.Errorf("expected 1 weight line, got %d\n%s", c, out)
	}
	if !strings.Contains(string(out), "weight: 3") {
		t.Errorf("weight was not updated to 3\n%s", out)
	}
}
