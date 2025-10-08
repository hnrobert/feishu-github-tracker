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
	Templates  TemplatesConfig
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
	Alias string `yaml:"alias"`
	URL   string `yaml:"url"`
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
	cfg := &Config{}

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

	// Load templates.jsonc (JSONC is required)
	templatesJSONC := filepath.Join(configDir, "templates.jsonc")
	if err := loadConfigFile(templatesJSONC, &cfg.Templates); err != nil {
		return nil, fmt.Errorf("failed to load templates.jsonc: %w", err)
	}

	return cfg, nil
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
		return json.Unmarshal([]byte(cleaned), out)
	}

	// default to YAML
	return yaml.Unmarshal(data, out)
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
