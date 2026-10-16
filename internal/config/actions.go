package config

import "strings"

const (
	ActionModeResume = "resume"
	ActionModeNew    = "new"

	// GenericActions is the actions scope that applies to every tool.
	GenericActions = "generic"
)

// ActionScopes are the keys accepted under "actions" in the config file.
var ActionScopes = []string{GenericActions, "claude", "codex", "opencode", "pi"}

// Action launches the selected session's tool with a prompt. Prompt is a
// text/template over the selected session; a prompt that uses {{.Input}}
// asks for that text before running.
type Action struct {
	Key           string `mapstructure:"key"            json:"key,omitempty"`
	Label         string `mapstructure:"label"          json:"label"`
	Mode          string `mapstructure:"mode"           json:"mode,omitempty"`
	Prompt        string `mapstructure:"prompt"         json:"prompt"`
	InputLabel    string `mapstructure:"input_label"    json:"input_label,omitempty"`
	InputOptional bool   `mapstructure:"input_optional" json:"input_optional,omitempty"`
	Disabled      bool   `mapstructure:"disabled"       json:"disabled,omitempty"`
}

// NeedsInput reports whether the prompt asks for text before running.
func (a Action) NeedsInput() bool {
	return strings.Contains(a.Prompt, ".Input")
}

// IsNew reports whether the action starts a new session instead of resuming.
func (a Action) IsNew() bool {
	return a.Mode == ActionModeNew
}

var defaultActions = map[string][]Action{
	GenericActions: {
		{Key: "ctrl+r", Label: "continue", Prompt: "continue"},
		{Key: "ctrl+e", Label: "message", Prompt: "{{.Input}}", InputLabel: "Message"},
		{
			Key:        "ctrl+n",
			Label:      "new with prompt",
			Mode:       ActionModeNew,
			Prompt:     "{{.Input}}",
			InputLabel: "Prompt",
		},
	},
	"claude": {
		{
			Label:      "artifact",
			Prompt:     "Create an artifact that summarizes {{.Input}}",
			InputLabel: "Summarize",
		},
	},
}

// ActionsFor returns the actions available for tool, its own first and then
// the generic ones. A configured action replaces the default with the same
// label in the same scope, and disabled removes it. A tool action hides a
// generic action with the same key or label.
func (c Config) ActionsFor(tool string) []Action {
	own := mergeActions(defaultActions[tool], c.Actions[tool])
	generic := mergeActions(defaultActions[GenericActions], c.Actions[GenericActions])
	out := own
	for _, g := range generic {
		hidden := false
		for _, a := range own {
			if a.Label == g.Label || (g.Key != "" && a.Key == g.Key) {
				hidden = true
				break
			}
		}
		if !hidden {
			out = append(out, g)
		}
	}
	return out
}

func mergeActions(defaults, configured []Action) []Action {
	out := make([]Action, 0, len(defaults)+len(configured))
	out = append(out, defaults...)
	for _, a := range configured {
		idx := -1
		for i, d := range out {
			if d.Label == a.Label {
				idx = i
				break
			}
		}
		switch {
		case idx >= 0 && a.Disabled:
			out = append(out[:idx], out[idx+1:]...)
		case idx >= 0:
			out[idx] = a
		case !a.Disabled:
			out = append(out, a)
		}
	}
	valid := out[:0]
	for _, a := range out {
		if a.Label != "" && strings.TrimSpace(a.Prompt) != "" {
			valid = append(valid, a)
		}
	}
	return valid
}
