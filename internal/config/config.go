package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration
type Config struct {
	Server     ServerConfig
	Repos      ReposConfig
	Events     EventsConfig
	FeishuBots FeishuBotsConfig
	Templates  map[string]TemplatesConfig
}

type ServerConfig struct {
	Server struct {
		Host           string `yaml:"host"`
		Port           int    `yaml:"port"`
		Secret         string `yaml:"secret"`
		LogLevel       string `yaml:"log_level"`
		MatchAllRules  bool   `yaml:"match_all_rules"`
		MaxPayloadSize string `yaml:"max_payload_size"`
		Timeout        int    `yaml:"timeout"`
	} `yaml:"server"`
	AllowedSources []string    `yaml:"allowed_sources"`
	Panel          PanelConfig `yaml:"panel"`
}

type PanelConfig struct {
	Enabled      bool   `yaml:"enabled,omitempty"`
	Username     string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
	PasswordHash string `yaml:"password_hash,omitempty"`
	Secret       string `yaml:"secret,omitempty"`
	PublicURL    string `yaml:"public_url,omitempty"`
}

type ReposConfig struct {
	Repos []RepoPattern `yaml:"repos"`
}

type RepoPattern struct {
	Pattern  string         `yaml:"pattern"`
	Events   map[string]any `yaml:"events"`
	NotifyTo []string       `yaml:"notify_to"`
	Secret   string         `yaml:"secret,omitempty"`
}

type EventsConfig struct {
	EventSets map[string]map[string]any `yaml:"event_sets"`
	Events    map[string]any            `yaml:"events"`
}

type FeishuBotsConfig struct {
	FeishuBots []FeishuBot `yaml:"feishu_bots"`
}

type FeishuBot struct {
	Alias    string `yaml:"alias"`
	URL      string `yaml:"url"`
	Template string `yaml:"template,omitempty"`
}

type TemplatesConfig struct {
	Templates map[string]EventTemplate `yaml:"templates"`
}

type EventTemplate struct {
	Payloads []PayloadTemplate `yaml:"payloads"`
}

type PayloadTemplate struct {
	Tags    []string       `yaml:"tags"`
	Payload map[string]any `yaml:"payload"`
}

// Load loads all configuration from configDir, transparently supporting
// both legacy flat files and per-item subdirectories (subdirs checked first).
func Load(configDir string) (*Config, error) {
	cfg := &Config{Templates: make(map[string]TemplatesConfig)}

	if err := loadConfigFile(filepath.Join(configDir, "server.yaml"), &cfg.Server); err != nil {
		return nil, fmt.Errorf("failed to load server.yaml: %w", err)
	}

	repos, err := loadPatterns(configDir)
	if err != nil {
		return nil, err
	}
	cfg.Repos = ReposConfig{Repos: repos}

	bots, err := loadBots(configDir)
	if err != nil {
		return nil, err
	}
	cfg.FeishuBots = FeishuBotsConfig{FeishuBots: bots}

	events, err := loadEvents(configDir)
	if err != nil {
		return nil, err
	}
	cfg.Events = events

	templates, err := loadTemplates(configDir)
	if err != nil {
		return nil, err
	}
	cfg.Templates = templates

	return cfg, nil
}

// ── Per-type merge loaders (subdir-first, flat-file fallback) ──

func loadPatterns(configDir string) ([]RepoPattern, error) {
	subDir := filepath.Join(configDir, "patterns")
	entries, err := scanYAMLDir(subDir)
	if err == nil && len(entries) > 0 {
		var repos []RepoPattern
		for _, name := range entries {
			var rp RepoPattern
			if err := loadConfigFile(filepath.Join(subDir, name), &rp); err != nil {
				return nil, fmt.Errorf("failed to load patterns/%s: %w", name, err)
			}
			repos = append(repos, rp)
		}
		return repos, nil
	}
	var rc ReposConfig
	if err := loadConfigFile(filepath.Join(configDir, "repos.yaml"), &rc); err != nil {
		return nil, fmt.Errorf("failed to load repos.yaml: %w", err)
	}
	return rc.Repos, nil
}

func loadBots(configDir string) ([]FeishuBot, error) {
	var bc FeishuBotsConfig
	if err := loadConfigFile(filepath.Join(configDir, "feishu-bots.yaml"), &bc); err != nil {
		return nil, fmt.Errorf("failed to load feishu-bots.yaml: %w", err)
	}
	return bc.FeishuBots, nil
}

func loadEvents(configDir string) (EventsConfig, error) {
	eventsDir := filepath.Join(configDir, "events")
	esDir := filepath.Join(eventsDir, "event_sets")
	defDir := filepath.Join(eventsDir, "definitions")

	esEntries, esErr := scanYAMLDir(esDir)
	defEntries, defErr := scanYAMLDir(defDir)

	if (esErr == nil && len(esEntries) > 0) || (defErr == nil && len(defEntries) > 0) {
		ec := EventsConfig{
			EventSets: make(map[string]map[string]any),
			Events:    make(map[string]any),
		}
		for _, name := range esEntries {
			setName := strings.TrimSuffix(name, ".yaml")
			var body map[string]any
			if err := loadConfigFile(filepath.Join(esDir, name), &body); err != nil {
				return ec, fmt.Errorf("failed to load events/event_sets/%s: %w", name, err)
			}
			ec.EventSets[setName] = body
		}
		for _, name := range defEntries {
			evName := strings.TrimSuffix(name, ".yaml")
			var body any
			if err := loadConfigFile(filepath.Join(defDir, name), &body); err != nil {
				return ec, fmt.Errorf("failed to load events/definitions/%s: %w", name, err)
			}
			ec.Events[evName] = body
		}
		return ec, nil
	}

	var ec EventsConfig
	if err := loadConfigFile(filepath.Join(configDir, "events.yaml"), &ec); err != nil {
		return EventsConfig{}, fmt.Errorf("failed to load events.yaml: %w", err)
	}
	if ec.EventSets == nil {
		ec.EventSets = make(map[string]map[string]any)
	}
	if ec.Events == nil {
		ec.Events = make(map[string]any)
	}
	return ec, nil
}

