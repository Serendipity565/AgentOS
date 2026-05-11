package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the root runtime configuration loaded from config.yaml.
type Config struct {
	LLM       LLMConfig       `yaml:"llm"`
	Transport TransportConfig `yaml:"transport"`
	Skills    []SkillConfig   `yaml:"skills"`
	MCP       []MCPConfig     `yaml:"mcp"`
}

type LLMConfig struct {
	Active    string              `yaml:"active"`
	Providers []LLMProviderConfig `yaml:"providers"`
}

type LLMProviderConfig struct {
	Name     string `yaml:"name"`
	Provider string `yaml:"provider"`
	Driver   string `yaml:"driver"`
	BaseURL  string `yaml:"base_url"`
	APIKey   string `yaml:"api_key"`
	Model    string `yaml:"model"`
	Enabled  bool   `yaml:"enabled"`
}

type TransportConfig struct {
	Web WebConfig `yaml:"web"`
}

type WebConfig struct {
	Addr string `yaml:"addr"`
}

type SkillConfig struct {
	Name    string         `yaml:"name"`
	Enabled bool           `yaml:"enabled"`
	Options map[string]any `yaml:"options"`
}

type MCPConfig struct {
	Name    string         `yaml:"name"`
	Enabled bool           `yaml:"enabled"`
	Options map[string]any `yaml:"options"`
}

// Load merges file contents onto a default config so optional fields can stay
// omitted in YAML while the application still receives stable defaults.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}

	cfg := defaultConfig()
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}

	cfg.applyDefaults()

	return cfg, nil
}

// defaultConfig centralizes fallback values used when a field is absent from
// the YAML file.
func defaultConfig() Config {
	return Config{
		LLM: LLMConfig{
			Active: "mock",
			Providers: []LLMProviderConfig{
				{
					Name:     "mock",
					Provider: "mock",
					Driver:   "mock",
					Enabled:  true,
				},
			},
		},
		Transport: TransportConfig{
			Web: WebConfig{
				Addr: ":8080",
			},
		},
		Skills: []SkillConfig{},
		MCP:    []MCPConfig{},
	}
}

func (c *LLMConfig) UnmarshalYAML(node *yaml.Node) error {
	type modernLLMConfig LLMConfig

	var modern modernLLMConfig
	if err := node.Decode(&modern); err == nil && (len(modern.Providers) > 0 || modern.Active != "") {
		*c = LLMConfig(modern)
		return nil
	}

	var legacy []LLMProviderConfig
	if err := node.Decode(&legacy); err == nil {
		c.Providers = legacy
		return nil
	}

	return fmt.Errorf("invalid llm config: expected mapping or list")
}

func (c *Config) applyDefaults() {
	c.LLM.applyDefaults()
}

func (c *LLMConfig) applyDefaults() {
	if len(c.Providers) == 0 {
		c.Providers = defaultConfig().LLM.Providers
	}

	for i := range c.Providers {
		provider := &c.Providers[i]

		if strings.TrimSpace(provider.Name) == "" {
			provider.Name = provider.Provider
		}
		if strings.TrimSpace(provider.Driver) == "" {
			provider.Driver = inferDriver(provider.Provider)
		}
	}

	if strings.TrimSpace(c.Active) == "" && len(c.Providers) > 0 {
		c.Active = c.Providers[0].Name
	}
}

func (c LLMConfig) ActiveProvider() (LLMProviderConfig, error) {
	if len(c.Providers) == 0 {
		return LLMProviderConfig{}, fmt.Errorf("no llm providers configured")
	}

	active := strings.TrimSpace(c.Active)
	if active == "" {
		for _, provider := range c.Providers {
			if provider.Enabled {
				return provider, nil
			}
		}
		return c.Providers[0], nil
	}

	for _, provider := range c.Providers {
		if provider.Name == active {
			if !provider.Enabled {
				return LLMProviderConfig{}, fmt.Errorf("llm active provider %q is disabled", active)
			}
			return provider, nil
		}
	}

	return LLMProviderConfig{}, fmt.Errorf("llm active provider %q not found", active)
}

func inferDriver(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "mock":
		return "mock"
	default:
		return "openai-compatible"
	}
}
