package panel

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

// handleBots lists all configured Feishu bots.
func (a *App) handleBots(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	if cfg, err := a.loadConfig(); err == nil {
		for i, b := range cfg.FeishuBots.FeishuBots {
			data.Bots = append(data.Bots, BotRow{Index: i, Alias: b.Alias, URL: b.URL, Template: b.Template})
		}
		data.Templates = a.knownTemplates(cfg)
	}
	a.renderPage(w, "bots", data)
}

// handleBotNew renders a blank bot edit form.
func (a *App) handleBotNew(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	data.EditBot = BotRow{Index: -1}
	if cfg, err := a.loadConfig(); err == nil {
		data.Templates = a.knownTemplates(cfg)
	}
	a.renderPage(w, "bot_edit", data)
}

// handleBotEdit renders the edit form for an existing bot by index.
func (a *App) handleBotEdit(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	idx, _ := strconv.Atoi(r.URL.Query().Get("index"))
	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/bots", a.message(r, "flash.configLoadFailed"), "err")
		return
	}
	data.Templates = a.knownTemplates(cfg)
	if idx < 0 || idx >= len(cfg.FeishuBots.FeishuBots) {
		a.redirectFlash(w, r, "/bots", a.message(r, "flash.botNotFound"), "err")
		return
	}
	b := cfg.FeishuBots.FeishuBots[idx]
	data.EditBot = BotRow{Index: idx, Alias: b.Alias, URL: b.URL, Template: b.Template}
	a.renderPage(w, "bot_edit", data)
}

// handleBotSave creates or updates a bot.
func (a *App) handleBotSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/bots", a.message(r, "flash.invalidForm"), "err")
		return
	}
	idx, _ := strconv.Atoi(r.FormValue("index"))
	alias := strings.TrimSpace(r.FormValue("alias"))
	url := strings.TrimSpace(r.FormValue("url"))
	tmpl := strings.TrimSpace(r.FormValue("template"))
	if alias == "" || url == "" {
		a.redirectFlash(w, r, "/bots", a.message(r, "flash.botFieldsRequired"), "err")
		return
	}

	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/bots", a.message(r, "flash.configLoadFailed"), "err")
		return
	}

	bot := config.FeishuBot{Alias: alias, URL: url, Template: tmpl}
	if idx >= 0 && idx < len(cfg.FeishuBots.FeishuBots) {
		cfg.FeishuBots.FeishuBots[idx] = bot
	} else {
		cfg.FeishuBots.FeishuBots = append(cfg.FeishuBots.FeishuBots, bot)
	}

	if err := SaveYAML(a.cfgDir+"/feishu-bots.yaml", cfg.FeishuBots); err != nil {
		a.redirectFlash(w, r, "/bots", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	a.notifySaved()
	a.redirectFlash(w, r, "/bots", a.message(r, "flash.botSaved"), "ok")
}

// handleBotDelete removes a bot by index.
func (a *App) handleBotDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/bots", a.message(r, "flash.invalidForm"), "err")
		return
	}
	idx, _ := strconv.Atoi(r.FormValue("index"))
	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/bots", a.message(r, "flash.configLoadFailed"), "err")
		return
	}
	if idx < 0 || idx >= len(cfg.FeishuBots.FeishuBots) {
		a.redirectFlash(w, r, "/bots", a.message(r, "flash.botNotFound"), "err")
		return
	}
	cfg.FeishuBots.FeishuBots = append(cfg.FeishuBots.FeishuBots[:idx], cfg.FeishuBots.FeishuBots[idx+1:]...)
	if err := SaveYAML(a.cfgDir+"/feishu-bots.yaml", cfg.FeishuBots); err != nil {
		a.redirectFlash(w, r, "/bots", a.message(r, "flash.saveFailed", err), "err")
		return
	}
	a.notifySaved()
	a.redirectFlash(w, r, "/bots", a.message(r, "flash.botDeleted"), "ok")
}
