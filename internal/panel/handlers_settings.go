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

// handleSettingsSave persists server.yaml edits. Port/secret changes require a
// restart to take effect (the running process keeps its original listener).
func (a *App) handleSettingsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/settings", "表单解析失败 / invalid form", "err")
		return
	}

	cfg, err := a.loadConfig()
	if err != nil {
		a.redirectFlash(w, r, "/settings", "读取配置失败 / failed to load config", "err")
		return
	}

	port, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("port")))
	timeout, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("timeout")))

	cfg.Server.Server.Host = strings.TrimSpace(r.FormValue("host"))
	if port > 0 {
		cfg.Server.Server.Port = port
	}
	cfg.Server.Server.Secret = strings.TrimSpace(r.FormValue("secret"))
	cfg.Server.Server.LogLevel = strings.TrimSpace(r.FormValue("log_level"))
	cfg.Server.Server.MaxPayloadSize = strings.TrimSpace(r.FormValue("max_payload_size"))
	if timeout > 0 {
		cfg.Server.Server.Timeout = timeout
	}
	cfg.Server.AllowedSources = splitLines(r.FormValue("allowed_sources"))

	// Optional panel password rotation: if a new password is provided, hash it
	// and store as password_hash (clearing any plaintext password).
	if pw := strings.TrimSpace(r.FormValue("panel_password")); pw != "" {
		hash, err := auth.HashPassword(pw)
		if err != nil {
			a.redirectFlash(w, r, "/settings", "密码哈希失败: "+err.Error(), "err")
			return
		}
		cfg.Server.Panel.PasswordHash = hash
		cfg.Server.Panel.Password = ""
		cfg.Server.Panel.Enabled = true
	}

	if err := SaveYAML(a.cfgDir+"/server.yaml", cfg.Server); err != nil {
		a.redirectFlash(w, r, "/settings", "保存失败: "+err.Error(), "err")
		return
	}
	a.redirectFlash(w, r, "/settings", "服务设置已保存（端口/密钥需重启生效）/ settings saved (port/secret need restart)", "ok")
}
