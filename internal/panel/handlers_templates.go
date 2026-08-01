package panel

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// templateDir returns the locale directory for templates.
// New layout: configs/templates/<locale>/
// Legacy fallback: configs/templates.<locale>.jsonc (handled by loadConfigFile).
func (a *App) templateLocaleDir(locale string) string {
	if locale == "" {
		locale = "default"
	}
	return filepath.Join(a.cfgDir, "templates", locale)
}

// handleTemplatesList lists every template locale with its event count.
func (a *App) handleTemplatesList(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	if cfg, err := a.loadConfig(); err == nil {
		for name, tc := range cfg.Templates {
			data.TemplateFilesList = append(data.TemplateFilesList, TemplateFileRow{
				Name:  name,
				Count: len(tc.Templates),
			})
		}
		sort.Slice(data.TemplateFilesList, func(i, j int) bool {
			return data.TemplateFilesList[i].Name < data.TemplateFilesList[j].Name
		})
	}
	a.renderPage(w, "templates_list", data)
}

// handleTemplateEdit shows the payloads for one event in one locale as
// editable JSON. Works with both new split layout (templates/<locale>/<event>.json)
// and legacy flat files (templates.jsonc).
func (a *App) handleTemplateEdit(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	event := r.URL.Query().Get("event")
	if file == "" {
		file = "default"
	}

	// Get the list of events for this locale from loaded config
	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/templates", a.message(r, "flash.templateLoadFailed", err), "err")
		return
	}
	tc := cfg.GetTemplateConfig(file)
	events := make([]string, 0, len(tc.Templates))
	for k := range tc.Templates {
		events = append(events, k)
	}
	sort.Strings(events)

	ed := EditTemplateData{File: file, Events: events, Event: event}
	if event != "" {
		// Read the single per-event file (new layout)
		eventPath := filepath.Join(a.templateLocaleDir(file), event+".json")
		var perEvent struct {
			Payloads []map[string]any `json:"payloads"`
		}
		if b, err := os.ReadFile(eventPath); err == nil {
			json.Unmarshal(b, &perEvent)
			ed.PayloadsJSON = prettyJSON(perEvent.Payloads)
		} else {
			// Legacy fallback: read from loaded config
			if et, ok := tc.Templates[event]; ok {
				ed.PayloadsJSON = prettyJSON(et.Payloads)
			} else {
				ed.PayloadsJSON = "[]"
			}
		}
	}
	data := a.baseData(r)
	data.EditTemplate = ed
	a.renderPage(w, "template_edit", data)
}

// handleTemplateSave writes one event's payloads to its per-event .json file.
func (a *App) handleTemplateSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/templates", a.message(r, "flash.invalidForm"), "err")
		return
	}
	file := r.FormValue("file")
	event := r.FormValue("event")
	text := r.FormValue("payloads_json")
	if event == "" {
		a.redirectFlash(w, r, "/templates", a.message(r, "flash.eventRequired"), "err")
		return
	}

	var payloads any
	if err := json.Unmarshal([]byte(text), &payloads); err != nil {
		a.redirectFlash(w, r, "/templates", a.message(r, "flash.payloadsParseFailed", err), "err")
		return
	}

	// Write to templates/<locale>/<event>.json
	dir := a.templateLocaleDir(file)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.redirectFlash(w, r, "/templates", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	path := filepath.Join(dir, event+".json")
	out, _ := json.MarshalIndent(map[string]any{"payloads": payloads}, "", "  ")
	if err := SaveJSON(path, map[string]any{"payloads": payloads}); err != nil {
		a.redirectFlash(w, r, "/templates", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	_ = out // SaveJSON handles the actual write
	a.notifySaved()
	a.redirectFlash(w, r, "/templates", a.message(r, "flash.templateSaved"), "ok")
}

// prettyJSON marshals v as indented JSON, returning "[]" on error.
func prettyJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(b)
}

// ensure these vars are referenced to avoid unused warnings
var _ = strings.TrimSpace
