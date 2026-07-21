package panel

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"gopkg.in/yaml.v3"
)

// handleRepos lists all repo rules.
func (a *App) handleRepos(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	if cfg, err := a.loadConfig(); err == nil {
		for i, rp := range cfg.Repos.Repos {
			data.Repos = append(data.Repos, repoListRow(i, rp))
		}
	}
	a.renderPage(w, "repos", data)
}

// handleRepoNew renders a blank edit form for a new repo rule.
func (a *App) handleRepoNew(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	data.EditRepo = RepoRow{Index: -1}
	a.renderPage(w, "repo_edit", data)
}

// handleRepoEdit renders the edit form for an existing repo rule by index.
func (a *App) handleRepoEdit(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	idx, _ := strconv.Atoi(r.URL.Query().Get("index"))
	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/repos", "读取配置失败 / failed to load config", "err")
		return
	}
	if idx < 0 || idx >= len(cfg.Repos.Repos) {
		a.redirectFlash(w, r, "/repos", "仓库不存在 / repo not found", "err")
		return
	}
	data.EditRepo = repoEditRow(idx, cfg.Repos.Repos[idx])
	a.renderPage(w, "repo_edit", data)
}

// handleRepoSave creates or updates a repo rule.
func (a *App) handleRepoSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/repos", "表单解析失败 / invalid form", "err")
		return
	}
	idx, _ := strconv.Atoi(r.FormValue("index"))
	pattern := strings.TrimSpace(r.FormValue("pattern"))
	if pattern == "" {
		a.redirectFlash(w, r, "/repos", "pattern 不能为空 / pattern must not be empty", "err")
		return
	}

	events, err := parseEventsYAML(r.FormValue("events"))
	if err != nil {
		a.redirectFlash(w, r, "/repos", "events YAML 解析失败: "+err.Error(), "err")
		return
	}
	notifyTo := splitLines(r.FormValue("notify_to"))

	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/repos", "读取配置失败 / failed to load config", "err")
		return
	}

	rp := config.RepoPattern{Pattern: pattern, Events: events, NotifyTo: notifyTo}
	if idx >= 0 && idx < len(cfg.Repos.Repos) {
		cfg.Repos.Repos[idx] = rp
	} else {
		cfg.Repos.Repos = append(cfg.Repos.Repos, rp)
	}

	if err := SaveYAML(a.cfgDir+"/repos.yaml", cfg.Repos); err != nil {
		a.redirectFlash(w, r, "/repos", "保存失败: "+err.Error(), "err")
		return
	}
	a.notifySaved()
	a.redirectFlash(w, r, "/repos", "仓库规则已保存 / repo rule saved", "ok")
}

// handleRepoDelete removes a repo rule by index.
func (a *App) handleRepoDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/repos", "表单解析失败 / invalid form", "err")
		return
	}
	idx, _ := strconv.Atoi(r.FormValue("index"))
	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/repos", "读取配置失败 / failed to load config", "err")
		return
	}
	if idx < 0 || idx >= len(cfg.Repos.Repos) {
		a.redirectFlash(w, r, "/repos", "仓库不存在 / repo not found", "err")
		return
	}
	cfg.Repos.Repos = append(cfg.Repos.Repos[:idx], cfg.Repos.Repos[idx+1:]...)
	if err := SaveYAML(a.cfgDir+"/repos.yaml", cfg.Repos); err != nil {
		a.redirectFlash(w, r, "/repos", "保存失败: "+err.Error(), "err")
		return
	}
	a.notifySaved()
	a.redirectFlash(w, r, "/repos", "仓库规则已删除 / repo rule deleted", "ok")
}

// repoListRow builds a RepoRow for list display.
func repoListRow(i int, rp config.RepoPattern) RepoRow {
	return RepoRow{
		Index:      i,
		Pattern:    rp.Pattern,
		NotifyTo:   rp.NotifyTo,
		EventCount: len(rp.Events),
	}
}

// repoEditRow builds a RepoRow for the edit form (with raw textarea contents).
func repoEditRow(i int, rp config.RepoPattern) RepoRow {
	row := RepoRow{
		Index:       i,
		Pattern:     rp.Pattern,
		NotifyTo:    rp.NotifyTo,
		NotifyToRaw: strings.Join(rp.NotifyTo, "\n"),
		Events:      rp.Events,
		EventCount:  len(rp.Events),
	}
	if len(rp.Events) > 0 {
		if b, err := yaml.Marshal(rp.Events); err == nil {
			row.EventsYAML = strings.TrimRight(string(b), "\n")
		}
	}
	return row
}

// parseEventsYAML parses the events textarea into a map[string]any. An empty
// textarea yields an empty (non-nil) map so the rule still subscribes to events.
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
