package ui

import (
	"charm.land/bubbles/v2/table"

	"github.com/adin/ai-dash/internal/session"
	uilayout "github.com/adin/ai-dash/internal/ui/layout"
)

func detailPaneSectionHeights(termHeight int) (summary, detail, related int) {
	bodyH := uilayout.PaneBodyHeight(uilayout.BottomPaneHeight(termHeight))
	// Layout: summaryLabel(1) + summaryText(1) + divider(1) + detailTable + divider(1) + relatedLabel(1) + relatedTable
	summary = 2
	related = 6
	fixed := summary + 2 + 2 + related // summary section + 2 divider/label lines + 2 table join newlines
	detail = max(3, bodyH-fixed)
	return summary, detail, related
}

func (m *Model) resizeDetailTable(filtered []session.Session) {
	sel := m.sessionTable.Cursor()
	if m.width == 0 || len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		m.detailTable.SetRows(nil)
		m.detailTable.UpdateViewport()
		return
	}
	detailW := m.width - m.width*70/100
	innerW := max(10, detailW-2) // subtract pane border
	_, detailH, _ := detailPaneSectionHeights(m.height)
	keyW := 14
	valW := max(10, innerW-keyW-4) // subtract cell padding (1 each side × 2 cols)
	m.detailTable.SetColumns([]table.Column{
		{Title: "", Width: keyW},
		{Title: "", Width: valW},
	})
	m.detailTable.SetWidth(innerW)
	items := m.detailItems(filtered[sel])
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, table.Row{item.title, truncate(item.desc, valW)})
	}
	m.detailTable.SetRows(rows)
	m.detailTable.SetHeight(detailH)
	m.detailTable.UpdateViewport()
}

func (m *Model) resizeRelatedTable(filtered []session.Session) {
	detailW := m.width - m.width*70/100
	width := max(30, detailW-2) // subtract pane border
	_, _, relatedH := detailPaneSectionHeights(m.height)
	m.relatedTable.SetColumns([]table.Column{
		{Title: "Tool", Width: 7},
		{Title: "Project", Width: 12},
		{Title: "Relation", Width: 8},
		{Title: "Summary", Width: max(8, width-35)},
	})
	m.relatedTable.SetWidth(width)
	m.relatedTable.SetHeight(relatedH)
	m.syncRelatedTable(filtered)
}

func (m *Model) syncRelatedTable(filtered []session.Session) {
	rows := make([]table.Row, 0)
	sel := m.sessionTable.Cursor()
	if len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		m.relatedTable.SetRows(rows)
		return
	}
	selected := filtered[sel]
	for _, candidate := range m.sessions {
		if candidate.ID == selected.ID {
			continue
		}
		relation := relationLabel(selected, candidate)
		if relation == "" {
			continue
		}
		rows = append(
			rows,
			table.Row{
				candidate.Tool,
				cleanProjectName(candidate.Project),
				relation,
				candidate.Summary,
			},
		)
	}
	m.relatedTable.SetRows(rows)
	m.relatedTable.SetCursor(0)
	m.relatedTable.UpdateViewport()
}
