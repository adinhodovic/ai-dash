package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Terminal     string `mapstructure:"terminal"     json:"terminal"`
	PollInterval string `mapstructure:"poll_interval" json:"poll_interval"`
	MaxAge       string `mapstructure:"max_age"       json:"max_age"`
}

func Init() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME/.config/ai-dash")
	viper.AddConfigPath(".")

	viper.SetDefault("terminal", "")
	viper.SetDefault("poll_interval", "10s")
	viper.SetDefault("max_age", "14d")

	viper.SetEnvPrefix("AIDASH")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// $TERMINAL maps to terminal config key
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

func (c Config) MaxAgeDuration() time.Duration {
	return parseDurationWithDays(c.MaxAge, 14*24*time.Hour)
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
			"max_age": map[string]any{
				"type":        "string",
				"description": "Maximum age of sessions to display (e.g. 14d, 30d, 720h)",
				"default":     "14d",
			},
		},
		"additionalProperties": false,
	}
	data, _ := json.MarshalIndent(schema, "", "  ")
	return string(data)
}
