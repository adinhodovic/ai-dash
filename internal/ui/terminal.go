package ui

import (
	"fmt"
	"os"
	"os/exec"
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
	kind := detectMultiplexer(cfg.Multiplexer, nil)
	if kind == multiplexerTmux && cfg.TmuxMode == tmuxModeTerminal && cfg.Terminal != "" {
		name, argv := tmuxTerminalCommand(cfg.Terminal, args)
		cmd := exec.Command(name, argv...)
		cmd.Env = withoutTmuxEnv(os.Environ())
		return cmd
	}
	if kind != multiplexerNone {
		if name, argv := multiplexerCommand(kind, args, cfg.TmuxMode); name != "" {
			return exec.Command(name, argv...)
		}
	}
	name, argv := terminalCommand(cfg.Terminal, args)
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

func terminalCommand(terminal string, args []string) (string, []string) {
	if terminal == "" {
		return "", nil
	}
	shell := strings.Join(args, " ")
	return terminal, []string{"-e", "sh", "-c", shell}
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
	return func() tea.Msg {
		target, err := startSession(cmd)
		if err != nil {
			return statusMsg{message: fmt.Sprintf("Failed to open terminal: %v", err)}
		}
		return statusMsg{message: openedMessage("new "+tool, target)}
	}
}

func (m *Model) openSelectedExternally(filtered []session.Session) tea.Cmd {
	sel := m.sessionCursor
	if len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		return nil
	}
	s := filtered[sel]
	redirectedFromSubagent := false
	if s.ParentID != "" {
		// A subagent's ID is an internal transcript for a Task-tool
		// invocation, not a standalone session the tool's own CLI can
		// --resume — resume the parent instead, since that's almost always
		// what's actually wanted.
		parent, ok := findSessionByID(m.sessions, s.ParentID)
		if !ok {
			m.statusMessage = "Subagent sessions can't be resumed directly, and the parent session wasn't found"
			return nil
		}
		s = parent
		redirectedFromSubagent = true
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
	return func() tea.Msg {
		target, err := startSession(cmd)
		if err != nil {
			return statusMsg{message: fmt.Sprintf("Failed to open terminal: %v", err)}
		}
		return statusMsg{message: openedMessage(s.Tool, target)}
	}
}
