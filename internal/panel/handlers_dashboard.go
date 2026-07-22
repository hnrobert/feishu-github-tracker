package panel

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

// handleDashboard renders the overview page with counts, server info and recent
// delivery-log activity.
func (a *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)

	cfg, _ := a.loadConfig()
	if cfg != nil {
		data.RepoCount = len(cfg.Repos.Repos)
		data.BotCount = len(cfg.FeishuBots.FeishuBots)
		data.EventSetCount = len(cfg.Events.EventSets)
		for name := range cfg.Templates {
			data.TemplateFiles = append(data.TemplateFiles, name)
		}
		data.TemplateFiles = sortedStrings(data.TemplateFiles)
		data.ServerInfo = serverInfoFrom(cfg)
	}
	// /webhook URL for the setup guide: prefer panel.public_url, else derive
	// from the request (hidden when accessed locally).
	publicURL := ""
	if cfg != nil {
		publicURL = cfg.Server.Panel.PublicURL
	}
	data.PayloadURL = payloadURLFor(r, publicURL)
	data.RecentLines = readRecentLogLines(a.logDir, 20)

	a.renderPage(w, "dashboard", data)
}

// payloadURLFor returns the /webhook URL to show in the setup guide.
//
//   - If publicURL (panel.public_url) is set, it is used verbatim (always shown,
//     even on local access), so the operator can pin the canonical address and
//     scheme — useful behind a TLS-terminating proxy that doesn't forward the
//     original scheme.
//   - Otherwise the URL is derived from the request Host + detected scheme; it
//     is hidden ("") when accessed via localhost / loopback / private IP.
func payloadURLFor(r *http.Request, publicURL string) string {
	if pu := strings.TrimSpace(publicURL); pu != "" {
		return joinWebhook(pu)
	}
	host := r.Host
	if host == "" {
		return ""
	}
	if isLocalHost(portlessHost(host)) {
		return ""
	}
	return requestScheme(r) + "://" + host + "/webhook"
}

// joinWebhook ensures base ends with /webhook.
func joinWebhook(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(base), "/webhook") {
		return base
	}
	return base + "/webhook"
}

// requestScheme detects http vs https. In order of preference: explicit
// forwarded headers (set by a TLS-terminating proxy), a direct TLS connection,
// Cloudflare's CF-Visitor, then the Referer header — a same-origin Referer
// carries the real public scheme even when the proxy chain (e.g. Caddy→nginx)
// doesn't forward X-Forwarded-Proto. Last resort is http.
func requestScheme(r *http.Request) string {
	for _, h := range []string{"X-Forwarded-Proto", "X-Forwarded-Scheme", "X-Forwarded-Protocol"} {
		if v := strings.ToLower(strings.TrimSpace(r.Header.Get(h))); v != "" {
			if strings.HasPrefix(v, "https") {
				return "https"
			}
			if strings.HasPrefix(v, "http") {
				return "http"
			}
		}
	}
	if r.TLS != nil {
		return "https"
	}
	if cf := strings.ToLower(r.Header.Get("CF-Visitor")); strings.Contains(cf, `"scheme":"https"`) {
		return "https"
	}
	if ref := r.Referer(); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.IsAbs() && sameHost(u.Host, r.Host) {
			return u.Scheme
		}
	}
	return "http"
}

// sameHost compares two host[:port] values ignoring port and case.
func sameHost(a, b string) bool {
	return strings.EqualFold(portlessHost(a), portlessHost(b))
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
