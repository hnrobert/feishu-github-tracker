package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Migrate detects legacy flat-file configs in the config root and migrates
// them to the new per-item subdirectory layout. Old files are ALWAYS moved
// to legacy/ (comments preserved) and split into per-item files, regardless
// of whether the target subdirectory already exists (e.g. from
// initializeConfigDir seeding example defaults). This ensures the user's
// actual configuration is never ignored.
func Migrate(configDir string) error {
	legacyDir := filepath.Join(configDir, "legacy")

	// repos.yaml → patterns/*.yaml
	if fileExists(filepath.Join(configDir, "repos.yaml")) {
		if err := ensureDir(legacyDir); err != nil {
			return err
		}
		if err := migrateRepos(configDir, legacyDir); err != nil {
			return fmt.Errorf("migrate repos: %w", err)
		}
	}

	// events.yaml → events/event_sets/*.yaml + events/definitions/*.yaml
	if fileExists(filepath.Join(configDir, "events.yaml")) {
		if err := ensureDir(legacyDir); err != nil {
			return err
		}
		if err := migrateEvents(configDir, legacyDir); err != nil {
			return fmt.Errorf("migrate events: %w", err)
		}
	}

	// templates.jsonc + templates.*.jsonc → templates/<locale>/*.json
	if fileExists(filepath.Join(configDir, "templates.jsonc")) {
		if err := ensureDir(legacyDir); err != nil {
			return err
		}
		if err := migrateTemplates(configDir, legacyDir); err != nil {
			return fmt.Errorf("migrate templates: %w", err)
		}
	}

	return nil
}

// ── Per-type migrations (using yaml.Node to preserve comments) ──

func migrateRepos(configDir, legacyDir string) error {
	src := filepath.Join(configDir, "repos.yaml")
	dstDir := filepath.Join(configDir, "patterns")

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Parse into yaml.Node tree — preserves HeadComment/LineComment/FootComment
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}

	// Navigate: document → top mapping → "repos" key → sequence
	seqNode := findSequenceByKey(root, "repos")
	if seqNode == nil {
		// No repos found — just move the file
		return os.Rename(src, filepath.Join(legacyDir, "repos.yaml"))
	}

	if err := ensureDir(dstDir); err != nil {
		return err
	}

	total := len(seqNode.Content)
	for i, item := range seqNode.Content {
		// Assign weight: first rule (index 0) gets highest weight (total-1),
		// last rule gets weight 0. This preserves the original evaluation order
		// where the first match wins, and the catch-all "*" at the end gets
		// the lowest priority (weight 0).
		weight := total - 1 - i
		injectWeight(item, weight)

		out, err := marshalYAMLNode(item)
		if err != nil {
			return fmt.Errorf("marshal repo %d: %w", i, err)
		}

		// Filename is just the sanitized pattern name (no index prefix)
		name := nodePattern(item)
		if name == "" {
			name = fmt.Sprintf("rule-%d", i)
		}
		fname := sanitizeFilename(name) + ".yaml"
		if err := os.WriteFile(filepath.Join(dstDir, fname), out, 0o644); err != nil {
			return err
		}
	}

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

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}

	if err := ensureDir(esDir); err != nil {
		return err
	}
	if err := ensureDir(defDir); err != nil {
		return err
	}

	// event_sets: top mapping → "event_sets" key → mapping of setName → body
	if esMap := findMappingByKey(root, "event_sets"); esMap != nil {
		for i := 0; i+1 < len(esMap.Content); i += 2 {
			keyNode := esMap.Content[i]
			valNode := esMap.Content[i+1]
			// The comment above the key (e.g., "# custom template...") is
			// stored on keyNode.HeadComment. Move it to valNode so it's
			// preserved when we marshal the value as a standalone file.
			if keyNode.HeadComment != "" && valNode.HeadComment == "" {
				valNode.HeadComment = keyNode.HeadComment
			}
			out, err := marshalYAMLNode(valNode)
			if err != nil {
				continue
			}
			fname := sanitizeFilename(keyNode.Value) + ".yaml"
			if err := os.WriteFile(filepath.Join(esDir, fname), out, 0o644); err != nil {
				return err
			}
		}
	}

	// events: top mapping → "events" key → mapping of eventName → body
	if evMap := findMappingByKey(root, "events"); evMap != nil {
		for i := 0; i+1 < len(evMap.Content); i += 2 {
			keyNode := evMap.Content[i]
			valNode := evMap.Content[i+1]
			if keyNode.HeadComment != "" && valNode.HeadComment == "" {
				valNode.HeadComment = keyNode.HeadComment
			}
			out, err := marshalYAMLNode(valNode)
			if err != nil {
				continue
			}
			fname := sanitizeFilename(keyNode.Value) + ".yaml"
			if err := os.WriteFile(filepath.Join(defDir, fname), out, 0o644); err != nil {
				return err
			}
		}
	}

	return os.Rename(src, filepath.Join(legacyDir, "events.yaml"))
}

// Templates are JSON/JSONC — comments cannot be preserved in JSON output.
// Originals are kept intact in legacy/ for reference.
func migrateTemplates(configDir, legacyDir string) error {
	templatesDir := filepath.Join(configDir, "templates")

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

		baseName := filepath.Base(tf.path)
		if err := os.Rename(tf.path, filepath.Join(legacyDir, baseName)); err != nil {
			return err
		}
	}

	return nil
}

// ── yaml.Node navigation helpers ──

// findSequenceByKey drills into a document node → top mapping → key → sequence.
func findSequenceByKey(root yaml.Node, key string) *yaml.Node {
	m := topMappingNode(&root)
	if m == nil {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key && m.Content[i+1].Kind == yaml.SequenceNode {
			return m.Content[i+1]
		}
	}
	return nil
}

// findMappingByKey drills into a document node → top mapping → key → mapping.
func findMappingByKey(root yaml.Node, key string) *yaml.Node {
	m := topMappingNode(&root)
	if m == nil {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key && m.Content[i+1].Kind == yaml.MappingNode {
			return m.Content[i+1]
		}
	}
	return nil
}

func topMappingNode(root *yaml.Node) *yaml.Node {
	if root == nil {
		return nil
	}
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind == yaml.MappingNode {
		return root
	}
	return nil
}

// nodePattern extracts the "pattern" value from a repo mapping node.
func nodePattern(n *yaml.Node) string {
	if n == nil || n.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == "pattern" {
			return n.Content[i+1].Value
		}
	}
	return ""
}

// marshalYAMLNode encodes a yaml.Node with 2-space indent, preserving all
// attached comments (HeadComment, LineComment, FootComment). Also strips
// trailing ": null" → ":" for cosmetic compatibility (Go's yaml.v3 writes
// `key: null` for nil values; the original files used `key:`).
func marshalYAMLNode(n *yaml.Node) ([]byte, error) {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(n); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	// Strip ": null\n" → ":\n" for cosmetic compatibility
	out := strings.ReplaceAll(buf.String(), ": null\n", ":\n")
	return []byte(out), nil
}

// injectWeight inserts a `weight: N` key-value pair at the beginning of a
// mapping node, preserving any existing HeadComment on the node.
func injectWeight(mappingNode *yaml.Node, weight int) {
	if mappingNode == nil || mappingNode.Kind != yaml.MappingNode {
		return
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "", Value: "weight"}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "", Value: strconv.Itoa(weight)}
	// Prepend key+value to the mapping's Content
	mappingNode.Content = append([]*yaml.Node{keyNode, valNode}, mappingNode.Content...)
}

// ── Generic helpers ──

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "*", "all")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, " ", "_")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
