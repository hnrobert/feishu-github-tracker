// Package panel implements the web management UI for feishu-github-tracker.
// It is a server-rendered admin panel modeled on the lumgr project: net/http
// + html/template with an embedded layout, JWT-cookie auth, and CRUD over the
// tracker's YAML/JSONC configuration files. Edits are written to disk and take
// effect via the binary's -reload flag (or a restart).
package panel

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hnrobert/feishu-github-tracker/internal/auth"
	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

//go:embed templates/*.html
var templatesFS embed.FS

const sessionTTL = 24 * time.Hour

// Options configures a panel App at construction time.
type Options struct {
	ConfigDir string // directory holding server.yaml, repos.yaml, etc.
	LogDir    string // directory holding delivery logs (for dashboard tail)
	PassHash  []byte // bcrypt hash of the admin password; if empty, login is disabled
	JWTSecret []byte // JWT signing secret; if empty, an ephemeral random secret is used
}

// App holds panel state and serves HTTP.
type App struct {
	secret     []byte
	cookieName string
	cfgDir     string
	logDir     string
	passHash   []byte
	pages      map[string]*template.Template
	handler    http.Handler
}

// ViewData is the single render context passed to every page template.
type ViewData struct {
	// auth / nav
	Authed      bool
	Username    string
	HideNav     bool
	Flash       string
	FlashKind   string // "ok" | "err" | ""
	CurrentPage string

	// dashboard
	RepoCount     int
	BotCount      int
	EventSetCount int
	TemplateFiles []string
	ServerInfo    ServerInfo
	RecentLines   []string

	// repos
	Repos    []RepoRow
	EditRepo RepoRow

	// bots
	Bots      []BotRow
	EditBot   BotRow
	Templates []string // known template names for the bot template selector

	// server settings
	ServerForm ServerForm

	// events
	EventSetsYAML string
	EventsYAML    string

	// templates
	TemplateFilesList []TemplateFileRow
	EditTemplate      EditTemplateData
}

// ServerInfo captures read-only server status shown on the dashboard.
type ServerInfo struct {
	Host           string
	Port           int
	LogLevel       string
	MaxPayloadSize string
	Timeout        int
	AllowedSources []string
	PanelEnabled   bool
}

// RepoRow represents one repos.yaml entry, for both list display and editing.
type RepoRow struct {
	Index       int
	Pattern     string
	NotifyTo    []string // list display
	NotifyToRaw string   // newline-joined, for the edit form
	Events      map[string]any
	EventsYAML  string // raw YAML text, for the edit textarea
	EventCount  int
}

// BotRow represents one feishu-bots.yaml entry.
type BotRow struct {
	Index    int
	Alias    string
	URL      string
	Template string
}

// ServerForm holds editable server.yaml fields.
type ServerForm struct {
	Host           string
	Port           int
	Secret         string
	LogLevel       string
	MaxPayloadSize string
	Timeout        int
	AllowedSources string // newline-joined
	PanelPassword  string // optional new plaintext password (re-hashed on save)
}

// TemplateFileRow represents one templates.*.jsonc file.
type TemplateFileRow struct {
	Name  string
	Count int
}

// EditTemplateData holds the per-event template editor state.
type EditTemplateData struct {
	File         string   // template name (e.g. "default", "cn")
	Events       []string // available event keys
	Event        string   // selected event key
	PayloadsJSON string   // editable JSON for the event's payloads array
}

// New constructs a panel App from opts.
func New(opts Options) (*App, error) {
	secret := opts.JWTSecret
	if len(secret) == 0 {
		s, err := auth.NewRandomSecretB64(32)
		if err != nil {
			return nil, err
		}
		secret = []byte(s)
	}

	base := template.New("layout.html").Funcs(template.FuncMap{
		"eq": func(a, b string) bool {
			a = strings.TrimSpace(a)
			b = strings.TrimSpace(b)
			return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
		},
		"contains": func(list []string, s string) bool {
			for _, v := range list {
				if v == s {
					return true
				}
			}
			return false
		},
		"startsWith": func(s, p string) bool {
			return strings.HasPrefix(strings.TrimSpace(s), strings.TrimSpace(p))
		},
		"trim": func(s string) string { return strings.TrimSpace(s) },
		"toJSON": func(v any) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS("null")
			}
			return template.JS(b)
		},
	})

	pages := map[string]*template.Template{}
	for _, page := range []string{
		"login",
		"dashboard",
		"repos",
		"repo_edit",
		"bots",
		"bot_edit",
		"server_settings",
		"events",
		"templates_list",
		"template_edit",
	} {
		t, err := base.Clone()
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(templatesFS, "templates/layout.html", "templates/"+page+".html"); err != nil {
			return nil, err
		}
		pages[page] = t
	}

	a := &App{
		secret:     secret,
		cookieName: auth.DefaultCookieName,
		cfgDir:     opts.ConfigDir,
		logDir:     opts.LogDir,
		passHash:   opts.PassHash,
		pages:      pages,
	}
	a.handler = a.withAuthContext(a.routes())
	return a, nil
}

