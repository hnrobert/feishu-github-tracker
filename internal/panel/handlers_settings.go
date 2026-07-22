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
	// Show the effective admin username (env > config > "admin").
	if u, _ := resolveCredentials(a.cfgDir); u != "" {
		data.ServerForm.Username = u
	}
	a.renderPage(w, "server_settings", data)
}

// handleSettingsSave persists server.yaml edits via yaml.Node mutation so all
// existing comments (including the panel `# password: "admin"` hint) are
// preserved. It also handles panel username changes and password rotation
// (which requires the current password). After saving it triggers a reload so
// changes take effect immediately. Port/secret still require a restart.
func (a *App) handleSettingsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/settings", "表单解析失败 / invalid form", "err")
		return
	}

	port, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("port")))
	timeout, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("timeout")))
	host := strings.TrimSpace(r.FormValue("host"))
	secret := strings.TrimSpace(r.FormValue("secret"))
	logLevel := strings.TrimSpace(r.FormValue("log_level"))
	maxPayload := strings.TrimSpace(r.FormValue("max_payload_size"))
	allowed := splitLines(r.FormValue("allowed_sources"))

	newUsername := strings.TrimSpace(r.FormValue("panel_username"))
	oldPassword := r.FormValue("panel_old_password")
	newPassword := strings.TrimSpace(r.FormValue("panel_password"))
	confirmPassword := strings.TrimSpace(r.FormValue("panel_password_confirm"))

	// Validate credential changes BEFORE writing anything.
	currentUsername, currentHash := resolveCredentials(a.cfgDir)
	usernameChanged := newUsername != "" && newUsername != currentUsername
	passwordChanged := newPassword != ""
	if passwordChanged {
		if newPassword != confirmPassword {
			a.redirectFlash(w, r, "/settings", "两次输入的新密码不一致 / new passwords do not match", "err")
			return
		}
		if !auth.VerifyPassword(string(currentHash), oldPassword) {
			a.redirectFlash(w, r, "/settings", "旧密码错误，密码未修改 / incorrect old password", "err")
			return
		}
	}

	// 1. Persist server.* fields (+ panel.username) in one comment-preserving write.
	root, err := loadServerRoot(a.cfgDir)
	if err != nil {
		a.redirectFlash(w, r, "/settings", "读取配置失败 / failed to load config", "err")
		return
	}
	serverMap := ensureMap(root, "server")
	mapSet(serverMap, "host", host)
	if port > 0 {
		mapSetPlain(serverMap, "port", strconv.Itoa(port))
	}
	mapSet(serverMap, "secret", secret)
	mapSet(serverMap, "log_level", logLevel)
	mapSet(serverMap, "max_payload_size", maxPayload)
	if timeout > 0 {
		mapSetPlain(serverMap, "timeout", strconv.Itoa(timeout))
	}
	setTopLevelSequence(root, "allowed_sources", allowed)
	if usernameChanged {
		mapSet(ensureMap(root, "panel"), "username", newUsername)
	}
	if err := writeServerRoot(a.cfgDir, root); err != nil {
		a.redirectFlash(w, r, "/settings", "保存失败: "+err.Error(), "err")
		return
	}

	// 2. Persist password rotation. newPassword is already the SHA-256
	//    password_hash produced by the browser, so store it directly.
	if passwordChanged {
		if err := SetPanelPasswordHash(a.cfgDir, newPassword); err != nil {
			a.redirectFlash(w, r, "/settings", "密码保存失败: "+err.Error(), "err")
			return
		}
	}

	// 3. Reload so changes take effect immediately.
	a.notifySaved()

	flash := "服务设置已保存（端口/密钥需重启生效）/ settings saved (port/secret need restart)"
	switch {
	case usernameChanged && passwordChanged:
		flash = "服务设置、用户名与密码已保存 / settings, username and password saved"
	case usernameChanged:
		flash = "服务设置与用户名已保存 / settings and username saved"
	case passwordChanged:
		flash = "服务设置与密码已保存 / settings and password saved"
	}
	a.redirectFlash(w, r, "/settings", flash, "ok")
}
