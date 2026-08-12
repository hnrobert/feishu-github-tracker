package panel

import (
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
