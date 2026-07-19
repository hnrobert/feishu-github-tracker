package panel

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/auth"
)

// handleSettings renders the server.yaml editor.
func (a *App) handleSettings(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r)
	if cfg, err := a.loadConfig(); err == nil {
		s := cfg.Server.Server
		data.ServerForm = ServerForm{
			Host:           s.Host,
			Port:           s.Port,
			Secret:         s.Secret,
			LogLevel:       s.LogLevel,
			MaxPayloadSize: s.MaxPayloadSize,
			Timeout:        s.Timeout,
			AllowedSources: strings.Join(cfg.Server.AllowedSources, "\n"),
		}
	}
	a.renderPage(w, "server_settings", data)
}

// handleSettingsSave persists server.yaml edits via yaml.Node mutation so all
// existing comments (including the panel `# password: "admin"` hint) are
// preserved. Port/secret changes require a restart to take effect.
func (a *App) handleSettingsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/settings", "表单解析失败 / invalid form", "err")
		return
	}

	port, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("port")))
	timeout, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("timeout")))
	form := struct {
		host, secret, logLevel, maxPayload string
		port, timeout                      int
		allowed                            []string
	}{
		host:       strings.TrimSpace(r.FormValue("host")),
		secret:     strings.TrimSpace(r.FormValue("secret")),
		logLevel:   strings.TrimSpace(r.FormValue("log_level")),
		maxPayload: strings.TrimSpace(r.FormValue("max_payload_size")),
		port:       port,
		timeout:    timeout,
		allowed:    splitLines(r.FormValue("allowed_sources")),
	}

	root, err := loadServerRoot(a.cfgDir)
	if err != nil {
		a.redirectFlash(w, r, "/settings", "读取配置失败 / failed to load config", "err")
		return
	}
	serverMap := ensureMap(root, "server")
	mapSet(serverMap, "host", form.host)
	if form.port > 0 {
		mapSetPlain(serverMap, "port", strconv.Itoa(form.port))
	}
	mapSet(serverMap, "secret", form.secret)
	mapSet(serverMap, "log_level", form.logLevel)
	mapSet(serverMap, "max_payload_size", form.maxPayload)
	if form.timeout > 0 {
		mapSetPlain(serverMap, "timeout", strconv.Itoa(form.timeout))
	}
	setTopLevelSequence(root, "allowed_sources", form.allowed)

	if err := writeServerRoot(a.cfgDir, root); err != nil {
		a.redirectFlash(w, r, "/settings", "保存失败: "+err.Error(), "err")
		return
	}

	// Optional panel password rotation: re-hash and write via the comment-
	// preserving path. This removes any plaintext password and sets password_hash
	// with the `# password: "admin"` hint.
	flash := "服务设置已保存（端口/密钥需重启生效）/ settings saved (port/secret need restart)"
	if pw := strings.TrimSpace(r.FormValue("panel_password")); pw != "" {
		hash, err := auth.HashPassword(pw)
		if err != nil {
			a.redirectFlash(w, r, "/settings", "密码哈希失败: "+err.Error(), "err")
			return
		}
		if err := SetPanelPasswordHash(a.cfgDir, hash); err != nil {
			a.redirectFlash(w, r, "/settings", "密码保存失败: "+err.Error(), "err")
			return
		}
		flash = "服务设置与面板密码已保存 / settings and panel password saved"
	}
	a.redirectFlash(w, r, "/settings", flash, "ok")
}
