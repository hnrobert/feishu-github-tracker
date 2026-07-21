package panel

import (
	"net/http"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

// handleDashboard renders the overview page with counts, server info and recent
// delivery-log activity.
func (a *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)

	if cfg, err := a.loadConfig(); err == nil {
		data.RepoCount = len(cfg.Repos.Repos)
		data.BotCount = len(cfg.FeishuBots.FeishuBots)
		data.EventSetCount = len(cfg.Events.EventSets)
		for name := range cfg.Templates {
			data.TemplateFiles = append(data.TemplateFiles, name)
		}
		data.TemplateFiles = sortedStrings(data.TemplateFiles)
		data.ServerInfo = serverInfoFrom(cfg)
	}
	data.RecentLines = readRecentLogLines(a.logDir, 20)

	a.renderPage(w, "dashboard", data)
}

func serverInfoFrom(cfg *config.Config) ServerInfo {
	s := cfg.Server.Server
	return ServerInfo{
		Host:           s.Host,
		Port:           s.Port,
		LogLevel:       s.LogLevel,
		MaxPayloadSize: s.MaxPayloadSize,
		Timeout:        s.Timeout,
		AllowedSources: cfg.Server.AllowedSources,
	}
}