func loadTemplates(configDir string) (map[string]TemplatesConfig, error) {
	templatesDir := filepath.Join(configDir, "templates")
	entries, err := os.ReadDir(templatesDir)
	if err == nil && len(entries) > 0 {
		result := make(map[string]TemplatesConfig)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			locale := entry.Name()
			localeDir := filepath.Join(templatesDir, locale)
			tc := TemplatesConfig{Templates: make(map[string]EventTemplate)}

			files, ferr := os.ReadDir(localeDir)
			if ferr != nil {
				continue
			}
			for _, f := range files {
				if f.IsDir() {
					continue
				}
				ext := ""
				if strings.HasSuffix(f.Name(), ".json") {
					ext = ".json"
				} else if strings.HasSuffix(f.Name(), ".jsonc") {
					ext = ".jsonc"
				}
				if ext == "" {
					continue
				}
				eventName := strings.TrimSuffix(f.Name(), ext)
				var perEvent struct {
					Payloads []PayloadTemplate `json:"payloads"`
				}
				if err := loadConfigFile(filepath.Join(localeDir, f.Name()), &perEvent); err != nil {
					return nil, fmt.Errorf("failed to load templates/%s/%s: %w", locale, f.Name(), err)
				}
				tc.Templates[eventName] = EventTemplate{Payloads: perEvent.Payloads}
			}
			result[locale] = tc
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	// Fallback: legacy flat files
	result := make(map[string]TemplatesConfig)
	var defaultTmpl TemplatesConfig
	if err := loadConfigFile(filepath.Join(configDir, "templates.jsonc"), &defaultTmpl); err != nil {
		return nil, fmt.Errorf("failed to load templates.jsonc: %w", err)
	}
	result["default"] = defaultTmpl

	entries, err = os.ReadDir(configDir)
	if err != nil {
		return result, nil
	}
	templateRe := regexp.MustCompile(`^templates\.([a-zA-Z0-9_-]+)\.jsonc$`)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := templateRe.FindStringSubmatch(entry.Name())
		if len(matches) > 1 {
			var tmpl TemplatesConfig
			if err := loadConfigFile(filepath.Join(configDir, entry.Name()), &tmpl); err != nil {
				return nil, fmt.Errorf("failed to load %s: %w", entry.Name(), err)
			}
			result[matches[1]] = tmpl
		}
	}
	return result, nil
}

// ── Helpers ──

func scanYAMLDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

func (c *Config) GetBotTemplate(botAlias string) string {
	for _, bot := range c.FeishuBots.FeishuBots {
		if bot.Alias == botAlias {
			if bot.Template != "" {
				return bot.Template
			}
			return "default"
		}
	}
	return "default"
}

func (c *Config) GetTemplateConfig(templateName string) TemplatesConfig {
	if tmpl, exists := c.Templates[templateName]; exists {
		return tmpl
	}
	if tmpl, exists := c.Templates["default"]; exists {
		return tmpl
	}
	return TemplatesConfig{}
}

// loadConfigFile loads YAML, JSONC, or JSON based on extension.
// .jsonc: strips // and /* */ comments before JSON parsing.
// .json:  parsed directly (no comment stripping — prevents // inside URLs
//         from being misinterpreted as comments).
// .*:     YAML.
func loadConfigFile(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if strings.HasSuffix(path, ".jsonc") {
		cleaned := stripJSONCComments(string(data))
		if err := json.Unmarshal([]byte(cleaned), out); err != nil {
			if serr, ok := err.(*json.SyntaxError); ok {
				line, col := offsetToLineCol([]byte(cleaned), serr.Offset)
				return fmt.Errorf("%w at line %d column %d (offset %d)", err, line, col, serr.Offset)
			}
			return err
		}
		return nil
	}

	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, out); err != nil {
			if serr, ok := err.(*json.SyntaxError); ok {
				line, col := offsetToLineCol(data, serr.Offset)
				return fmt.Errorf("%w at line %d column %d (offset %d)", err, line, col, serr.Offset)
			}
			return err
		}
		return nil
	}

	return yaml.Unmarshal(data, out)
}

func offsetToLineCol(b []byte, offset int64) (int, int) {
	if offset <= 0 {
		return 1, 1
	}
	var line = 1
	var col = 1
	var i int64
	for i = 0; i < offset-1 && i < int64(len(b)); i++ {
		if b[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

func stripJSONCComments(s string) string {
	reBlock := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	s = reBlock.ReplaceAllString(s, "")
	reLine := regexp.MustCompile(`(?m)//.*$`)
	s = reLine.ReplaceAllString(s, "")
	return s
}
