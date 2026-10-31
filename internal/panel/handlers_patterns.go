package panel

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"gopkg.in/yaml.v3"
)

// handlePatterns lists all pattern rules.
func (a *App) handlePatterns(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	data.PatternsLegacy = a.patternsLegacy()
	if cfg, err := a.loadConfig(); err == nil {
		for i, rp := range cfg.Repos.Repos {
			data.Patterns = append(data.Patterns, patternListRow(i, rp))
		}
	}
	a.renderPage(w, "patterns", data)
}

// handlePatternNew renders a blank edit form for a new pattern rule.
func (a *App) handlePatternNew(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	data.PatternsLegacy = a.patternsLegacy()
	data.EditPattern = PatternRow{Index: -1}
	a.renderPage(w, "pattern_edit", data)
}

// handlePatternEdit renders the edit form for an existing pattern rule by index.
func (a *App) handlePatternEdit(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	idx, _ := strconv.Atoi(r.URL.Query().Get("index"))
	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.configLoadFailed"), "err")
		return
	}
	if idx < 0 || idx >= len(cfg.Repos.Repos) {
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.patternNotFound"), "err")
		return
	}
	data.EditPattern = patternEditRow(idx, cfg.Repos.Repos[idx])
	data.PatternsLegacy = a.patternsLegacy()
	a.renderPage(w, "pattern_edit", data)
}

// handlePatternSave creates or updates a pattern rule.
func (a *App) handlePatternSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.invalidForm"), "err")
		return
	}
	idx, _ := strconv.Atoi(r.FormValue("index"))
	pattern := strings.TrimSpace(r.FormValue("pattern"))
	if pattern == "" {
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.patternRequired"), "err")
		return
	}
	weight, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("weight")))

	events, err := parseEventsYAML(r.FormValue("events"))
	if err != nil {
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.eventsParseFailed", err), "err")
		return
	}
	notifyTo := splitLines(r.FormValue("notify_to"))
	secret := strings.TrimSpace(r.FormValue("secret"))

	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.configLoadFailed"), "err")
		return
	}

	rp := config.RepoPattern{Weight: weight, Pattern: pattern, Events: events, NotifyTo: notifyTo, Secret: secret}

	// Legacy repos.yaml is authoritative until the user migrates: splice the
	// rule back into the flat file (comments preserved) instead of writing a
	// split file that would shadow it.
	if a.patternsLegacy() {
		if err := a.savePatternLegacy(idx, rp); err != nil {
			a.redirectFlash(w, r, "/patterns", a.message(r, "flash.saveFailed", err), "err")
			return
		}
		a.notifySaved()
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.patternSaved"), "ok")
		return
	}

	patternsDir := filepath.Join(a.cfgDir, "patterns")
	_ = os.MkdirAll(patternsDir, 0o755)

	// If patterns/ is empty (e.g. a legacy install whose split migration did not
	// populate the directory), bootstrap it from the full in-memory list first so
	// the other rules are not dropped when we write only the edited rule.
	if !patternDirHasFiles(patternsDir) {
		for _, p := range cfg.Repos.Repos {
			_ = SaveYAML(patternFilePath(patternsDir, p.Pattern), p)
		}
	}

	// Editing an existing rule: remove its old file in case the pattern string
	// changed (so the new file uses the new name and no orphan remains).
	if idx >= 0 && idx < len(cfg.Repos.Repos) {
		old := cfg.Repos.Repos[idx]
		_ = os.Remove(patternFilePath(patternsDir, old.Pattern))
	}

	if err := SaveYAML(patternFilePath(patternsDir, pattern), rp); err != nil {
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	a.notifySaved()
	a.redirectFlash(w, r, "/patterns", a.message(r, "flash.patternSaved"), "ok")
}

// handlePatternDelete removes a pattern rule by index.
func (a *App) handlePatternDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.invalidForm"), "err")
		return
	}
	idx, _ := strconv.Atoi(r.FormValue("index"))
	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.configLoadFailed"), "err")
		return
	}
	if idx < 0 || idx >= len(cfg.Repos.Repos) {
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.patternNotFound"), "err")
		return
	}
	target := cfg.Repos.Repos[idx]

	// Legacy repos.yaml: remove the entry from the flat file in place.
	if a.patternsLegacy() {
		if err := a.deletePatternLegacy(idx); err != nil {
			a.redirectFlash(w, r, "/patterns", a.message(r, "flash.saveFailed", err), "err")
			return
		}
		a.notifySaved()
		a.redirectFlash(w, r, "/patterns", a.message(r, "flash.patternDeleted"), "ok")
		return
	}

	patternsDir := filepath.Join(a.cfgDir, "patterns")
	_ = os.Remove(patternFilePath(patternsDir, target.Pattern))

	a.notifySaved()
	a.redirectFlash(w, r, "/patterns", a.message(r, "flash.patternDeleted"), "ok")
}

// patternListRow builds a PatternRow for list display.
func patternListRow(i int, rp config.RepoPattern) PatternRow {
	return PatternRow{
		Index:      i,
		Weight:     rp.Weight,
		Pattern:    rp.Pattern,
		NotifyTo:   rp.NotifyTo,
		EventCount: len(rp.Events),
		HasSecret:  rp.Secret != "",
	}
}