// Enabled reports whether the panel has login configured.
func (a *App) Enabled() bool { return len(a.passHash) > 0 }

// ServeHTTP routes the request through the panel's mux.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/login", a.handleLogin)
	mux.HandleFunc("/logout", a.handleLogout)

	mux.HandleFunc("/", a.requireAuth(a.handleDashboard))
	mux.HandleFunc("/repos", a.requireAuth(a.handleRepos))
	mux.HandleFunc("/repos/new", a.requireAuth(a.handleRepoNew))
	mux.HandleFunc("/repos/edit", a.requireAuth(a.handleRepoEdit))
	mux.HandleFunc("/repos/save", a.requireAuth(a.handleRepoSave))
	mux.HandleFunc("/repos/delete", a.requireAuth(a.handleRepoDelete))

	mux.HandleFunc("/bots", a.requireAuth(a.handleBots))
	mux.HandleFunc("/bots/new", a.requireAuth(a.handleBotNew))
	mux.HandleFunc("/bots/edit", a.requireAuth(a.handleBotEdit))
	mux.HandleFunc("/bots/save", a.requireAuth(a.handleBotSave))
	mux.HandleFunc("/bots/delete", a.requireAuth(a.handleBotDelete))

	mux.HandleFunc("/settings", a.requireAuth(a.handleSettings))
	mux.HandleFunc("/settings/save", a.requireAuth(a.handleSettingsSave))

	mux.HandleFunc("/events", a.requireAuth(a.handleEvents))
	mux.HandleFunc("/events/save", a.requireAuth(a.handleEventsSave))

	mux.HandleFunc("/templates", a.requireAuth(a.handleTemplatesList))
	mux.HandleFunc("/templates/edit", a.requireAuth(a.handleTemplateEdit))
	mux.HandleFunc("/templates/save", a.requireAuth(a.handleTemplateSave))

	return mux
}

// loadConfig re-reads the configuration from disk so the panel always reflects
// the current state (the same loader the hot-reload path uses).
func (a *App) loadConfig() (*config.Config, error) {
	return config.Load(a.cfgDir)
}

// renderPage executes the named page template against data.
func (a *App) renderPage(w http.ResponseWriter, page string, data ViewData) {
	t, ok := a.pages[page]
	if !ok {
		http.Error(w, "unknown page: "+page, http.StatusInternalServerError)
		return
	}
	data.CurrentPage = page
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// baseData returns a ViewData pre-populated with auth + flash from the request.
func (a *App) baseData(r *http.Request) ViewData {
	q := r.URL.Query()
	return ViewData{
		Authed:    true,
		Username:  usernameFrom(r),
		Flash:     q.Get("flash"),
		FlashKind: q.Get("kind"),
	}
}

// redirectFlash redirects to dest with a flash message rendered on arrival.
func (a *App) redirectFlash(w http.ResponseWriter, r *http.Request, dest, flash, kind string) {
	u := dest
	if flash != "" {
		sep := "?"
		if strings.Contains(dest, "?") {
			sep = "&"
		}
		u = dest + sep + "kind=" + kind + "&flash=" + urlQueryEscape(flash)
	}
	http.Redirect(w, r, u, http.StatusSeeOther)
}

// urlQueryEscape is a minimal query escaper to avoid importing net/url in app.go.
func urlQueryEscape(s string) string { return strings.NewReplacer(" ", "%20", "&", "%26").Replace(s) }

// knownTemplates returns the template names available in the current config
// (e.g. ["default", "cn"]), sorted, for the bot template selector.
func (a *App) knownTemplates(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Templates))
	for k := range cfg.Templates {
		names = append(names, k)
	}
	return sortedStrings(names)
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	// simple insertion sort to avoid pulling sort for a tiny slice
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// readRecentLogLines tails up to n lines from the most recent .log file in
// logDir, keeping only delivery-relevant lines. It degrades to nil on any error.
func readRecentLogLines(logDir string, n int) []string {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil
	}
	var newest os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if newest == nil || e.Name() > newest.Name() {
			newest = e
		}
	}
	if newest == nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(logDir, newest.Name()))
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	var kept []string
	for _, l := range lines {
		if strings.Contains(l, "Successfully sent") || strings.Contains(l, "Failed") || strings.Contains(l, "notification") {
			kept = append(kept, l)
		}
	}
	if len(kept) > n {
		kept = kept[len(kept)-n:]
	}
	return kept
}
