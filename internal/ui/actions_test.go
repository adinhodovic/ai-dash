package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/adinhodovic/ai-dash/internal/config"
	"github.com/adinhodovic/ai-dash/internal/session"
)

func sendCtrlKey(m Model, r rune) Model {
	updated, _ := m.Update(tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl})
	return updated.(Model)
}

func TestActionArgs(t *testing.T) {
	s := session.Session{ID: "abc", Tool: "claude", Project: "/work/app", Summary: "auth"}
	tests := []struct {
		name   string
		tool   string
		action config.Action
		input  string
		want   string
	}{
		{
			name:   "resume with prompt",
			tool:   "claude",
			action: config.Action{Label: "continue", Prompt: "continue"},
			want:   "cd '/work/app' && claude --resume 'abc' 'continue'",
		},
		{
			name:   "input is quoted",
			tool:   "codex",
			action: config.Action{Label: "message", Prompt: "{{.Input}}"},
			input:  "don't $(rm) it",
			want:   `cd '/work/app' && codex resume 'abc' 'don'"'"'t $(rm) it'`,
		},
		{
			name:   "opencode uses --prompt",
			tool:   "opencode",
			action: config.Action{Label: "continue", Prompt: "continue"},
			want:   "cd '/work/app' && opencode -s 'abc' --prompt 'continue'",
		},
		{
			name: "new session with session fields",
			tool: "pi",
			action: config.Action{
				Label:  "follow up",
				Mode:   config.ActionModeNew,
				Prompt: "Follow up on {{.Summary}} in {{.Project}}",
			},
			want: "cd '/work/app' && pi 'Follow up on auth in /work/app'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := s
			s.Tool = tt.tool
			args, err := actionArgs(config.Config{}, tt.action, s, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(args, " "); got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestActionArgsErrors(t *testing.T) {
	s := session.Session{ID: "abc", Tool: "claude", Project: "/work/app"}
	bad := []config.Action{
		{Label: "broken", Prompt: "{{.Input"},
		{Label: "unknown field", Prompt: "{{.Nope}}"},
		{Label: "empty", Prompt: "{{.Input}}"},
	}
	for _, a := range bad {
		if _, err := actionArgs(config.Config{}, a, s, ""); err == nil {
			t.Errorf("%s: expected an error", a.Label)
		}
	}
	s.Tool = "unknown"
	hi := config.Action{Label: "x", Prompt: "hi"}
	if _, err := actionArgs(config.Config{}, hi, s, ""); err == nil {
		t.Error("unknown tool: expected an error")
	}
}

func TestActionKeyAsksForInput(t *testing.T) {
	m := resize(testModel(), 160, 50)
	m = sendCtrlKey(m, 'e')
	if m.pending == nil || m.pending.action.Label != "message" {
		t.Fatalf("ctrl+e should wait for input, pending = %+v", m.pending)
	}
	if !strings.Contains(m.View().Content, "Message") {
		t.Error("input label should be shown in the top bar")
	}

	m = sendNamedKey(m, "enter")
	if m.pending == nil || !strings.Contains(m.statusMessage, "needs input") {
		t.Errorf("empty input should keep waiting, status = %q", m.statusMessage)
	}

	m = sendNamedKey(m, "esc")
	if m.pending != nil || m.statusMessage != "Action cancelled" {
		t.Errorf("esc should cancel, status = %q", m.statusMessage)
	}
}

func TestActionsBuiltinKeysWin(t *testing.T) {
	m := testModel()
	m.meta.Config.Actions = map[string][]config.Action{
		config.GenericActions: {{Label: "shadow", Key: "r", Prompt: "hi"}},
	}
	if _, ok := m.actionForKey("r", m.filteredSessions()); ok {
		t.Error("an action must not take over a built-in key")
	}
}

func TestActionPicker(t *testing.T) {
	m := resize(testModel(), 160, 50)
	m = sendKey(m, "x")
	if !m.picker.active || m.picker.label != "action" {
		t.Fatal("x should open the action picker")
	}
	tool := m.filteredSessions()[m.sessionCursor].Tool
	if len(m.pickerActions) != len(m.meta.Config.ActionsFor(tool)) {
		t.Errorf("picker lists %d actions for %s", len(m.pickerActions), tool)
	}
	if !strings.Contains(m.View().Content, "Actions: "+tool) {
		t.Error("picker title should name the tool")
	}
}

func TestHelpListsActionsByScope(t *testing.T) {
	m := resize(testModel(), 220, 60)
	m.meta.Config.Actions = map[string][]config.Action{
		config.GenericActions: {{Label: "long key", Key: "ctrl+shift+r", Prompt: "hi"}},
		"codex":               {{Label: "review", Key: "r", Prompt: "review"}},
	}
	m = sendKey(m, "?")
	view := m.View().Content
	for _, want := range []string{
		"Generic actions", "ctrl+r", "continue", "ctrl+shift+r",
		"Claude actions", "artifact",
		"Codex actions", "review",
		"x picks from all of them",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("help overlay is missing %q", want)
		}
	}
	if strings.Contains(view, "Opencode actions") {
		t.Error("tools without their own actions should not get a section")
	}
	generic := strings.Index(view, "Generic actions")
	claude := strings.Index(view, "Claude actions")
	codex := strings.Index(view, "Codex actions")
	if generic > claude || claude > codex {
		t.Error("generic actions should come first, then each tool")
	}
}
