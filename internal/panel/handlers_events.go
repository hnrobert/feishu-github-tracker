package panel

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// EventFileRow represents one editable events file.
type EventFileRow struct {
	Name    string // file name without extension
	Content string // raw file content (preserves comments)
	Section string // "event_sets" or "definitions"
}

// handleEvents shows event sets and definitions as per-file editors.
func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)

	// Try new split layout first
	esDir := filepath.Join(a.cfgDir, "events", "event_sets")
	defDir := filepath.Join(a.cfgDir, "events", "definitions")

	var rows []EventFileRow
	rows = append(rows, a.readEventFiles(esDir, "event_sets")...)
	rows = append(rows, a.readEventFiles(defDir, "definitions")...)

	if len(rows) == 0 {
		// Fallback: legacy single events.yaml
		if b, err := os.ReadFile(filepath.Join(a.cfgDir, "events.yaml")); err == nil {
			rows = append(rows, EventFileRow{
				Name:    "events",
				Content: string(b),
				Section: "legacy",
			})
		}
	}

	data.EventFiles = rows
	a.renderPage(w, "events", data)
}

func (a *App) readEventFiles(dir, section string) []EventFileRow {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var rows []EventFileRow
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		rows = append(rows, EventFileRow{
			Name:    name,
			Content: string(b),
			Section: section,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows
}

// handleEventsSave writes one event file (identified by section + name).
// Content is written verbatim (preserving comments). YAML is validated.
func (a *App) handleEventsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/events", a.message(r, "flash.invalidForm"), "err")
		return
	}

	section := r.FormValue("section")
	name := r.FormValue("name")
	content := r.FormValue("content")

	// Determine target directory
	var dir string
	switch section {
	case "event_sets":
		dir = filepath.Join(a.cfgDir, "events", "event_sets")
	case "definitions":
		dir = filepath.Join(a.cfgDir, "events", "definitions")
	case "legacy":
		// Legacy single-file mode
		if err := validateYAML(content); err != nil {
			a.redirectFlash(w, r, "/events", a.message(r, "flash.eventsParseFailed", err), "err")
			return
		}
		if err := os.WriteFile(filepath.Join(a.cfgDir, "events.yaml"), []byte(content), 0o644); err != nil {
			a.redirectFlash(w, r, "/events", a.message(r, "flash.saveFailed", err), "err")
			return
		}
		a.notifySaved()
		a.redirectFlash(w, r, "/events", a.message(r, "flash.eventsSaved"), "ok")
		return
	default:
		a.redirectFlash(w, r, "/events", "invalid section", "err")
		return
	}

	// Sanitize filename
	safe := sanitizeEventFilename(name)
	if safe == "" {
		a.redirectFlash(w, r, "/events", "invalid name", "err")
		return
	}

	// Validate YAML
	if err := validateYAML(content); err != nil {
		a.redirectFlash(w, r, "/events", a.message(r, "flash.eventsParseFailed", err), "err")
		return
	}

	// Write verbatim (preserves comments)
	path := filepath.Join(dir, safe+".yaml")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.redirectFlash(w, r, "/events", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		a.redirectFlash(w, r, "/events", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	a.notifySaved()
	a.redirectFlash(w, r, "/events", a.message(r, "flash.eventsSaved"), "ok")
}

// handleEventsCreateFile creates a new event_set or definition file.
func (a *App) handleEventsCreateFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/events", a.message(r, "flash.invalidForm"), "err")
		return
	}
	section := r.FormValue("section")
	name := sanitizeEventFilename(r.FormValue("name"))
	if name == "" {
		a.redirectFlash(w, r, "/events", "name required", "err")
		return
	}

	var dir string
	switch section {
	case "event_sets":
		dir = filepath.Join(a.cfgDir, "events", "event_sets")
	case "definitions":
		dir = filepath.Join(a.cfgDir, "events", "definitions")
	default:
		a.redirectFlash(w, r, "/events", "invalid section", "err")
		return
	}

	path := filepath.Join(dir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		a.redirectFlash(w, r, "/events", "file already exists", "err")
		return
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.redirectFlash(w, r, "/events", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		a.redirectFlash(w, r, "/events", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	a.notifySaved()
	a.redirectFlash(w, r, "/events", "file created", "ok")
}

// handleEventsDeleteFile deletes an event file.
func (a *App) handleEventsDeleteFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/events", a.message(r, "flash.invalidForm"), "err")
		return
	}
	section := r.FormValue("section")
	name := sanitizeEventFilename(r.FormValue("name"))

	var dir string
	switch section {
	case "event_sets":
		dir = filepath.Join(a.cfgDir, "events", "event_sets")
	case "definitions":
		dir = filepath.Join(a.cfgDir, "events", "definitions")
	default:
		a.redirectFlash(w, r, "/events", "invalid section", "err")
		return
	}

	path := filepath.Join(dir, name+".yaml")
	if err := os.Remove(path); err != nil {
		a.redirectFlash(w, r, "/events", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	a.notifySaved()
	a.redirectFlash(w, r, "/events", "file deleted", "ok")
}

func validateYAML(content string) error {
	var check map[string]any
	return yaml.Unmarshal([]byte(content), &check)
}

func sanitizeEventFilename(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
