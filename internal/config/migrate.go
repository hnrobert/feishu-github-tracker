package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Migrate detects legacy flat-file configs and migrates them to the new
// per-item subdirectory layout. Old files are moved intact to legacy/ (all
// comments preserved) and split into clean per-item files in the new dirs.
// If the new dirs already exist with content, migration for that type is
// skipped. This is a no-op when no legacy files are found.
func Migrate(configDir string) error {
	legacyDir := filepath.Join(configDir, "legacy")
	migrated := false

	// repos.yaml → patterns/*.yaml
	if fileExists(filepath.Join(configDir, "repos.yaml")) && !dirHasYAML(filepath.Join(configDir, "patterns")) {
		if err := ensureDir(legacyDir); err != nil {
			return err
		}
		if err := migrateRepos(configDir, legacyDir); err != nil {
			return fmt.Errorf("migrate repos: %w", err)
		}
		migrated = true
	}

	// events.yaml → events/event_sets/*.yaml + events/definitions/*.yaml
	if fileExists(filepath.Join(configDir, "events.yaml")) && !dirHasYAML(filepath.Join(configDir, "events")) {
		if err := ensureDir(legacyDir); err != nil {
			return err
		}
		if err := migrateEvents(configDir, legacyDir); err != nil {
			return fmt.Errorf("migrate events: %w", err)
		}
		migrated = true
	}

	// templates.jsonc + templates.*.jsonc → templates/<locale>/*.json
	if fileExists(filepath.Join(configDir, "templates.jsonc")) && !dirHasContent(filepath.Join(configDir, "templates")) {
		if err := ensureDir(legacyDir); err != nil {
			return err
		}
		if err := migrateTemplates(configDir, legacyDir); err != nil {
			return fmt.Errorf("migrate templates: %w", err)
		}
		migrated = true
	}

	_ = migrated
	return nil
}

// ── Per-type migrations ──

func migrateRepos(configDir, legacyDir string) error {
	src := filepath.Join(configDir, "repos.yaml")
	dstDir := filepath.Join(configDir, "patterns")

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	var rc struct {
		Repos []RepoPattern `yaml:"repos"`
	}
	if err := yaml.Unmarshal(data, &rc); err != nil {
		return err
	}

	if err := ensureDir(dstDir); err != nil {
		return err
	}

	for i, rp := range rc.Repos {
		name := sanitizeFilename(rp.Pattern)
		if name == "" {
			name = fmt.Sprintf("rule-%d", i)
		}
		fname := fmt.Sprintf("%02d-%s.yaml", i, name)
		out, _ := yaml.Marshal(rp)
		if err := os.WriteFile(filepath.Join(dstDir, fname), out, 0o644); err != nil {
			return err
		}
	}

	// Move original to legacy/
	return os.Rename(src, filepath.Join(legacyDir, "repos.yaml"))
}

func migrateEvents(configDir, legacyDir string) error {
	src := filepath.Join(configDir, "events.yaml")
	esDir := filepath.Join(configDir, "events", "event_sets")
	defDir := filepath.Join(configDir, "events", "definitions")

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	var ec EventsConfig
	if err := yaml.Unmarshal(data, &ec); err != nil {
		return err
	}

	if err := ensureDir(esDir); err != nil {
		return err
	}
	if err := ensureDir(defDir); err != nil {
		return err
	}

	// event_sets
	for name, body := range ec.EventSets {
		fname := sanitizeFilename(name) + ".yaml"
		out, _ := yaml.Marshal(body)
		if err := os.WriteFile(filepath.Join(esDir, fname), out, 0o644); err != nil {
			return err
		}
	}

	// definitions
	for name, body := range ec.Events {
		fname := sanitizeFilename(name) + ".yaml"
		var out []byte
		if body != nil {
			out, _ = yaml.Marshal(body)
		} else {
			out = []byte("null\n")
		}
		if err := os.WriteFile(filepath.Join(defDir, fname), out, 0o644); err != nil {
			return err
		}
	}

	return os.Rename(src, filepath.Join(legacyDir, "events.yaml"))
}

func migrateTemplates(configDir, legacyDir string) error {
	templatesDir := filepath.Join(configDir, "templates")

	// Find all template files: templates.jsonc + templates.*.jsonc
	defaultPath := filepath.Join(configDir, "templates.jsonc")
	entries, _ := os.ReadDir(configDir)
	templateRe := regexp.MustCompile(`^templates\.([a-zA-Z0-9_-]+)\.jsonc$`)

	type tmplFile struct {
		path   string
		locale string
	}
	var files []tmplFile

	if fileExists(defaultPath) {
		files = append(files, tmplFile{defaultPath, "default"})
	}
	for _, e := range entries {
		if m := templateRe.FindStringSubmatch(e.Name()); len(m) > 1 {
			files = append(files, tmplFile{filepath.Join(configDir, e.Name()), m[1]})
		}
	}

	if len(files) == 0 {
		return nil
	}

	for _, tf := range files {
		data, err := os.ReadFile(tf.path)
		if err != nil {
			return err
		}
		cleaned := stripJSONCComments(string(data))
		var tc TemplatesConfig
		if err := json.Unmarshal([]byte(cleaned), &tc); err != nil {
			return fmt.Errorf("parse %s: %w", tf.path, err)
		}

		localeDir := filepath.Join(templatesDir, tf.locale)
		if err := ensureDir(localeDir); err != nil {
			return err
		}

		// Sort event names for deterministic output
		events := make([]string, 0, len(tc.Templates))
		for k := range tc.Templates {
			events = append(events, k)
		}
		sort.Strings(events)

		for _, event := range events {
			payloads := tc.Templates[event].Payloads
			out, _ := json.MarshalIndent(map[string]any{"payloads": payloads}, "", "  ")
			fname := sanitizeFilename(event) + ".json"
			if err := os.WriteFile(filepath.Join(localeDir, fname), append(out, '\n'), 0o644); err != nil {
				return err
			}
		}

		// Move original to legacy/
		baseName := filepath.Base(tf.path)
		if err := os.Rename(tf.path, filepath.Join(legacyDir, baseName)); err != nil {
			return err
		}
	}

	return nil
}

// ── Helpers ──

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirHasYAML(dir string) bool {
	files, err := scanYAMLDir(dir)
	return err == nil && len(files) > 0
}

func dirHasContent(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			if dirHasContent(filepath.Join(dir, e.Name())) {
				return true
			}
		} else {
			return true
		}
	}
	return false
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// sanitizeFilename converts a pattern/name to a safe filename component.
func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "*", "all")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, " ", "_")
	// keep only safe chars
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
