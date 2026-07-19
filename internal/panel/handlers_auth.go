package panel

import (
	"net/http"

	"github.com/hnrobert/feishu-github-tracker/internal/auth"
)

// handleLogin: GET renders the login card; POST verifies the admin password and
// establishes a JWT session.
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.handleLoginPost(w, r)
		return
	}

	q := r.URL.Query()
	a.renderPage(w, "login", ViewData{
		HideNav:     true,
		CurrentPage: "login",
		Flash:       q.Get("flash"),
		FlashKind:   q.Get("kind"),
	})
}

func (a *App) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if !a.Enabled() {
		a.redirectFlash(w, r, "/login", "面板未配置管理员密码 / panel login not configured", "err")
		return
	}
	if err := r.ParseForm(); err != nil {
		a.redirectFlash(w, r, "/login", "表单解析失败 / invalid form", "err")
		return
	}
	password := r.FormValue("password")
	if !auth.VerifyPassword(string(a.passHash), password) {
		a.redirectFlash(w, r, "/login", "密码错误 / incorrect password", "err")
		return
	}

	tok, err := auth.SignHS256(a.secret, "admin", true, sessionTTL)
	if err != nil {
		http.Error(w, "failed to issue session", http.StatusInternalServerError)
		return
	}
	a.issueCookie(w, tok)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	a.clearCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
