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

// ── Explicit, opt-in migration ──
//
// Migration NEVER happens automatically at startup. Legacy flat files keep
// working forever (the loaders fall back to them); a user converts to the
// split per-item layout only by:
//   1. pressing "Migrate" in the web panel (internal/panel settings page), or
//   2. setting `migrate_config: true` under `server:` in server.yaml.
//
// The config-file path runs MigrateIfRequested at startup and comments the
// option back out once the migration is done (mirroring how the panel
// normalizes panel.password).

// LegacyStatus describes which legacy flat files are still present.
type LegacyStatus struct {
	Repos     bool     // repos.yaml
	Events    bool     // events.yaml
	Templates bool     // templates.jsonc and/or templates.<locale>.jsonc
	Files     []string // file names for display
}

func (s LegacyStatus) Any() bool { return s.Repos || s.Events || s.Templates }

// DetectLegacy scans the config root for unmigrated legacy flat files.
// A type counts as legacy when its flat file exists AND its split
// subdirectory is empty — exactly the condition under which the loaders
// would serve the flat file, so this matches the active format.
func DetectLegacy(configDir string) LegacyStatus {
	var st LegacyStatus
	if fileExists(filepath.Join(configDir, "repos.yaml")) && !dirHasYAML(filepath.Join(configDir, "patterns")) {
		st.Repos = true
		st.Files = append(st.Files, "repos.yaml")
	}
	if fileExists(filepath.Join(configDir, "events.yaml")) &&
		!dirHasYAML(filepath.Join(configDir, "events", "event_sets")) &&
		!dirHasYAML(filepath.Join(configDir, "events", "definitions")) {
		st.Events = true
		st.Files = append(st.Files, "events.yaml")
	}
	if templatesFlat := listLegacyTemplates(configDir); len(templatesFlat) > 0 && !templatesDirHasContent(configDir) {
		st.Templates = true
		st.Files = append(st.Files, templatesFlat...)
	}
	return st
}

// MigrateAll runs every pending legacy→split migration (explicit user action:
// panel button or server.migrate_config). Each legacy file is moved to
// legacy/ (comments preserved) and split into per-item files; patterns get a
// weight injected per their original order so evaluation order is unchanged.
func MigrateAll(configDir string) (LegacyStatus, error) {
	st := DetectLegacy(configDir)
	if !st.Any() {
		return st, nil
	}
	legacyDir := filepath.Join(configDir, "legacy")
	if err := ensureDir(legacyDir); err != nil {
		return st, err
	}
	if st.Repos {
		if err := migrateRepos(configDir, legacyDir); err != nil {
			return st, fmt.Errorf("migrate repos: %w", err)
		}
	}
	if st.Events {
		if err := migrateEvents(configDir, legacyDir); err != nil {
			return st, fmt.Errorf("migrate events: %w", err)
		}
	}
	if st.Templates {
		if err := migrateTemplates(configDir, legacyDir); err != nil {
			return st, fmt.Errorf("migrate templates: %w", err)
		}
	}
	return st, nil
}

// MigrateIfRequested implements the server.yaml trigger: when `server:
// migrate_config: true` is set, MigrateAll runs and the option is commented
// back out (so it fires exactly once). Returns whether a migration ran and
// what it migrated. A missing server.yaml is a plain no-op.
func MigrateIfRequested(configDir string) (ran bool, status LegacyStatus, err error) {
	serverPath := filepath.Join(configDir, "server.yaml")
	data, readErr := os.ReadFile(serverPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return false, LegacyStatus{}, nil
		}
		return false, LegacyStatus{}, readErr
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return false, LegacyStatus{}, fmt.Errorf("parse server.yaml: %w", err)
	}
	serverMap := findMappingByKey(root, "server")
	if serverMap == nil {
		return false, LegacyStatus{}, nil
	}
	keyIdx := findKeyIndex(serverMap, "migrate_config")
	if keyIdx < 0 || !isTrueScalar(serverMap.Content[keyIdx+1]) {
		return false, LegacyStatus{}, nil
	}

	status, err = MigrateAll(configDir)
	if err != nil {
		return true, status, err
	}

	// Comment the option back out regardless of whether anything was left to
	// migrate: the user asked once, it is done (or nothing to do).
	commentOutKey(serverMap, keyIdx)
	if err := writeYAMLNodeFile(serverPath, &root); err != nil {
		return true, status, err
	}
	return true, status, nil
}

// SeedingSkips returns the example-configs subdirectories that must NOT be
// seeded into configDir because the corresponding legacy flat file is
// authoritative there (seeding patterns/ next to a user's repos.yaml would
// let the split dir silently shadow the user's config).
func SeedingSkips(configDir string) map[string]bool {
	st := DetectLegacy(configDir)
	skips := map[string]bool{}
	if st.Repos {
		skips["patterns"] = true
	}
	if st.Events {
		skips["events"] = true
	}
	if st.Templates {
		skips["templates"] = true
	}
	return skips
}

// ── yaml key helpers (server.migrate_config handling) ──

