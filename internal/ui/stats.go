package ui

import (
	"fmt"
	"sort"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/ui/theme"
	uiutil "github.com/adin/ai-dash/internal/ui/util"
)

func (m Model) renderOverviewStats(filtered []session.Session) string {
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

	dim := m.styles.Muted
	val := m.styles.Highlight
	sel := m.styles.Selected

	// Header stats row
	statsLine := sel.Render(fmt.Sprintf("%d", visible)) +
		dim.Render("/") +
		dim.Render(fmt.Sprintf("%d", total)) +
		dim.PaddingLeft(1).Render("sessions")
	if active > 0 {
		statsLine += val.MarginLeft(2).
			Render(fmt.Sprintf("%d", active)) +
			dim.PaddingLeft(1).
				Render("active")
	}

	toolKeys := make([]string, 0, len(tools))
	for k := range tools {
		toolKeys = append(toolKeys, k)
	}
	sort.Strings(toolKeys)

	var toolLines []string
	for _, k := range toolKeys {
		label := dim.Render(fmt.Sprintf("%-10s", uiutil.Capitalize(k)))
		toolLines = append(toolLines, label+val.Render(fmt.Sprintf("%d", tools[k])))
	}

	lines := []string{statsLine}

	if subagents > 0 {
		lines = append(
			lines,
			dim.Render(fmt.Sprintf("%d subagent sessions (press a to show)", subagents)),
		)
	}

	lines = append(lines, "")
	lines = append(lines, toolLines...)

	if tokensIn+tokensOut > 0 {
		lines = append(lines, "")
		lines = append(
			lines,
			dim.MarginRight(1).
				Render(theme.Token+" Tokens")+
				val.Render(
					uiutil.FormatTokens(tokensIn, tokensOut),
				),
		)
	}
	if totalCost > 0 {
		lines = append(
			lines,
			dim.MarginRight(1).Render(theme.Cost+" Cost")+val.Render(uiutil.FormatCost(totalCost)),
		)
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
		project := s.Project
		ps, ok := stats[project]
		if !ok {
			ps = &projStats{}
			stats[project] = ps
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
			less = uiutil.CleanProjectName(keys[i]) < uiutil.CleanProjectName(keys[j])
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
