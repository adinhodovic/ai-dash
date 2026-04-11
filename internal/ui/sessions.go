package ui

import (
	"charm.land/bubbles/v2/table"

	"github.com/adinhodovic/ai-dash/internal/session"
	uilayout "github.com/adinhodovic/ai-dash/internal/ui/layout"
	"github.com/adinhodovic/ai-dash/internal/ui/theme"
	uiutil "github.com/adinhodovic/ai-dash/internal/ui/util"
)

func (m *Model) resizeTable(filtered []session.Session) {
	width := max(44, m.width)
	if !m.detailCollapsed {
		width = max(40, m.width*70/100)
	}
	// Subtract pane border (2) for inner width; header join (1) for height.
	tableW := max(40, width-2)
	summaryW := max(16, tableW-63)
	height := max(2, uilayout.PaneBodyHeight(uilayout.BottomPaneHeight(m.height))-1)
	m.sessionTable.SetColumns([]table.Column{
		{Title: m.sortHeader("Last Active", session.SortUpdated), Width: 14},
		{Title: m.sortHeader("Tool", session.SortTool), Width: 8},
		{Title: m.sortHeader("Status", session.SortStatus), Width: 11},
		{Title: m.sortHeader("Project", session.SortProject), Width: 20},
		{Title: m.sortHeader("Summary", session.SortSummary), Width: summaryW},
	})
	m.sessionTable.SetWidth(tableW)
	m.sessionTable.SetHeight(height)
	m.syncTable(filtered)
}

func (m *Model) syncTable(filtered []session.Session) {
	rows := make([]table.Row, 0, len(filtered))
	summaryWidth := max(16, m.sessionTable.Width()-63)
	for _, s := range filtered {
		status := uiutil.SessionStatusLabel(s)
		rows = append(rows, table.Row{
			uiutil.TimeAgo(uiutil.LastActive(s)),
			uiutil.Capitalize(s.Tool),
			theme.StatusStyle(status).Render(uiutil.TruncateForCell(status, 11)),
			uiutil.TruncateForCell(uiutil.CleanProjectName(s.Project), 20),
			uiutil.TruncateForCell(uiutil.CleanSummary(s.Summary), summaryWidth),
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
		rows = append(
			rows,
			table.Row{source.Tool, source.Kind, status, uiutil.ShortenPath(source.Path)},
		)
	}
	m.sourceTable.SetRows(rows)
	m.sourceTable.SetCursor(0)
}
