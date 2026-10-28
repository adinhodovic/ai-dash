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
	Multiplexer      string   `mapstructure:"multiplexer"        json:"multiplexer,omitempty"`
	TmuxMode         string   `mapstructure:"tmux_mode"          json:"tmux_mode,omitempty"`
	PollInterval     string   `mapstructure:"poll_interval"      json:"poll_interval"`
	DefaultAgeFilter string   `mapstructure:"default_age_filter" json:"default_age_filter,omitempty"`
	AgePresets       []string `mapstructure:"age_presets"        json:"age_presets"`
	DefaultTool      string   `mapstructure:"default_tool"       json:"default_tool"`
	AutoSelectTool   bool     `mapstructure:"auto_select_tool"   json:"auto_select_tool"`
	NerdFont         *bool    `mapstructure:"nerd_font"          json:"nerd_font,omitempty"`
	OpencodePath     string   `mapstructure:"opencode_path"      json:"opencode_path"`
	CodexPath        string   `mapstructure:"codex_path"         json:"codex_path"`
	ClaudePath       string   `mapstructure:"claude_path"        json:"claude_path"`
	PiPath           string   `mapstructure:"pi_path"            json:"pi_path"`
	RenameAgent      string   `mapstructure:"rename_agent"       json:"rename_agent,omitempty"`

	Actions map[string][]Action `mapstructure:"actions" json:"actions,omitempty"`
}

func Init() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME/.config/ai-dash")
	viper.AddConfigPath(".")

	viper.SetDefault("terminal", "")
	viper.SetDefault("multiplexer", "auto")
	viper.SetDefault("tmux_mode", "terminal")
	viper.SetDefault("poll_interval", "5s")
	viper.SetDefault("default_age_filter", "14d")
	viper.SetDefault("default_tool", "")
	viper.SetDefault("auto_select_tool", false)
	viper.SetDefault("opencode_path", "")
	viper.SetDefault("codex_path", "")
	viper.SetDefault("claude_path", "")
	viper.SetDefault("pi_path", "")
	viper.SetDefault("rename_agent", "")

	viper.SetEnvPrefix("AIDASH")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = viper.BindEnv("terminal", "TERMINAL", "AIDASH_TERMINAL")
	_ = viper.BindEnv("multiplexer", "AIDASH_MULTIPLEXER")

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
	return 5 * time.Second
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
	case "pi":
		if c.PiPath != "" {
			return c.PiPath
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
				"description": "Terminal emulator to use when opening sessions (e.g. ghostty, kitty, alacritty, wezterm, foot). Defaults to $TERMINAL, then the first known terminal found on PATH.",
				"default":     "",
			},
			"multiplexer": map[string]any{
				"type":        "string",
				"description": "Terminal multiplexer used to open sessions when ai-dash runs inside one. 'auto' detects tmux/zellij from $TMUX/$ZELLIJ; 'tmux' or 'zellij' force a specific one; 'off' disables multiplexer use.",
				"enum":        []string{"auto", "tmux", "zellij", "off"},
				"default":     "auto",
			},
			"tmux_mode": map[string]any{
				"type":        "string",
				"description": "How sessions open under tmux. 'window' opens a new window and switches to it; 'background' opens a new window without switching; 'terminal' opens a new terminal emulator window running the session in its own tmux session.",
				"enum":        []string{"terminal", "window", "background"},
				"default":     "terminal",
			},
			"poll_interval": map[string]any{
				"type":        "string",
				"description": "How often to reload sessions (Go duration, e.g. 5s, 30s, 1m)",
				"default":     "5s",
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
				"description": "Path to OpenCode SQLite database (default: ~/.local/share/opencode/opencode.db, or ~/Library/Application Support/opencode/opencode.db on macOS)",
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
			"pi_path": map[string]any{
				"type":        "string",
				"description": "Path to pi agent sessions directory (default: ~/.pi/agent/sessions)",
				"default":     "",
			},
			"rename_agent": map[string]any{
				"type":        "string",
				"description": "Agent that writes titles for autorename (T key, `ai-dash autorename`). Omit to use the first of claude, codex, pi found on PATH.",
				"enum":        []string{"claude", "codex", "pi"},
			},
			"nerd_font": map[string]any{
				"type":        "boolean",
				"description": "Use Nerd Font icons. Null/omitted means auto-detect via fc-list. Set false to force Unicode fallback.",
			},
			"age_presets": map[string]any{
				"type":        "array",
				"description": "Date range options cycled with D key (e.g. 15m, 1h, 1d, 3d, 7d, 14d, 30d)",
				"items":       map[string]any{"type": "string"},
				"default":     []string{"15m", "1h", "1d", "3d", "7d", "14d", "30d"},
			},
			"actions": actionsSchema(),
		},
		"$defs":                map[string]any{"action": actionSchema()},
		"additionalProperties": false,
	}
	data, _ := json.MarshalIndent(schema, "", "  ")
	return string(data)
}

func actionSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"key": map[string]any{
				"type":        "string",
				"description": "Key that runs the action directly (e.g. ctrl+r). Optional; every action is also listed under x. Keys used by ai-dash itself are ignored.",
			},
			"label": map[string]any{
				"type":        "string",
				"description": "Name shown in the action picker. A configured action replaces the built-in action with the same label.",
			},
			"mode": map[string]any{
				"type":        "string",
				"description": "'resume' sends the prompt to the selected session; 'new' starts a new session in its project.",
				"enum":        []string{ActionModeResume, ActionModeNew},
				"default":     ActionModeResume,
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "Prompt template. Fields: {{.Input}}, {{.Project}}, {{.SessionID}}, {{.Tool}}, {{.Summary}}, {{.Model}}. Using {{.Input}} asks for text first.",
			},
			"input_label": map[string]any{
				"type":        "string",
				"description": "Label shown while asking for {{.Input}}",
			},
			"input_optional": map[string]any{
				"type":        "boolean",
				"description": "Run even when the input is left empty",
				"default":     false,
			},
			"disabled": map[string]any{
				"type":        "boolean",
				"description": "Remove the built-in action with the same label",
				"default":     false,
			},
		},
		"required":             []string{"label"},
		"additionalProperties": false,
	}
}

func actionsSchema() map[string]any {
	scopes := map[string]any{}
	for _, scope := range ActionScopes {
		desc := "Actions for " + scope + " sessions"
		if scope == GenericActions {
			desc = "Actions for every tool"
		}
		scopes[scope] = map[string]any{
			"type":        "array",
			"description": desc,
			"items":       map[string]any{"$ref": "#/$defs/action"},
		}
	}
	return map[string]any{
		"type":                 "object",
		"description":          "Prompts to run against the selected session, from the x picker or their own key. Tool actions take precedence over generic ones.",
		"properties":           scopes,
		"additionalProperties": false,
	}
}