// findKeyIndex returns the content index of the key node for key, or -1.
func findKeyIndex(m *yaml.Node, key string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func isTrueScalar(n *yaml.Node) bool {
	switch strings.ToLower(strings.TrimSpace(n.Value)) {
	case "true", "yes", "on":
		return n.Tag == "!!bool" || n.Tag == "" || n.Tag == "!!str"
	}
	return false
}

// commentOutKey removes the key/value pair at keyIdx from the mapping and
// leaves a rendered comment in its place, attached to the next remaining key
// (yaml.v3 renders a pair's leading comment from its key node; when the pair
// was last, the comment becomes the previous key's foot comment instead).
func commentOutKey(m *yaml.Node, keyIdx int) {
	if m == nil || keyIdx < 0 || keyIdx+1 >= len(m.Content) {
		return
	}
	keyNode, valueNode := m.Content[keyIdx], m.Content[keyIdx+1]

	line := "migrate_config: true  # 迁移完成，已自动注释；如需再次触发请取消注释"
	for _, extra := range []string{valueNode.HeadComment, keyNode.HeadComment} {
		if extra != "" {
			line = extra + "\n" + line
		}
	}

	rest := append(m.Content[:keyIdx:keyIdx], m.Content[keyIdx+2:]...)
	if keyIdx < len(rest) {
		next := rest[keyIdx]
		if next.HeadComment != "" {
			line = line + "\n" + next.HeadComment
		}
		next.HeadComment = line
	} else if len(rest) > 0 {
		last := rest[len(rest)-1]
		if last.FootComment != "" {
			line = last.FootComment + "\n" + line
		}
		last.FootComment = line
	}
	m.Content = rest
}

// writeYAMLNodeFile writes a yaml.Node document with 2-space indent atomically.
func writeYAMLNodeFile(path string, root *yaml.Node) error {
	out, err := marshalYAMLNode(root)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
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
		return moveToLegacy(src, legacyDir)
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
		fname := SanitizeFilename(name) + ".yaml"
		if err := os.WriteFile(filepath.Join(dstDir, fname), out, 0o644); err != nil {
			return err
		}
	}

	return moveToLegacy(src, legacyDir)
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
			fname := SanitizeFilename(keyNode.Value) + ".yaml"
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
			fname := SanitizeFilename(keyNode.Value) + ".yaml"
			if err := os.WriteFile(filepath.Join(defDir, fname), out, 0o644); err != nil {
				return err
			}
		}
	}

	return moveToLegacy(src, legacyDir)
}

// Templates are JSON/JSONC — comments cannot be preserved in JSON output.
// Originals are kept intact in legacy/ for reference.

var legacyTemplateRe = regexp.MustCompile(`^templates\.([a-zA-Z0-9_-]+)\.jsonc$`)

// listLegacyTemplates returns the legacy template file names present in the
// config root (templates.jsonc plus any templates.<locale>.jsonc).
func listLegacyTemplates(configDir string) []string {
	var names []string
	if fileExists(filepath.Join(configDir, "templates.jsonc")) {
		names = append(names, "templates.jsonc")
	}
	entries, _ := os.ReadDir(configDir)
	for _, e := range entries {
		if !e.IsDir() && legacyTemplateRe.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

func migrateTemplates(configDir, legacyDir string) error {
	templatesDir := filepath.Join(configDir, "templates")

	type tmplFile struct {
		path   string
		locale string
	}
	var files []tmplFile

	for _, name := range listLegacyTemplates(configDir) {
		locale := "default"
		if m := legacyTemplateRe.FindStringSubmatch(name); len(m) > 1 {
			locale = m[1]
		}
		files = append(files, tmplFile{filepath.Join(configDir, name), locale})
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
			fname := SanitizeFilename(event) + ".json"
			if err := os.WriteFile(filepath.Join(localeDir, fname), append(out, '\n'), 0o644); err != nil {
				return err
			}
		}

		if err := moveToLegacy(tf.path, legacyDir); err != nil {
			return err
		}
	}

	return nil
}

// moveToLegacy moves src into legacyDir keeping its base name. If the target
// already exists (e.g. from an earlier migration round), a numeric suffix is
// added instead of overwriting the previous backup.
func moveToLegacy(src, legacyDir string) error {
	base := filepath.Base(src)
	dst := filepath.Join(legacyDir, base)
	if _, err := os.Stat(dst); err == nil {
		ext := filepath.Ext(base)
		stem := strings.TrimSuffix(base, ext)
		for i := 1; ; i++ {
			dst = filepath.Join(legacyDir, fmt.Sprintf("%s-%d%s", stem, i, ext))
			if _, err := os.Stat(dst); os.IsNotExist(err) {
				break
			}
		}
	}
	return os.Rename(src, dst)
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

// injectWeight ensures a mapping node has exactly one `weight: N` entry with
// the given value. If a weight key already exists (e.g. the source file was
// already migrated and re-processed), its value is updated in place rather
// than a second key being prepended — this keeps the migration idempotent.
func injectWeight(mappingNode *yaml.Node, weight int) {
	if mappingNode == nil || mappingNode.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(mappingNode.Content); i += 2 {
		if mappingNode.Content[i].Value == "weight" {
			mappingNode.Content[i+1].Value = strconv.Itoa(weight)
			return
		}
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

// dirHasYAML reports whether dir contains at least one .yaml file (non-recursive).
func dirHasYAML(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			return true
		}
	}
	return false
}

// templatesDirHasContent reports whether the templates/ locale layout already
// holds any per-event template files (split format adopted).
func templatesDirHasContent(configDir string) bool {
	templatesDir := filepath.Join(configDir, "templates")
	locales, err := os.ReadDir(templatesDir)
	if err != nil {
		return false
	}
	for _, locale := range locales {
		if !locale.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(templatesDir, locale.Name()))
		if err != nil {
			continue
		}
		for _, f := range files {
			if !f.IsDir() && (strings.HasSuffix(f.Name(), ".json") || strings.HasSuffix(f.Name(), ".jsonc")) {
				return true
			}
		}
	}
	return false
}

// SanitizeFilename converts a pattern or event name into a safe filename stem
// (no extension). Shared by migration and the panel so they target the same
// per-item files.
func SanitizeFilename(s string) string {
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
