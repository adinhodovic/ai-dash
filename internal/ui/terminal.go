package ui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/adin/ai-dash/internal/config"
	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources"
	uiutil "github.com/adin/ai-dash/internal/ui/util"
)

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
	return spawnTerminal(cfg.Terminal, args)
}

func spawnTerminal(terminal string, args []string) *exec.Cmd {
	name, argv := terminalCommand(terminal, args)
	if name == "" {
		return nil
	}
	return exec.Command(name, argv...)
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
	// Get project dir from focused table
	var projectDir string
	if m.focus == focusFilters {
		cursor := m.overviewTable.Cursor()
		if cursor >= 0 && cursor < len(m.projectPaths) {
			projectDir = m.projectPaths[cursor]
		}
	} else {
		filtered := m.filteredSessions()
		sel := m.sessionTable.Cursor()
		if sel >= 0 && sel < len(filtered) {
			projectDir = sessionDir(filtered[sel])
		}
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
	cmd := spawnTerminal(m.meta.Config.Terminal, args)
	if cmd == nil {
		m.statusMessage = "Set $TERMINAL to open sessions (e.g. ghostty or kitty)"
		return nil
	}
	m.statusMessage = fmt.Sprintf(
		"Opening new %s session in %s...", tool, uiutil.CleanProjectName(projectDir),
	)
	return func() tea.Msg {
		if err := cmd.Start(); err != nil {
			return statusMsg{message: fmt.Sprintf("Failed to open terminal: %v", err)}
		}
		return statusMsg{message: fmt.Sprintf("Opened new %s session", tool)}
	}
}

func (m *Model) openSelectedExternally(filtered []session.Session) tea.Cmd {
	sel := m.sessionTable.Cursor()
	if len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		return nil
	}
	s := filtered[sel]
	cmd := sessionCommand(s, m.meta.Config)
	if cmd == nil {
		m.statusMessage = "Set $TERMINAL to open sessions (e.g. ghostty or kitty)"
		return nil
	}
	m.statusMessage = fmt.Sprintf("Opening %s session in new terminal...", s.Tool)
	return func() tea.Msg {
		if err := cmd.Start(); err != nil {
			return statusMsg{message: fmt.Sprintf("Failed to open terminal: %v", err)}
		}
		return statusMsg{message: fmt.Sprintf("Opened %s session in new terminal", s.Tool)}
	}
}
