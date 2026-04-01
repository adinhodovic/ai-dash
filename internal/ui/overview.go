package ui

import (
	"fmt"

	"charm.land/bubbles/v2/table"

	"github.com/adin/ai-dash/internal/session"
	uilayout "github.com/adin/ai-dash/internal/ui/layout"
	"github.com/adin/ai-dash/internal/ui/theme"
	uiutil "github.com/adin/ai-dash/internal/ui/util"
)

func (m *Model) resizeOverviewTable(filtered []session.Session) {
	if m.width == 0 {
		return
	}
	topH := uilayout.TopPaneHeight(m.height)
	bodyH := uilayout.PaneBodyHeight(topH)

	projW := max(40, m.width*70/100)
	tableW := max(30, projW-2) // subtract pane border
	fixed := 14 + 16           // Last Active + Sessions column widths
	cellPad := 3 * 2           // 3 columns × 2 padding
	nameW := max(8, tableW-fixed-cellPad)
	m.overviewTable.SetColumns([]table.Column{
		{Title: m.projSortHeader("Last Active", "last", 14), Width: 14},
		{Title: m.projSortHeader("Project", "project", nameW), Width: nameW},
		{Title: m.projSortHeader(theme.Session+" Sessions", "sessions", 16), Width: 16},
	})
	m.overviewTable.SetWidth(tableW)
	m.overviewTable.SetHeight(max(2, bodyH-1))

	stats := projectStats(filtered)
	keys := m.sortedProjectKeys(stats)

	rows := make([]table.Row, 0, len(keys))
	m.projectPaths = make([]string, 0, len(keys))
	for _, k := range keys {
		ps := stats[k]
		countStr := fmt.Sprintf("%d", ps.count)
		if ps.active == 1 {
			countStr += " (1 active)"
		} else if ps.active > 1 {
			countStr += fmt.Sprintf(" (%d active)", ps.active)
		}
		rows = append(rows, table.Row{
			uiutil.TimeAgo(ps.last),
			uiutil.Truncate(uiutil.CleanProjectName(k), nameW),
			countStr,
		})
		m.projectPaths = append(m.projectPaths, k)
	}
	m.overviewTable.SetRows(rows)
	m.overviewTable.UpdateViewport()
}

func (m *Model) resizeRightTables(filtered []session.Session) {
	if m.width == 0 || m.height == 0 || len(filtered) == 0 {
		return
	}
	m.resizeSourceTable()
	m.resizeRelatedTable(filtered)
}
