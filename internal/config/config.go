package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration
type Config struct {
	Server     ServerConfig
	Repos      ReposConfig
	Events     EventsConfig
	FeishuBots FeishuBotsConfig
	Templates  map[string]TemplatesConfig // Key: template name (e.g., "default", "cn")
}

// ServerConfig represents server.yaml
type ServerConfig struct {
	Server struct {
		Host           string `yaml:"host"`
		Port           int    `yaml:"port"`
		Secret         string `yaml:"secret"`
		LogLevel       string `yaml:"log_level"`
		MaxPayloadSize string `yaml:"max_payload_size"`
		Timeout        int    `yaml:"timeout"`
	} `yaml:"server"`
	AllowedSources []string `yaml:"allowed_sources"`
}

// ReposConfig represents repos.yaml
type ReposConfig struct {
	Repos []RepoPattern `yaml:"repos"`
}

type RepoPattern struct {
	Pattern  string         `yaml:"pattern"`
	Events   map[string]any `yaml:"events"`
	NotifyTo []string       `yaml:"notify_to"`
}

// EventsConfig represents events.yaml
type EventsConfig struct {
	EventSets map[string]map[string]any `yaml:"event_sets"`
	Events    map[string]any            `yaml:"events"`
}

// FeishuBotsConfig represents feishu-bots.yaml
type FeishuBotsConfig struct {
	FeishuBots []FeishuBot `yaml:"feishu_bots"`
}

type FeishuBot struct {
	Alias    string `yaml:"alias"`
	URL      string `yaml:"url"`
	Template string `yaml:"template"` // Optional: template name (e.g., "cn"), defaults to "default"
}

// TemplatesConfig represents templates.jsonc (JSONC)
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

// Load loads all configuration files from the given directory
func Load(configDir string) (*Config, error) {
	cfg := &Config{
		Templates: make(map[string]TemplatesConfig),
	}

	// Load server.yaml
	if err := loadConfigFile(filepath.Join(configDir, "server.yaml"), &cfg.Server); err != nil {
		return nil, fmt.Errorf("failed to load server.yaml: %w", err)
	}

	// Load repos.yaml
	if err := loadConfigFile(filepath.Join(configDir, "repos.yaml"), &cfg.Repos); err != nil {
		return nil, fmt.Errorf("failed to load repos.yaml: %w", err)
	}

	// Load events.yaml
	if err := loadConfigFile(filepath.Join(configDir, "events.yaml"), &cfg.Events); err != nil {
		return nil, fmt.Errorf("failed to load events.yaml: %w", err)
	}

	// Load feishu-bots.yaml
	if err := loadConfigFile(filepath.Join(configDir, "feishu-bots.yaml"), &cfg.FeishuBots); err != nil {
		return nil, fmt.Errorf("failed to load feishu-bots.yaml: %w", err)
	}

	// Load templates.jsonc as default template (required)
	defaultTemplatesPath := filepath.Join(configDir, "templates.jsonc")
	var defaultTemplates TemplatesConfig
	if err := loadConfigFile(defaultTemplatesPath, &defaultTemplates); err != nil {
		return nil, fmt.Errorf("failed to load templates.jsonc: %w", err)
	}
	cfg.Templates["default"] = defaultTemplates

	// Load additional template files (templates.*.jsonc)
	// Scan for templates.cn.jsonc, templates.en.jsonc, etc.
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	templatePattern := regexp.MustCompile(`^templates\.([a-zA-Z0-9_-]+)\.jsonc$`)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := templatePattern.FindStringSubmatch(entry.Name())
		if len(matches) > 1 {
			templateName := matches[1]
			var tmpl TemplatesConfig
			templatePath := filepath.Join(configDir, entry.Name())
			if err := loadConfigFile(templatePath, &tmpl); err != nil {
				return nil, fmt.Errorf("failed to load %s: %w", entry.Name(), err)
			}
			cfg.Templates[templateName] = tmpl
		}
	}

	return cfg, nil
}

// GetBotTemplate returns the template name for a given bot alias
// Returns "default" if the bot doesn't specify a template or if the bot is not found
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

// GetTemplateConfig returns the template configuration for a given template name
// Returns the default template if the specified template is not found
func (c *Config) GetTemplateConfig(templateName string) TemplatesConfig {
	if tmpl, exists := c.Templates[templateName]; exists {
		return tmpl
	}
	// Fallback to default
	if tmpl, exists := c.Templates["default"]; exists {
		return tmpl
	}
	// Return empty config if even default is missing (shouldn't happen)
	return TemplatesConfig{}
}

// loadConfigFile loads either YAML or JSONC (JSON with comments) based on file extension
func loadConfigFile(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if strings.HasSuffix(path, ".jsonc") || strings.HasSuffix(path, ".json") {
		// strip comments and unmarshal as JSON
		cleaned := stripJSONCComments(string(data))
		if err := json.Unmarshal([]byte(cleaned), out); err != nil {
			// If it's a syntax error, try to compute line/column
			if serr, ok := err.(*json.SyntaxError); ok {
				line, col := offsetToLineCol([]byte(cleaned), serr.Offset)
				return fmt.Errorf("%w at line %d column %d (offset %d)", err, line, col, serr.Offset)
			}
			return err
		}
		return nil
	}

	// default to YAML
	return yaml.Unmarshal(data, out)
}

// offsetToLineCol converts a 1-based byte offset into line and column numbers (1-based)
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

// stripJSONCComments removes // and /* */ style comments from JSONC input.
func stripJSONCComments(s string) string {
	// Remove /* ... */ block comments
	reBlock := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	s = reBlock.ReplaceAllString(s, "")
	// Remove // line comments
	reLine := regexp.MustCompile(`(?m)//.*$`)
	s = reLine.ReplaceAllString(s, "")
	return s
}
