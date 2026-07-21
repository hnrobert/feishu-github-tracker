package panel

import (
	"net"
	"net/http"
	"strings"

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
	// Public /webhook URL for the setup guide. Empty (hidden) when the panel is
	// accessed via localhost / loopback / private IP, since such an address
	// isn't reachable by GitHub.
	data.PayloadURL = payloadURLFor(r)
	data.RecentLines = readRecentLogLines(a.logDir, 20)

	a.renderPage(w, "dashboard", data)
}

// payloadURLFor returns the public /webhook URL based on the request Host (and
// forwarded-proto), or "" when the access host is local (localhost / loopback /
// private / link-local), in which case the guide hides the URL.
func payloadURLFor(r *http.Request) string {
	host := r.Host
	if host == "" {
		return ""
	}
	hostName := portlessHost(host)
	if isLocalHost(hostName) {
		return ""
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + host + "/webhook"
}

// portlessHost strips the port from a Host header value, tolerating bracketed
// IPv6 literals (e.g. "[::1]:4594").
func portlessHost(host string) string {
	// IPv6 literal in brackets
	if strings.HasPrefix(host, "[") {
		if i := strings.Index(host, "]"); i > -1 {
			return host[1:i]
		}
	}
	if i := strings.LastIndex(host, ":"); i > -1 {
		return host[:i]
	}
	return host
}

// isLocalHost reports whether host is a local-only address (DNS name or IP).
func isLocalHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" || strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
	}
	return false
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
