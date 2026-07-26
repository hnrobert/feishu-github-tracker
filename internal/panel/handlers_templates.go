package panel

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
)

// templateFileName maps a logical template name to its on-disk file name.
// "default" -> templates.jsonc; anything else -> templates.<name>.jsonc.
func templateFileName(name string) string {
	if name == "" || name == "default" {
		return "templates.jsonc"
	}
	return "templates." + name + ".jsonc"
}

func (a *App) templateFilePath(name string) string {
	return filepath.Join(a.cfgDir, templateFileName(name))
}

// handleTemplatesList lists every templates.*.jsonc file with its event count.
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

// handleTemplateEdit shows the payloads for one event in one template file as
// editable JSON.
func (a *App) handleTemplateEdit(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	event := r.URL.Query().Get("event")
	path := a.templateFilePath(file)

	root := map[string]any{}
	if err := loadJSONC(path, &root); err != nil {
		a.redirectFlash(w, r, "/templates", a.message(r, "flash.templateLoadFailed", err), "err")
		return
	}

	events := eventKeys(root)
	ed := EditTemplateData{File: file, Events: events, Event: event}
	if event != "" {
		ed.PayloadsJSON = payloadsJSON(root, event)
	}
	data := a.baseData(r)
	data.EditTemplate = ed
	a.renderPage(w, "template_edit", data)
}

// handleTemplateSave replaces one event's payloads in a templates file.
//
// NOTE: the file is re-marshalled as JSON, which strips // comments and
// alphabetically reorders keys. Functionality is preserved.
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

	path := a.templateFilePath(file)
	root := map[string]any{}
	if err := loadJSONC(path, &root); err != nil {
		a.redirectFlash(w, r, "/templates", a.message(r, "flash.templateLoadFailed", err), "err")
		return
	}

	templatesNode, _ := root["templates"].(map[string]any)
	if templatesNode == nil {
		templatesNode = map[string]any{}
		root["templates"] = templatesNode
	}
	eventNode, _ := templatesNode[event].(map[string]any)
	if eventNode == nil {
		eventNode = map[string]any{}
	}
	eventNode["payloads"] = payloads
	templatesNode[event] = eventNode

	if err := SaveJSON(path, root); err != nil {
		a.redirectFlash(w, r, "/templates", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	a.notifySaved()
	a.redirectFlash(w, r, "/templates", a.message(r, "flash.templateSaved"), "ok")
}

// eventKeys returns the sorted event keys present in a parsed templates file.
func eventKeys(root map[string]any) []string {
	templatesNode, ok := root["templates"].(map[string]any)
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(templatesNode))
	for k := range templatesNode {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// payloadsJSON returns the indented JSON for one event's payloads array.
func payloadsJSON(root map[string]any, event string) string {
	templatesNode, _ := root["templates"].(map[string]any)
	if templatesNode == nil {
		return "[]"
	}
	eventNode, _ := templatesNode[event].(map[string]any)
	if eventNode == nil {
		return "[]"
	}
	payloads, _ := eventNode["payloads"].([]any)
	if payloads == nil {
		// payloads may be missing (means "default payload"); expose empty array.
		return "[]"
	}
	b, err := json.MarshalIndent(payloads, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(b)
}
