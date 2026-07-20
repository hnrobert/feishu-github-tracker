package panel

import (
	"net/http"
	"os"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"gopkg.in/yaml.v3"
)

// handleEvents shows the raw events.yaml in an editor. The whole file is
// edited as text so comments and ordering are preserved exactly.
func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	if b, err := os.ReadFile(a.cfgDir + "/events.yaml"); err == nil {
		data.EventsYAML = string(b)
	}
	a.renderPage(w, "events", data)
}

// handleEventsSave validates the edited events.yaml (by parsing it) and, if
// valid, writes the raw text back verbatim.
func (a *App) handleEventsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/events", "表单解析失败 / invalid form", "err")
		return
	}
	text := r.FormValue("events_yaml")

	var check config.EventsConfig
	if err := yaml.Unmarshal([]byte(text), &check); err != nil {
		a.redirectFlash(w, r, "/events", "events.yaml 解析失败: "+err.Error(), "err")
		return
	}

	if err := os.WriteFile(a.cfgDir+"/events.yaml", []byte(text), 0o644); err != nil {
		a.redirectFlash(w, r, "/events", "保存失败: "+err.Error(), "err")
		return
	}
	a.notifySaved()
	a.redirectFlash(w, r, "/events", "事件配置已保存 / events saved", "ok")
}
