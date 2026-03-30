package ui

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/adin/ai-dash/internal/session"
)

func (m *Model) resizeTable(filtered []session.Session) {
	width := max(44, m.width)
	if !m.detailCollapsed {
		width = max(40, m.width*60/100)
	}
	height := max(2, paneBodyHeight(bottomPaneHeight(m.height))-1)
	m.sessionTable.SetColumns([]table.Column{
		{Title: "Last Active", Width: 12},
		{Title: m.sortHeader("Tool", session.SortTool), Width: 9},
		{Title: m.sortHeader("Project", session.SortProject), Width: 14},
		{Title: m.sortHeader("Status", session.SortStatus), Width: 10},
		{Title: "Summary", Width: max(16, width-50)},
	})
	m.sessionTable.SetWidth(width)
	m.sessionTable.SetHeight(height)
	m.syncTable(filtered)
}

func (m *Model) syncTable(filtered []session.Session) {
	rows := make([]table.Row, 0, len(filtered))
	summaryWidth := max(16, m.sessionTable.Width()-50)
	for _, s := range filtered {
		rows = append(rows, table.Row{
			timeAgo(lastActive(s)),
			capitalize(s.Tool),
			truncateForCell(cleanProjectName(s.Project), 14),
			s.Status,
			truncateForCell(cleanSummary(s.Summary), summaryWidth),
		})
	}
	m.sessionTable.SetRows(rows)
	m.clampSelection(len(filtered))
	m.sessionTable.SetCursor(m.selected)
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

func (m *Model) openSelectedExternally(filtered []session.Session) tea.Cmd {
	if len(filtered) == 0 || m.selected < 0 || m.selected >= len(filtered) {
		return nil
	}
	s := filtered[m.selected]
	cmd := sessionCommand(s)
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
