package ui

import (
	"fmt"
	"sort"
	"time"

	"charm.land/bubbles/v2/table"

	"github.com/adin/ai-dash/internal/session"
)

func (m *Model) resizeOverviewTable(filtered []session.Session) {
	if m.width == 0 {
		return
	}
	w := max(30, m.width-6) // pane border(2) + padding(4)
	fixed := 8 + 6 + 10     // Sessions + Active + Last
	nameW := max(8, w-fixed-8)
	m.overviewTable.SetColumns([]table.Column{
		{Title: "Project", Width: nameW},
		{Title: "Sessions", Width: 8},
		{Title: "Active", Width: 6},
		{Title: "Last", Width: 10},
	})
	m.overviewTable.SetWidth(w)

	type projectStats struct {
		count, active int
		last          time.Time
	}
	stats := map[string]*projectStats{}
	for _, s := range filtered {
		name := cleanProjectName(s.Project)
		ps, ok := stats[name]
		if !ok {
			ps = &projectStats{}
			stats[name] = ps
		}
		ps.count++
		if s.Status == "active" {
			ps.active++
		}
		if s.StartedAt.After(ps.last) {
			ps.last = s.StartedAt
		}
	}

	keys := make([]string, 0, len(stats))
	for k := range stats {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	rows := make([]table.Row, 0, len(keys))
	for _, k := range keys {
		ps := stats[k]
		lastStr := ps.last.Format("01/02 15:04")
		rows = append(rows, table.Row{
			truncate(k, nameW),
			fmt.Sprintf("%d", ps.count),
			fmt.Sprintf("%d", ps.active),
			lastStr,
		})
	}
	m.overviewTable.SetRows(rows)
	m.overviewTable.SetHeight(paneBodyHeight(topPaneHeight(m.height)))
}

func (m *Model) resizeRightTables(filtered []session.Session) {
	if m.width == 0 || m.height == 0 || len(filtered) == 0 {
		return
	}
	m.resizeSourceTable()
	m.resizeRelatedTable(filtered)
}
