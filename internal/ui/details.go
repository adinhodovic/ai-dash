package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"

	"github.com/adin/ai-dash/internal/session"
)

func detailPaneSectionHeights(termHeight int) (summary, detail, related int) {
	bodyH := paneBodyHeight(bottomPaneHeight(termHeight))
	related = 4
	summary = 3
	detail = max(3, bodyH-summary-related-4)
	return summary, detail, related
}

func (m *Model) resizeDetailTable(filtered []session.Session) {
	if m.width == 0 || len(filtered) == 0 || m.selected < 0 || m.selected >= len(filtered) {
		m.detailTable.SetRows(nil)
		return
	}
	detailW := m.width - m.width*60/100 // 40% of screen
	_, detailH, _ := detailPaneSectionHeights(m.height)
	keyW := 10
	valW := max(10, detailW-keyW-4)
	m.detailTable.SetColumns([]table.Column{
		{Title: "Field", Width: keyW},
		{Title: "Value", Width: valW},
	})
	m.detailTable.SetWidth(detailW)
	items := m.detailItems(filtered[m.selected])
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, table.Row{item.title, truncate(item.desc, valW)})
	}
	m.detailTable.SetRows(rows)
	m.detailTable.SetHeight(detailH)
}

func (m *Model) resizeRelatedTable(filtered []session.Session) {
	detailW := m.width - m.width*60/100
	width := max(30, detailW-4)
	_, _, relatedH := detailPaneSectionHeights(m.height)
	m.relatedTable.SetColumns([]table.Column{
		{Title: "Tool", Width: 7},
		{Title: "Project", Width: 12},
		{Title: "Relation", Width: 8},
		{Title: "Summary", Width: max(8, width-31)},
	})
	m.relatedTable.SetWidth(width)
	m.relatedTable.SetHeight(relatedH)
	m.syncRelatedTable(filtered)
}

func (m *Model) syncRelatedTable(filtered []session.Session) {
	rows := make([]table.Row, 0)
	if len(filtered) == 0 || m.selected >= len(filtered) {
		m.relatedTable.SetRows(rows)
		return
	}
	selected := filtered[m.selected]
	for _, candidate := range m.sessions {
		if candidate.ID == selected.ID {
			continue
		}
		relation := relationLabel(selected, candidate)
		if relation == "" {
			continue
		}
		rows = append(rows, table.Row{candidate.Tool, cleanProjectName(candidate.Project), relation, candidate.Summary})
	}
	m.relatedTable.SetRows(rows)
	m.relatedTable.SetCursor(0)
}

func (m *Model) jumpToRelated(filtered []session.Session) bool {
	if len(filtered) == 0 || m.selected < 0 || m.selected >= len(filtered) {
		return false
	}
	rows := m.relatedTable.Rows()
	cursor := m.relatedTable.Cursor()
	if len(rows) == 0 || cursor >= len(rows) || len(rows[cursor]) < 4 {
		return false
	}
	selected := filtered[m.selected]
	row := rows[cursor]
	for i, candidate := range m.filteredSessions() {
		if candidate.ID == selected.ID {
			continue
		}
		if candidate.Tool == row[0] && cleanProjectName(candidate.Project) == row[1] && candidate.Summary == row[3] {
			m.selected = i
			m.sessionTable.SetCursor(i)
			m.statusMessage = fmt.Sprintf("Jumped to %s session", strings.ToLower(candidate.Tool))
			return true
		}
	}
	return false
}
