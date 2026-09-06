package panel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

// TestSavePatternLegacyPreservesComments verifies that editing a rule through
// the panel while the legacy repos.yaml is active splices only that entry and
// leaves comments on every other line intact.
func TestSavePatternLegacyPreservesComments(t *testing.T) {
	dir := t.TempDir()
	original := `# top comment about the whole file
repos:
  # aim project rule
  - pattern: "AIMEtherCAT/*"
    events:
      basic:
    notify_to:
      - aim-ecat
  - pattern: "org/repo"   # inline comment
    events:
      push:
    notify_to:
      - dev-team
`
	if err := os.WriteFile(filepath.Join(dir, "repos.yaml"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{cfgDir: dir}
	if !a.patternsLegacy() {
		t.Fatal("patternsLegacy should be true with repos.yaml and no patterns/")
	}

	// Edit the SECOND rule (index 1) with a new target.
	rp := config.RepoPattern{
		Pattern:  "org/repo",
		Events:   map[string]any{"push": nil},
		NotifyTo: []string{"ops-team"},
	}
	if err := a.savePatternLegacy(1, rp); err != nil {
		t.Fatalf("savePatternLegacy failed: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(dir, "repos.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	// File-level and entry-level head comments survive; the replaced entry's
	// own body (including its inline comments) is rewritten wholesale — that
	// is the documented contract for splicing.
	for _, want := range []string{
		"# top comment about the whole file",
		"# aim project rule",
		"- pattern: \"AIMEtherCAT/*\"",
		"- aim-ecat",
		"- ops-team",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("rewritten repos.yaml lost %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "weight") {
		t.Errorf("legacy write-back must not inject weight:\n%s", text)
	}
	if strings.Contains(text, "- dev-team") {
		t.Errorf("replaced entry's old notify_to should be gone:\n%s", text)
	}

	// Append a NEW rule (index -1) and delete one.
	newRule := config.RepoPattern{Pattern: "foo/bar", Events: map[string]any{"all": nil}, NotifyTo: []string{"x"}}
	if err := a.savePatternLegacy(-1, newRule); err != nil {
		t.Fatal(err)
	}
	if err := a.deletePatternLegacy(0); err != nil {
		t.Fatal(err)
	}
	out2, err := os.ReadFile(filepath.Join(dir, "repos.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out2), "AIMEtherCAT") {
		t.Errorf("deletePatternLegacy did not remove entry 0:\n%s", out2)
	}
	if !strings.Contains(string(out2), "foo/bar") {
		t.Errorf("appended rule missing:\n%s", out2)
	}
}

// TestTemplatesLegacyWriteBack verifies that saving one event while a legacy
// templates.jsonc is active merges into the flat file instead of creating a
// split file that would shadow the rest.
func TestTemplatesLegacyWriteBack(t *testing.T) {
	dir := t.TempDir()
	original := `{
  // my templates
  "templates": {
    "push": {"payloads": [{"tags": ["default"], "payload": {"msg_type": "text"}}]},
    "issues": {"payloads": [{"tags": ["default"], "payload": {"msg_type": "text"}}]}
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "templates.jsonc"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{cfgDir: dir}
	if got := a.templatesLegacyPath("default"); got == "" {
		t.Fatal("templatesLegacyPath should detect templates.jsonc")
	}

	if err := a.saveTemplateLegacy(filepath.Join(dir, "templates.jsonc"), "release",
		[]any{map[string]any{"tags": []string{"default"}, "payload": map[string]any{"msg_type": "text"}}}); err != nil {
		t.Fatalf("saveTemplateLegacy failed: %v", err)
	}

	// No split file may appear while legacy is active.
	if entries, err := os.ReadDir(filepath.Join(dir, "templates")); err == nil && len(entries) > 0 {
		t.Fatalf("split template dir must stay empty in legacy mode, got %v", entries)
	}

	out, err := os.ReadFile(filepath.Join(dir, "templates.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{`"push"`, `"issues"`, `"release"`} {
		if !strings.Contains(text, want) {
			t.Errorf("merged templates.jsonc lost %s:\n%s", want, text)
		}
	}

	// A split dir with content wins over the legacy file.
	splitDir := filepath.Join(dir, "templates", "default")
	if err := os.MkdirAll(splitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(splitDir, "push.json"), []byte(`{"payloads": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := a.templatesLegacyPath("default"); got != "" {
		t.Errorf("templatesLegacyPath should yield to the split dir, got %q", got)
	}
}
