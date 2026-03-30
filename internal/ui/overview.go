package ui

import (
	"fmt"
	"sort"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/ui/icon"
)

func (m *Model) resizeOverviewTable(filtered []session.Session) {
	if m.width == 0 {
		return
	}
	topH := topPaneHeight(m.height)
	bodyH := paneBodyHeight(topH)

	projW := max(40, m.width*70/100)
	tableW := max(30, projW-2) // subtract pane border
	fixed := 14 + 16           // Last Active + Sessions column widths
	cellPad := 3 * 2           // 3 columns × 2 padding
	nameW := max(8, tableW-fixed-cellPad)
	m.overviewTable.SetColumns([]table.Column{
		{Title: m.projSortHeader("Last Active", "last", 14), Width: 14},
		{Title: m.projSortHeader("Project", "project", nameW), Width: nameW},
		{Title: m.projSortHeader(icon.Session+" Sessions", "sessions", 16), Width: 16},
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
			timeAgo(ps.last),
			truncate(cleanProjectName(k), nameW),
			countStr,
		})
		m.projectPaths = append(m.projectPaths, k)
	}
	m.overviewTable.SetRows(rows)
	m.overviewTable.UpdateViewport()
}

func (m Model) renderOverviewStats(filtered []session.Session, _, _ int) string {
	all := m.sessions
	total := len(all)
	visible := len(filtered)
	var active, subagents int
	tools := map[string]int{}
	var totalCost float64
	var tokensIn, tokensOut int

	for _, s := range all {
		if s.Status == "active" {
			active++
		}
		if s.ParentID != "" {
			subagents++
		}
		tools[s.Tool]++
		totalCost += s.CostUSD
		tokensIn += s.TokensIn
		tokensOut += s.TokensOut
	}

	dim := m.styles.muted
	val := m.styles.highlight
	sel := m.styles.selected

	// Header stats row
	statsLine := sel.Render(fmt.Sprintf("%d", visible)) +
		dim.Render("/") +
		dim.Render(fmt.Sprintf("%d", total)) +
		dim.PaddingLeft(1).Render("sessions")
	if active > 0 {
		statsLine += val.MarginLeft(2).Render(fmt.Sprintf("%d", active)) + dim.PaddingLeft(1).Render("active")
	}

	toolKeys := make([]string, 0, len(tools))
	for k := range tools {
		toolKeys = append(toolKeys, k)
	}
	sort.Strings(toolKeys)

	var toolLines []string
	for _, k := range toolKeys {
		label := dim.Render(fmt.Sprintf("%-10s", capitalize(k)))
		toolLines = append(toolLines, label+val.Render(fmt.Sprintf("%d", tools[k])))
	}

	lines := []string{statsLine}

	if subagents > 0 {
		lines = append(lines, dim.Render(fmt.Sprintf("%d subagent sessions (press a to show)", subagents)))
	}

	lines = append(lines, "")
	lines = append(lines, toolLines...)

	if tokensIn+tokensOut > 0 {
		lines = append(lines, "")
		lines = append(lines, dim.Render(icon.Token+" Tokens ")+val.Render(formatTokens(tokensIn, tokensOut)))
	}
	if totalCost > 0 {
		lines = append(lines, dim.Render(icon.Cost+" Cost   ")+val.Render(formatCost(totalCost)))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.NewStyle().PaddingLeft(1).Render(content)
}

type projStats struct {
	count, active int
	last          time.Time
}

func projectStats(sessions []session.Session) map[string]*projStats {
	stats := map[string]*projStats{}
	for _, s := range sessions {
		name := cleanProjectName(s.Project)
		ps, ok := stats[name]
		if !ok {
			ps = &projStats{}
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
	return stats
}

func (m *Model) sortedProjectKeys(stats map[string]*projStats) []string {
	keys := make([]string, 0, len(stats))
	for k := range stats {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		var less bool
		switch m.projSortField {
		case "project":
			less = cleanProjectName(keys[i]) < cleanProjectName(keys[j])
		case "sessions":
			less = stats[keys[i]].count < stats[keys[j]].count
		case "active":
			less = stats[keys[i]].active < stats[keys[j]].active
		default: // "last"
			less = stats[keys[i]].last.Before(stats[keys[j]].last)
		}
		if m.projSortDesc {
			return !less
		}
		return less
	})
	return keys
}

var projectSortFields = []string{"last", "project", "sessions"}

func (m *Model) cycleSortForward() {
	if m.focus == focusFilters {
		m.projSortField = nextInSlice(m.projSortField, projectSortFields)
		m.statusMessage = fmt.Sprintf("Projects sort: %s", m.projSortField)
	} else {
		m.sortField = nextSortField(m.sortField)
		m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
	}
}

func (m *Model) cycleSortBackward() {
	if m.focus == focusFilters {
		m.projSortField = prevInSlice(m.projSortField, projectSortFields)
		m.statusMessage = fmt.Sprintf("Projects sort: %s", m.projSortField)
	} else {
		m.sortField = prevSortField(m.sortField)
		m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
	}
}

func (m *Model) toggleSortDirection() {
	if m.focus == focusFilters {
		m.projSortDesc = !m.projSortDesc
		m.statusMessage = fmt.Sprintf("Projects sort: %s", m.projSortField)
	} else {
		m.sortDescending = !m.sortDescending
		m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
	}
}

func nextInSlice(current string, options []string) string {
	for i, v := range options {
		if v == current {
			return options[(i+1)%len(options)]
		}
	}
	return options[0]
}

func prevInSlice(current string, options []string) string {
	for i, v := range options {
		if v == current {
			return options[(i-1+len(options))%len(options)]
		}
	}
	return options[0]
}

func (m *Model) resizeRightTables(filtered []session.Session) {
	if m.width == 0 || m.height == 0 || len(filtered) == 0 {
		return
	}
	m.resizeSourceTable()
	m.resizeRelatedTable(filtered)
}
