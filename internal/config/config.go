package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Terminal         string   `mapstructure:"terminal"           json:"terminal"`
	PollInterval     string   `mapstructure:"poll_interval"      json:"poll_interval"`
	DefaultAgeFilter string   `mapstructure:"default_age_filter" json:"default_age_filter,omitempty"`
	AgePresets       []string `mapstructure:"age_presets"        json:"age_presets"`
	DefaultTool      string   `mapstructure:"default_tool"       json:"default_tool"`
	AutoSelectTool   bool     `mapstructure:"auto_select_tool"   json:"auto_select_tool"`
	NerdFont         *bool    `mapstructure:"nerd_font"          json:"nerd_font,omitempty"`
	OpencodePath     string   `mapstructure:"opencode_path"      json:"opencode_path"`
	CodexPath        string   `mapstructure:"codex_path"         json:"codex_path"`
	ClaudePath       string   `mapstructure:"claude_path"        json:"claude_path"`
}

func Init() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME/.config/ai-dash")
	viper.AddConfigPath(".")

	viper.SetDefault("terminal", "")
	viper.SetDefault("poll_interval", "10s")
	viper.SetDefault("default_age_filter", "14d")
	viper.SetDefault("default_tool", "")
	viper.SetDefault("auto_select_tool", false)
	viper.SetDefault("opencode_path", "")
	viper.SetDefault("codex_path", "")
	viper.SetDefault("claude_path", "")

	viper.SetEnvPrefix("AIDASH")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = viper.BindEnv("terminal", "TERMINAL", "AIDASH_TERMINAL")

	_ = viper.ReadInConfig()
}

func Load() Config {
	var cfg Config
	_ = viper.Unmarshal(&cfg)
	return cfg
}

func (c Config) PollDuration() time.Duration {
	if d, err := time.ParseDuration(c.PollInterval); err == nil && d >= time.Second {
		return d
	}
	return 10 * time.Second
}

func (c Config) DefaultAgeFilterDuration() time.Duration {
	return parseDurationWithDays(c.DefaultAgeFilter, 14*24*time.Hour)
}

func parseDurationWithDays(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%d", &days); err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return fallback
}

// AgeDurations parses configured age presets into durations.
func (c Config) AgeDurations() []time.Duration {
	if len(c.AgePresets) == 0 {
		return nil
	}
	var durations []time.Duration
	for _, s := range c.AgePresets {
		d := parseDurationWithDays(s, 0)
		if d > 0 {
			durations = append(durations, d)
		}
	}
	return durations
}

// SourcePath returns the configured path for a source, or the fallback default.
func (c Config) SourcePath(source, fallback string) string {
	switch source {
	case "opencode":
		if c.OpencodePath != "" {
			return c.OpencodePath
		}
	case "codex":
		if c.CodexPath != "" {
			return c.CodexPath
		}
	case "claude":
		if c.ClaudePath != "" {
			return c.ClaudePath
		}
	}
	return fallback
}

func GenerateSchema() string {
	schema := map[string]any{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"title":       "ai-dash configuration",
		"description": "Configuration for the ai-dash TUI",
		"type":        "object",
		"properties": map[string]any{
			"terminal": map[string]any{
				"type":        "string",
				"description": "Terminal emulator to use when opening sessions (e.g. ghostty, kitty, alacritty)",
				"default":     "",
			},
			"poll_interval": map[string]any{
				"type":        "string",
				"description": "How often to reload sessions (Go duration, e.g. 10s, 30s, 1m)",
				"default":     "10s",
			},
			"default_age_filter": map[string]any{
				"type":        "string",
				"description": "Default age filter applied to sessions on load and when clearing filters (e.g. 14d, 30d, 720h)",
				"default":     "14d",
			},
			"default_tool": map[string]any{
				"type":        "string",
				"description": "Default tool for new sessions (e.g. claude, codex, opencode)",
				"default":     "",
			},
			"auto_select_tool": map[string]any{
				"type":        "boolean",
				"description": "Skip tool picker and use default_tool automatically when creating new sessions",
				"default":     false,
			},
			"opencode_path": map[string]any{
				"type":        "string",
				"description": "Path to OpenCode SQLite database (default: ~/.local/share/opencode/opencode.db)",
				"default":     "",
			},
			"codex_path": map[string]any{
				"type":        "string",
				"description": "Path to Codex config directory (default: ~/.codex/config.toml)",
				"default":     "",
			},
			"claude_path": map[string]any{
				"type":        "string",
				"description": "Path to Claude Code projects directory (default: ~/.claude/projects)",
				"default":     "",
			},
			"nerd_font": map[string]any{
				"type":        "boolean",
				"description": "Use Nerd Font icons. Null/omitted means auto-detect via fc-list. Set false to force Unicode fallback.",
			},
			"age_presets": map[string]any{
				"type":        "array",
				"description": "Date range options cycled with D key (e.g. 1h, 1d, 3d, 7d, 14d, 30d)",
				"items":       map[string]any{"type": "string"},
				"default":     []string{"1h", "1d", "3d", "7d", "14d", "30d"},
			},
		},
		"additionalProperties": false,
	}
	data, _ := json.MarshalIndent(schema, "", "  ")
	return string(data)
}
