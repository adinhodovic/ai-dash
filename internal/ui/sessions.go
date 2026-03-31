package ui

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources"
)

func (m *Model) resizeTable(filtered []session.Session) {
	width := max(44, m.width)
	if !m.detailCollapsed {
		width = max(40, m.width*70/100)
	}
	// Subtract pane border (2) for inner width; header join (1) for height.
	tableW := max(40, width-2)
	summaryW := max(16, tableW-60)
	height := max(2, paneBodyHeight(bottomPaneHeight(m.height))-1)
	m.sessionTable.SetColumns([]table.Column{
		{Title: m.sortHeader("Last Active", session.SortUpdated, 14), Width: 14},
		{Title: m.sortHeader("Tool", session.SortTool, 9), Width: 9},
		{Title: m.sortHeader("Project", session.SortProject, 28), Width: 28},
		{Title: m.sortHeader("Summary", session.SortSummary, summaryW), Width: summaryW},
	})
	m.sessionTable.SetWidth(tableW)
	m.sessionTable.SetHeight(height)
	m.syncTable(filtered)
}

func (m *Model) syncTable(filtered []session.Session) {
	rows := make([]table.Row, 0, len(filtered))
	summaryWidth := max(16, m.sessionTable.Width()-60)
	for _, s := range filtered {
		rows = append(rows, table.Row{
			timeAgo(lastActive(s)),
			capitalize(s.Tool),
			truncateForCell(cleanProjectName(s.Project), 28),
			truncateForCell(cleanSummary(s.Summary), summaryWidth),
		})
	}
	m.sessionTable.SetRows(rows)
}

func (m *Model) resizeSourceTable() {
	width := max(40, m.width*70/100-6)
	m.sourceTable.SetColumns([]table.Column{
		{Title: "Tool", Width: 9},
		{Title: "Format", Width: 8},
		{Title: "Status", Width: 8},
		{Title: "Path", Width: max(16, width-29)},
	})
	m.sourceTable.SetWidth(width)
	m.sourceTable.SetHeight(max(3, min(5, len(m.meta.Discovery.Sources)+1)))
	m.syncSourceTable()
}

func (m *Model) syncSourceTable() {
	rows := make([]table.Row, 0, len(m.meta.Discovery.Sources))
	for _, source := range m.meta.Discovery.Sources {
		status := "missing"
		if source.Exists {
			status = "present"
		}
		rows = append(rows, table.Row{source.Tool, source.Kind, status, shortenPath(source.Path)})
	}
	m.sourceTable.SetRows(rows)
	m.sourceTable.SetCursor(0)
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
	cmd := spawnTerminal(args)
	if cmd == nil {
		m.statusMessage = "Set $TERMINAL to open sessions (e.g. export TERMINAL=ghostty)"
		return nil
	}
	m.statusMessage = fmt.Sprintf(
		"Opening new %s session in %s...", tool, cleanProjectName(projectDir),
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
		m.statusMessage = "Set $TERMINAL to open sessions (e.g. export TERMINAL=ghostty)"
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