// patternEditRow builds a PatternRow for the edit form (with raw textarea contents).
func patternEditRow(i int, rp config.RepoPattern) PatternRow {
	row := PatternRow{
		Index:       i,
		Weight:      rp.Weight,
		Pattern:     rp.Pattern,
		NotifyTo:    rp.NotifyTo,
		NotifyToRaw: strings.Join(rp.NotifyTo, "\n"),
		Events:      rp.Events,
		EventCount:  len(rp.Events),
		Secret:      rp.Secret,
		HasSecret:   rp.Secret != "",
	}
	if len(rp.Events) > 0 {
		if b, err := yaml.Marshal(rp.Events); err == nil {
			row.EventsYAML = strings.TrimRight(string(b), "\n")
		}
	}
	return row
}

// parseEventsYAML parses the events textarea into a map[string]any.
func parseEventsYAML(text string) (map[string]any, error) {
	text = strings.TrimSpace(text)
	out := map[string]any{}
	if text == "" {
		return out, nil
	}
	if err := yaml.Unmarshal([]byte(text), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// splitLines splits a textarea into trimmed, non-empty lines.
func splitLines(s string) []string {
	var res []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			res = append(res, line)
		}
	}
	return res
}

// patternFilePath returns the per-item file path for a pattern, using the same
// sanitization rules as the migration so panel edits target the right file.
func patternFilePath(patternsDir, pattern string) string {
	return filepath.Join(patternsDir, config.SanitizeFilename(pattern)+".yaml")
}

// patternDirHasFiles reports whether the patterns/ directory already holds any
// split-format pattern files.
func patternDirHasFiles(patternsDir string) bool {
	entries, err := os.ReadDir(patternsDir)
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

// patternsLegacy reports whether pattern rules are still served from the
// legacy flat repos.yaml (file present AND patterns/ empty — the same rule the
// config loader uses, so panel writes always target the active format).
func (a *App) patternsLegacy() bool {
	if _, err := os.Stat(filepath.Join(a.cfgDir, "repos.yaml")); err != nil {
		return false
	}
	return !patternDirHasFiles(filepath.Join(a.cfgDir, "patterns"))
}

// reposSequence finds (or creates) the `repos:` sequence inside a decoded
// repos.yaml document, creating the top mapping and key when absent.
func reposSequence(root *yaml.Node) *yaml.Node {
	m := topMap(root)
	if m == nil {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == "repos" && m.Content[i+1].Kind == yaml.SequenceNode {
			return m.Content[i+1]
		}
	}
	// No repos key yet — create one.
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "repos"}
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	m.Content = append(m.Content, key, seq)
	return seq
}

// patternNode marshals a RepoPattern into a standalone mapping node.
// Weight is written only when non-zero so legacy repos.yaml entries stay
// weight-free by default.
func patternNode(rp config.RepoPattern) (*yaml.Node, error) {
	type plain struct {
		Weight   int            `yaml:"weight,omitempty"`
		Pattern  string         `yaml:"pattern"`
		Events   map[string]any `yaml:"events"`
		NotifyTo []string       `yaml:"notify_to"`
		Secret   string         `yaml:"secret,omitempty"`
	}
	b, err := yaml.Marshal(plain(rp))
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	return topMap(&doc), nil
}

// savePatternLegacy writes one pattern rule back into the legacy repos.yaml,
// splicing the item node in place so comments on all other entries (and on
// the file at large) survive the edit.
func (a *App) savePatternLegacy(idx int, rp config.RepoPattern) error {
	path := filepath.Join(a.cfgDir, "repos.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}
	seq := reposSequence(&root)
	if seq == nil {
		return fmt.Errorf("repos.yaml: cannot locate repos sequence")
	}

	item, err := patternNode(rp)
	if err != nil {
		return err
	}
	if idx >= 0 && idx < len(seq.Content) {
		// Preserve the replaced entry's comments on the new node.
		old := seq.Content[idx]
		item.HeadComment, item.FootComment, item.LineComment = old.HeadComment, old.FootComment, old.LineComment
		seq.Content[idx] = item
	} else {
		seq.Content = append(seq.Content, item)
	}
	return atomicWriteFile(path, marshalYAMLDoc(&root), 0o644)
}

// deletePatternLegacy removes one pattern rule from the legacy repos.yaml in
// place (comments on other entries preserved).
func (a *App) deletePatternLegacy(idx int) error {
	path := filepath.Join(a.cfgDir, "repos.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}
	seq := reposSequence(&root)
	if seq == nil || idx < 0 || idx >= len(seq.Content) {
		return fmt.Errorf("repos.yaml: rule index out of range")
	}
	seq.Content = append(seq.Content[:idx], seq.Content[idx+1:]...)
	return atomicWriteFile(path, marshalYAMLDoc(&root), 0o644)
}

// marshalYAMLDoc encodes a yaml.Node document with 2-space indent, stripping
// cosmetic ": null" tails (same normalization as the migration writer).
func marshalYAMLDoc(root *yaml.Node) []byte {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil
	}
	enc.Close()
	return []byte(strings.ReplaceAll(buf.String(), ": null\n", ":\n"))
}
