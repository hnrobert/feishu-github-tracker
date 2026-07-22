package panel

import (
	"os"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/auth"
	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"gopkg.in/yaml.v3"
)

// resolveCredentials returns the effective admin username and password_hash
// for login, resolved fresh from the current server.yaml + environment.
//
// Password precedence:
//  1. PANEL_PASSWORD env (plaintext, hashed here)
//  2. server.yaml panel.password (plaintext, hashed here) — takes priority over
//     password_hash so an operator can rotate by setting a new plaintext value
//  3. server.yaml panel.password_hash (sha256 hex, or legacy bcrypt)
//  4. fallback "admin" (so upgraders with no panel config can log in as admin/admin)
//
// Username precedence: PANEL_USERNAME env > panel.username > "admin".
func resolveCredentials(cfgDir string) (username string, passHash []byte) {
	username = "admin"
	if u := os.Getenv("PANEL_USERNAME"); u != "" {
		username = u
	}

	var pc config.PanelConfig
	if sc, err := readServerPanel(cfgDir); err == nil {
		pc = sc
	}
	if u := pc.Username; u != "" {
		if os.Getenv("PANEL_USERNAME") == "" {
			username = u
		}
	}

	switch {
	case os.Getenv("PANEL_PASSWORD") != "":
		passHash = []byte(auth.HashPlaintext(os.Getenv("PANEL_PASSWORD")))
	case pc.Password != "":
		passHash = []byte(auth.HashPlaintext(pc.Password))
	case pc.PasswordHash != "":
		passHash = []byte(pc.PasswordHash)
	default:
		// No password configured at all (e.g. upgraders): default to "admin".
		passHash = []byte(auth.HashPlaintext("admin"))
	}
	return username, passHash
}

// readServerPanel parses just server.yaml and returns its panel block.
func readServerPanel(cfgDir string) (config.PanelConfig, error) {
	data, err := os.ReadFile(cfgDir + "/server.yaml")
	if err != nil {
		return config.PanelConfig{}, err
	}
	var sc config.ServerConfig
	if err := yaml.Unmarshal(data, &sc); err != nil {
		return config.PanelConfig{}, err
	}
	return sc.Panel, nil
}

// passwordCommentHint is the documented default-password hint comment placed
// next to password_hash after a plaintext password is converted.
const passwordCommentHint = `password: "admin"`

// NormalizePanelPassword performs the plaintext→hash auto-conversion described
// in the panel config contract:
//
//   - If server.yaml has an uncommented panel.password (plaintext), hash it,
//     store it as panel.password_hash (overwriting any existing hash), remove
//     the plaintext password line, and add a `# password: "admin"` comment so
//     operators know they can rotate by setting a new plaintext password.
//
// It is a no-op (returns false) when no plaintext password is present. All
// other content/comments in server.yaml are preserved via yaml.Node round-trip.
func NormalizePanelPassword(cfgDir string) (changed bool, err error) {
	root, err := loadServerRoot(cfgDir)
	if err != nil {
		return false, err
	}
	panel := mapGet(topMap(root), "panel")
	if panel == nil || panel.Kind != yaml.MappingNode {
		return false, nil
	}

	pwNode := mapGet(panel, "password")
	if pwNode == nil || strings.TrimSpace(pwNode.Value) == "" {
		return false, nil
	}

	hash := auth.HashPlaintext(pwNode.Value)

	mapDelete(panel, "password")
	setPanelHashWithHint(panel, hash)

	if err := writeServerRoot(cfgDir, root); err != nil {
		return false, err
	}
	return true, nil
}

// setPanelHashWithHint sets password_hash to hash, attaches the
// `# password: "admin"` hint as a stable standalone comment on the password_hash
// KEY node (yaml.v3 renders a mapping pair's leading comment from the key node),
// and reorders the panel block canonically so the hint is not the trailing item.
func setPanelHashWithHint(panel *yaml.Node, hash string) {
	mapSet(panel, "password_hash", hash)
	for i := 0; i+1 < len(panel.Content); i += 2 {
		if panel.Content[i].Value == "password_hash" {
			panel.Content[i].LineComment = ""
			panel.Content[i].FootComment = ""
			panel.Content[i].HeadComment = passwordCommentHint
			break
		}
	}
	reorderPanel(panel)
}

// SetPanelPasswordHash writes the given password_hash (sha256 of the password,
// as produced by the browser) as panel.password_hash, removes
// any plaintext panel.password, and ensures the `# password: "admin"` hint
// comment is present. Used by the panel's "change password" form. Other content
// and comments in server.yaml are preserved.
func SetPanelPasswordHash(cfgDir, hash string) error {
	root, err := loadServerRoot(cfgDir)
	if err != nil {
		return err
	}
	panel := mapGet(topMap(root), "panel")
	if panel == nil || panel.Kind != yaml.MappingNode {
		// No panel block at all: nothing to do (login isn't file-configured).
		return nil
	}
	mapDelete(panel, "password")
	setPanelHashWithHint(panel, hash)
	return writeServerRoot(cfgDir, root)
}

// SetPanelUsername writes panel.username. Other content and comments in
// server.yaml are preserved.
func SetPanelUsername(cfgDir, username string) error {
	root, err := loadServerRoot(cfgDir)
	if err != nil {
		return err
	}
	panel := ensureMap(root, "panel")
	mapSet(panel, "username", username)
	return writeServerRoot(cfgDir, root)
}
