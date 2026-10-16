package ui

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/adinhodovic/ai-dash/internal/config"
	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources"
	uiutil "github.com/adinhodovic/ai-dash/internal/ui/util"
)

// pendingAction is an action waiting for its {{.Input}} text.
type pendingAction struct {
	action  config.Action
	session session.Session
}

// actionPrompt is the data an action's prompt template is rendered with.
type actionPrompt struct {
	Input     string
	Project   string
	SessionID string
	Tool      string
	Summary   string
	Model     string
}

func renderActionPrompt(a config.Action, s session.Session, input string) (string, error) {
	tmpl, err := template.New(a.Label).Parse(a.Prompt)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	err = tmpl.Execute(&b, actionPrompt{
		Input:     input,
		Project:   s.Project,
		SessionID: s.ID,
		Tool:      s.Tool,
		Summary:   s.Summary,
		Model:     s.Model,
	})
	return strings.TrimSpace(b.String()), err
}

func selectedSession(filtered []session.Session, cursor int) (session.Session, bool) {
	if cursor < 0 || cursor >= len(filtered) {
		return session.Session{}, false
	}
	return filtered[cursor], true
}

// sessionActions returns the actions for s's tool, minus direct keys that
// ai-dash already binds; those actions stay reachable from the picker.
func (m Model) sessionActions(s session.Session) []config.Action {
	actions := m.meta.Config.ActionsFor(s.Tool)
	reserved := m.keys.boundKeys()
	for i := range actions {
		if reserved[actions[i].Key] {
			actions[i].Key = ""
		}
	}
	return actions
}

func (m Model) actionForKey(k string, filtered []session.Session) (config.Action, bool) {
	s, ok := selectedSession(filtered, m.sessionCursor)
	if !ok {
		return config.Action{}, false
	}
	for _, a := range m.sessionActions(s) {
		if a.Key != "" && a.Key == k {
			return a, true
		}
	}
	return config.Action{}, false
}

// actionBindings lists the selected session's direct action keys for the
// help overlay.
func (m Model) actionBindings(filtered []session.Session) []key.Binding {
	s, ok := selectedSession(filtered, m.sessionCursor)
	if !ok {
		return nil
	}
	var bindings []key.Binding
	for _, a := range m.sessionActions(s) {
		if a.Key != "" {
			b := key.NewBinding(key.WithKeys(a.Key), key.WithHelp(a.Key, a.Label))
			bindings = append(bindings, b)
		}
	}
	return bindings
}

func (m *Model) openActionPicker(filtered []session.Session) {
	s, ok := selectedSession(filtered, m.sessionCursor)
	if !ok {
		m.statusMessage = "No session selected"
		return
	}
	actions := m.sessionActions(s)
	if len(actions) == 0 {
		m.statusMessage = fmt.Sprintf("No actions for %s", s.Tool)
		return
	}
	items := make([]list.Item, 0, len(actions))
	for i, a := range actions {
		display := a.Label
		if a.Key != "" {
			display += " (" + a.Key + ")"
		}
		if a.IsNew() {
			display += " · new session"
		}
		items = append(items, pickerItem{value: strconv.Itoa(i), display: display})
	}
	m.showSources = false
	m.showHelp = false
	m.pickerActions = actions
	m.picker = filterPicker{
		active: true,
		label:  "action",
		list:   newPickerList(fmt.Sprintf("Actions: %s", s.Tool), items, true),
	}
}

// pickedAction returns the action behind a picker item.
func (m Model) pickedAction(item pickerItem) (config.Action, bool) {
	i, err := strconv.Atoi(item.value)
	if err != nil || i < 0 || i >= len(m.pickerActions) {
		return config.Action{}, false
	}
	return m.pickerActions[i], true
}

// startAction runs a against the selected session, first asking for input
// when its prompt uses {{.Input}}.
func (m *Model) startAction(a config.Action, filtered []session.Session) tea.Cmd {
	s, ok := selectedSession(filtered, m.sessionCursor)
	if !ok {
		m.statusMessage = "No session selected"
		return nil
	}
	if !a.NeedsInput() {
		return m.runAction(a, s, "")
	}
	label := a.InputLabel
	if label == "" {
		label = uiutil.Capitalize(a.Label)
	}
	m.pending = &pendingAction{action: a, session: s}
	m.actionInput.SetValue("")
	m.actionInput.Placeholder = fmt.Sprintf("%s for %s", label, s.Tool)
	// Scroll within the top bar's search cell instead of being truncated.
	m.actionInput.SetWidth(max(10, searchCellWidth-len(label)-2))
	m.showHelp = false
	m.showSources = false
	m.picker.active = false
	return m.actionInput.Focus()
}

func (m *Model) runAction(a config.Action, s session.Session, input string) tea.Cmd {
	cfg := m.meta.Config
	target := s
	redirected := false
	if !a.IsNew() {
		var ok bool
		target, redirected, ok = m.resumeTarget(s)
		if !ok {
			return nil
		}
	}
	args, err := actionArgs(cfg, a, target, input)
	if err != nil {
		m.statusMessage = err.Error()
		return nil
	}
	cmd := spawnSession(cfg, args)
	if cmd == nil {
		m.statusMessage = "Set $TERMINAL or run inside tmux/zellij to open sessions"
		return nil
	}
	m.statusMessage = fmt.Sprintf("Running %s on %s session...", a.Label, target.Tool)
	if redirected {
		m.statusMessage = "Subagents can't be resumed directly — " + m.statusMessage
	}
	label := target.Tool
	if a.IsNew() {
		label = "new " + label
	}
	return launchSession(cmd, label)
}

// actionArgs builds the shell command that opens s with a's prompt.
func actionArgs(
	cfg config.Config,
	a config.Action,
	s session.Session,
	input string,
) ([]string, error) {
	prompt, err := renderActionPrompt(a, s, input)
	if err != nil {
		return nil, fmt.Errorf("action %s: bad prompt: %w", a.Label, err)
	}
	if prompt == "" {
		return nil, fmt.Errorf("action %s has an empty prompt", a.Label)
	}
	dir := sessionDir(s)
	var args []string
	if a.IsNew() {
		if dir == "" {
			return nil, fmt.Errorf("no project directory for the selected session")
		}
		args = sources.NewSessionArgs(cfg, s.Tool, dir)
	} else {
		args = sources.ResumeArgs(cfg, s.Tool, s.ID, dir)
	}
	promptArgs := sources.PromptArgs(cfg, s.Tool, prompt)
	if len(args) == 0 || len(promptArgs) == 0 {
		return nil, fmt.Errorf("no action support for %s", s.Tool)
	}
	return append(args, promptArgs...), nil
}

func (m Model) updateActionInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.pending = nil
		m.actionInput.Blur()
		m.statusMessage = "Action cancelled"
	case "enter":
		p := m.pending
		input := strings.TrimSpace(m.actionInput.Value())
		if input == "" && !p.action.InputOptional {
			m.statusMessage = fmt.Sprintf("%s needs input", uiutil.Capitalize(p.action.Label))
			return m, nil
		}
		m.pending = nil
		m.actionInput.Blur()
		return m, m.runAction(p.action, p.session, input)
	default:
		m.actionInput, cmd = m.actionInput.Update(msg)
	}
	return m, cmd
}
