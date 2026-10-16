package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/adinhodovic/ai-dash/internal/config"
	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources"
	uiutil "github.com/adinhodovic/ai-dash/internal/ui/util"
)

func findSessionByID(sessions []session.Session, id string) (session.Session, bool) {
	for _, s := range sessions {
		if s.ID == id {
			return s, true
		}
	}
	return session.Session{}, false
}

func sessionDir(s session.Session) string {
	if s.Repo != "" && s.Repo != "/" {
		return s.Repo
	}
	if s.Project != "" && (strings.HasPrefix(s.Project, "/") || strings.HasPrefix(s.Project, "~")) {
		return s.Project
	}
	return ""
}

func sessionCommand(s session.Session, cfg config.Config) *exec.Cmd {
	args := sources.ResumeArgs(cfg, s.Tool, s.ID, sessionDir(s))
	if len(args) == 0 {
		return nil
	}
	return spawnSession(cfg, args)
}

// spawnSession picks a multiplexer if one is detected, otherwise falls back to
// launching a fresh terminal emulator. Returns nil when neither is available.
func spawnSession(cfg config.Config, args []string) *exec.Cmd {
	terminal := resolveTerminal(cfg.Terminal, exec.LookPath)
	kind := detectMultiplexer(cfg.Multiplexer, nil)
	if kind == multiplexerTmux && cfg.TmuxMode == tmuxModeTerminal && terminal != "" {
		name, argv := tmuxTerminalCommand(terminal, args)
		cmd := exec.Command(name, argv...)
		cmd.Env = withoutTmuxEnv(os.Environ())
		return cmd
	}
	if kind != multiplexerNone {
		if name, argv := multiplexerCommand(kind, args, cfg.TmuxMode); name != "" {
			return exec.Command(name, argv...)
		}
	}
	name, argv := terminalCommand(terminal, args)
	if name == "" {
		return nil
	}
	return exec.Command(name, argv...)
}

// startSession launches cmd and returns the tmux window it opened in, if the
// command reports one. Commands that print a target exit immediately, so their
// output is awaited; terminal emulators are started without waiting.
func startSession(cmd *exec.Cmd) (string, error) {
	if !slices.Contains(cmd.Args, "-P") {
		return "", cmd.Start()
	}
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func openedMessage(label, target string) string {
	if target != "" {
		return fmt.Sprintf("Opened %s session in tmux window %s", label, target)
	}
	return fmt.Sprintf("Opened %s session", label)
}

// terminalCandidates is the detection order used when no terminal is
// configured.
var terminalCandidates = []string{
	"ghostty",
	"kitty",
	"alacritty",
	"wezterm",
	"foot",
	"gnome-terminal",
	"konsole",
	"xfce4-terminal",
	"x-terminal-emulator",
	"xterm",
}

// resolveTerminal returns the configured terminal, or the first known terminal
// emulator found on PATH.
func resolveTerminal(configured string, lookPath func(string) (string, error)) string {
	if configured != "" {
		return configured
	}
	for _, name := range terminalCandidates {
		if _, err := lookPath(name); err == nil {
			return name
		}
	}
	return ""
}

// terminalExecArgs returns the arguments that make terminal run argv, since
// emulators disagree on how a command is passed.
func terminalExecArgs(terminal string, argv []string) []string {
	switch filepath.Base(terminal) {
	case "kitty", "foot":
		return argv
	case "wezterm":
		return append([]string{"start", "--"}, argv...)
	case "gnome-terminal":
		return append([]string{"--"}, argv...)
	case "xfce4-terminal":
		return append([]string{"-x"}, argv...)
	}
	return append([]string{"-e"}, argv...)
}

func terminalCommand(terminal string, args []string) (string, []string) {
	if terminal == "" {
		return "", nil
	}
	shell := strings.Join(args, " ")
	return terminal, terminalExecArgs(terminal, []string{"sh", "-c", shell})
}

func (m *Model) openNewSession(tool string) tea.Cmd {
	if tool == "" {
		m.statusMessage = "No tool selected"
		return nil
	}
	// Prefer the selected session's project; fall back to the active project
	// filter (set via `p`) when no session gives one, e.g. an empty list.
	var projectDir string
	filtered := m.filteredSessions()
	sel := m.sessionCursor
	if sel >= 0 && sel < len(filtered) {
		projectDir = sessionDir(filtered[sel])
	}
	if projectDir == "" && len(m.filters.projects) > 0 {
		projectDir = m.filters.projects[0]
	}
	if projectDir == "" {
		m.statusMessage = "No project selected"
		return nil
	}
	args := sources.NewSessionArgs(m.meta.Config, tool, projectDir)
	if len(args) == 0 {
		m.statusMessage = fmt.Sprintf("No new session support for %s", tool)
		return nil
	}
	cmd := spawnSession(m.meta.Config, args)
	if cmd == nil {
		m.statusMessage = "Set $TERMINAL or run inside tmux/zellij to open sessions"
		return nil
	}
	m.statusMessage = fmt.Sprintf(
		"Opening new %s session in %s...", tool, uiutil.CleanProjectName(projectDir),
	)
	return launchSession(cmd, "new "+tool)
}

// launchSession starts cmd in the background and reports the outcome in the
// status bar.
func launchSession(cmd *exec.Cmd, label string) tea.Cmd {
	return func() tea.Msg {
		target, err := startSession(cmd)
		if err != nil {
			return statusMsg{message: fmt.Sprintf("Failed to open terminal: %v", err)}
		}
		return statusMsg{message: openedMessage(label, target)}
	}
}

func (m *Model) openSelectedExternally(filtered []session.Session) tea.Cmd {
	sel := m.sessionCursor
	if len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		return nil
	}
	s, redirectedFromSubagent, ok := m.resumeTarget(filtered[sel])
	if !ok {
		return nil
	}
	cmd := sessionCommand(s, m.meta.Config)
	if cmd == nil {
		m.statusMessage = "Set $TERMINAL or run inside tmux/zellij to open sessions"
		return nil
	}
	m.statusMessage = fmt.Sprintf("Opening %s session...", s.Tool)
	if redirectedFromSubagent {
		m.statusMessage = "Subagents can't be resumed directly — " + m.statusMessage
	}
	return launchSession(cmd, s.Tool)
}

// resumeTarget returns the session to resume for s. A subagent's ID is an
// internal transcript for a Task-tool invocation, not a standalone session
// the tool's own CLI can --resume, so its parent is resumed instead. ok is
// false, with the reason in the status bar, when the parent is missing.
func (m *Model) resumeTarget(s session.Session) (session.Session, bool, bool) {
	if s.ParentID == "" {
		return s, false, true
	}
	parent, ok := findSessionByID(m.sessions, s.ParentID)
	if !ok {
		m.statusMessage = "Subagent sessions can't be resumed directly, and the parent session wasn't found"
		return session.Session{}, false, false
	}
	return parent, true, true
}
