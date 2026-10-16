package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func labels(actions []Action) []string {
	out := make([]string, len(actions))
	for i, a := range actions {
		out[i] = a.Label
	}
	return out
}

func TestActionsForDefaults(t *testing.T) {
	got := strings.Join(labels(Config{}.ActionsFor("claude")), ",")
	if got != "artifact,continue,message,new with prompt" {
		t.Errorf("claude actions = %s", got)
	}
	got = strings.Join(labels(Config{}.ActionsFor("codex")), ",")
	if got != "continue,message,new with prompt" {
		t.Errorf("codex actions = %s", got)
	}
}

func TestActionsForOverrides(t *testing.T) {
	cfg := Config{Actions: map[string][]Action{
		GenericActions: {
			{Label: "continue", Key: "ctrl+r", Prompt: "keep going"},
			{Label: "message", Disabled: true},
			{Label: "no prompt", Key: "ctrl+p"},
		},
		"codex": {
			{Label: "review", Key: "ctrl+r", Prompt: "review the diff"},
		},
	}}

	generic := cfg.ActionsFor("claude")
	if got := strings.Join(labels(generic), ","); got != "artifact,continue,new with prompt" {
		t.Fatalf("claude actions = %s", got)
	}
	if generic[1].Prompt != "keep going" {
		t.Errorf("continue prompt = %q, want the configured one", generic[1].Prompt)
	}

	// The codex action takes ctrl+r, so the generic continue is hidden.
	codex := cfg.ActionsFor("codex")
	if got := strings.Join(labels(codex), ","); got != "review,new with prompt" {
		t.Errorf("codex actions = %s", got)
	}
}

func TestActionsFromConfigFile(t *testing.T) {
	v := viper.New()
	v.SetConfigType("json")
	err := v.ReadConfig(strings.NewReader(`{
		"actions": {
			"pi": [{"label": "plan", "key": "ctrl+o", "mode": "new", "prompt": "Plan {{.Input}}", "input_optional": true}]
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	actions := cfg.ActionsFor("pi")
	if len(actions) == 0 || actions[0].Label != "plan" {
		t.Fatalf("pi actions = %v", labels(actions))
	}
	a := actions[0]
	if a.Key != "ctrl+o" || !a.IsNew() || !a.NeedsInput() || !a.InputOptional {
		t.Errorf("plan action = %+v", a)
	}
}
