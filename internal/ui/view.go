package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/adin/ai-dash/internal/session"
)

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return altView("")
	}
	if m.err != nil && len(m.sessions) == 0 {
		return altView(m.styles.error.Render(m.err.Error()))
	}

	filtered := m.filteredSessions()

	// Layout budget: topBar(1) + content(variable) + footer(2: rule+keys) = m.height
	contentH := m.height - 3
	if contentH < 4 {
		contentH = 4
	}

	top := ansi.Truncate(m.renderTopBar(filtered), m.width, "")
	footer := m.renderFooter(filtered)

	if len(m.sessions) == 0 {
		body := m.renderPane(m.styles.panel, "Sessions", "No sessions loaded.", m.width, contentH)
		return altView(m.composePageStr(top, body, footer))
	}
	if len(filtered) == 0 {
		body := m.renderPane(m.styles.panel, "Sessions", m.styles.muted.Render("No matches. Press c to clear or / to search."), m.width, contentH)
		return altView(m.composePageStr(top, body, footer))
	}

	if m.detailCollapsed {
		previewH := 2
		tableH := contentH - previewH
		tablePane := m.renderPane(m.panelStyle(m.focus == focusList), "Sessions", m.renderSessionPane(filtered), m.width, tableH)
		preview := ansi.Truncate(m.renderCollapsedPreview(filtered), m.width, "")
		// Stitch without extra newline: replace last empty line of table pane with preview
		tLines := strings.Split(tablePane, "\n")
		if len(tLines) > tableH {
			tLines = tLines[:tableH]
		}
		tLines = append(tLines, preview)
		if len(tLines) > contentH {
			tLines = tLines[:contentH]
		}
		content := strings.Join(tLines, "\n")
		return altView(m.composePageStr(top, content, footer))
	}

	// Layout: top row = projects, bottom row = sessions + details
	topH := topPaneHeight(m.height)
	botH := bottomPaneHeight(m.height)
	leftW := max(40, m.width*60/100)
	rightW := m.width - leftW

	projects := m.renderPane(m.panelStyle(m.focus == focusFilters), "Projects", m.overviewTable.View(), m.width, topH)
	sessions := m.renderPane(m.panelStyle(m.focus == focusList), "Sessions", m.renderSessionPane(filtered), leftW, botH)
	detail := m.renderPane(m.panelStyle(m.focus == focusDetail), "Details", m.renderDetailPane(), rightW, botH)

	botRow := lipgloss.JoinHorizontal(lipgloss.Top, sessions, detail)
	content := lipgloss.JoinVertical(lipgloss.Left, projects, botRow)

	page := m.composePageStr(top, content, footer)
	if m.picker.active {
		page = m.overlayPicker(page)
	}
	if m.showSources {
		page = m.overlaySources(page)
	}
	if m.showHelp {
		page = m.overlayHelp(page)
	}
	return altView(page)
}

func (m Model) composePageStr(top, content, footer string) string {
	return lipgloss.JoinVertical(lipgloss.Left, top, content, footer)
}

func altView(s string) tea.View {
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func (m Model) renderFooter(_ []session.Session) string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.help
	h.SetWidth(w)
	return ansi.Truncate(strings.ReplaceAll(h.ShortHelpView(m.keys.ShortHelp()), "\n", " "), w, "")
}

func (m Model) overlayPicker(_ string) string {
	width := max(30, m.width*40/100)
	height := min(m.picker.list.Height()+4, m.height-4)
	overlay := m.styles.overlay.Width(width).Height(height).Render(m.picker.list.View())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}

func (m Model) overlaySources(_ string) string {
	width := max(40, m.width*70/100)
	height := min(12, m.height-6)
	title := m.styles.header.PaddingLeft(1).PaddingRight(1).MarginBottom(1).Render("Sources")
	hint := m.styles.muted.MarginTop(1).Render("Press S or Esc to close")
	body := lipgloss.JoinVertical(lipgloss.Left, title, m.sourceTable.View(), hint)
	overlay := m.styles.overlay.Width(width).Height(height).Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}

func (m Model) overlayHelp(_ string) string {
	width := max(30, min(60, m.width-4))
	height := min(20, m.height-6)
	h := m.help
	h.SetWidth(width - 6)
	title := m.styles.header.PaddingLeft(1).PaddingRight(1).MarginBottom(1).Render("Keyboard Shortcuts")
	helpText := lipgloss.NewStyle().MarginBottom(1).Render(h.FullHelpView(m.keys.FullHelp()))
	body := lipgloss.JoinVertical(lipgloss.Left, title, helpText, m.styles.muted.Render("Press ? to close"))
	overlay := m.styles.overlay.Width(width).Height(height).Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}

func (m Model) renderSessionPane(filtered []session.Session) string {
	return lipgloss.JoinVertical(lipgloss.Left, m.renderActiveFilters(), m.sessionTable.View())
}

func (m Model) renderDetailPane() string {
	detailW := m.width - m.width*60/100
	innerW := max(10, detailW-4)
	divider := m.styles.muted.Render(strings.Repeat("─", max(1, innerW)))
	summaryH, _, _ := detailPaneSectionHeights(m.height)

	summary := m.selectedSummary()
	if len(summary) > 500 {
		summary = summary[:497] + "..."
	}
	summaryLabel := m.styles.muted.Render("Summary")
	summaryText := lipgloss.NewStyle().Width(innerW).Height(summaryH - 1).MaxHeight(summaryH - 1).Render(summary)

	relatedLabel := m.styles.highlight.Render("Related Sessions")

	return lipgloss.JoinVertical(lipgloss.Left,
		summaryLabel,
		summaryText,
		divider,
		m.detailTable.View(),
		divider,
		relatedLabel,
		m.relatedTable.View(),
	)
}

func (m Model) selectedSummary() string {
	filtered := m.filteredSessions()
	if len(filtered) == 0 || m.selected < 0 || m.selected >= len(filtered) {
		return ""
	}
	return filtered[m.selected].Summary
}

func (m Model) renderCollapsedPreview(filtered []session.Session) string {
	if len(filtered) == 0 || m.selected >= len(filtered) {
		return m.styles.muted.Render("  No session selected")
	}
	s := filtered[m.selected]
	parts := []string{
		m.styles.highlight.Render(s.Project),
		s.Tool,
		s.Status,
		valueOrUnknown(s.Model),
		durationLabel(s),
	}
	if s.CostUSD > 0 {
		parts = append(parts, formatCost(s.CostUSD))
	}
	if s.TokensIn+s.TokensOut > 0 {
		parts = append(parts, formatTokens(s.TokensIn, s.TokensOut))
	}
	line := strings.Join(parts, "  ")
	summary := s.Summary
	if len(summary) > 60 {
		summary = summary[:57] + "..."
	}
	content := lipgloss.JoinVertical(lipgloss.Left, line, m.styles.muted.Render(summary))
	return m.styles.subpanel.Width(max(40, m.width-2)).Render(content)
}

func (m Model) detailItems(s session.Session) []detailItem {
	items := []detailItem{
		{"Tool", s.Tool},
		{"Project", cleanProjectName(s.Project)},
		{"Status", s.Status},
		{"Model", valueOrUnknown(s.Model)},
		{"Started", s.StartedAt.Format("2006-01-02 15:04:05")},
		{"Ended", session.EndedLabel(s.EndedAt, s.Status)},
		{"Duration", durationLabel(s)},
	}
	if s.Repo != "" {
		items = append(items, detailItem{"Repo", shortenPath(s.Repo)})
	}
	if s.Branch != "" {
		items = append(items, detailItem{"Branch", s.Branch})
	}
	if s.Slug != "" {
		items = append(items, detailItem{"Slug", s.Slug})
	}
	if s.ParentID != "" {
		items = append(items, detailItem{"Parent", s.ParentID})
	}
	if s.TokensIn+s.TokensOut > 0 {
		items = append(items, detailItem{"Tokens", formatTokens(s.TokensIn, s.TokensOut)})
	}
	if s.CostUSD > 0 {
		items = append(items, detailItem{"Cost", formatCost(s.CostUSD)})
	}
	if len(s.Tags) > 0 {
		items = append(items, detailItem{"Tags", strings.Join(s.Tags, ", ")})
	}
	items = append(items, detailItem{"ID", s.ID})
	return items
}

func (m Model) renderPane(style lipgloss.Style, title, body string, width, height int) string {
	innerW := max(1, width-2)
	innerH := max(1, height-4) // border top(1) + border bottom(1) + title(1) + gap(1)

	// Title + body, truncated to exact dimensions
	titleLine := m.styles.header.PaddingRight(1).PaddingLeft(1).MarginBottom(1).Render(title)
	full := lipgloss.JoinVertical(lipgloss.Left, titleLine, body)
	lines := strings.Split(full, "\n")
	if len(lines) > innerH {
		lines = lines[:innerH]
	}
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, innerW, "")
	}

	return style.
		Width(innerW).
		Height(innerH).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderActiveFilters() string {
	var chips []string
	if m.filters.tool != "" {
		chips = append(chips, m.styles.badge.Render("tool:"+m.filters.tool))
	}
	if m.filters.status != "" {
		chips = append(chips, m.styles.badge.Render("status:"+m.filters.status))
	}
	if m.filters.project != "" {
		chips = append(chips, m.styles.badge.Render("project:"+cleanProjectName(m.filters.project)))
	}
	if len(chips) == 0 {
		return m.styles.muted.Render("no filters active")
	}
	return strings.Join(chips, " ") + "  " + m.styles.muted.Render("c to clear")
}

func (m Model) renderTopBar(filtered []session.Session) string {
	search := strings.TrimSpace(m.searchQuery())
	if search == "" {
		search = "all"
	}
	sep := m.styles.muted.Render(" │ ")
	left := strings.Join([]string{
		m.styles.muted.Render("search ") + m.styles.selected.Render(search),
		m.styles.muted.Render("sort ") + m.styles.selected.Render(m.sortLabel()),
		m.styles.selected.Render(fmt.Sprintf("%d/%d", len(filtered), len(m.sessions))),
	}, sep)

	title := m.styles.header.Render(" AI Dashboard ")
	gap := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(title))
	return left + strings.Repeat(" ", gap) + title
}

func (m Model) panelStyle(active bool) lipgloss.Style {
	if active {
		return m.styles.active
	}
	return m.styles.panel
}

func (m Model) sortLabel() string {
	dir := "asc"
	if m.sortDescending {
		dir = "desc"
	}
	return fmt.Sprintf("%s %s", m.sortField, dir)
}

func (m Model) sortHeader(label string, field session.SortField) string {
	if m.sortField != field {
		return label
	}
	if m.sortDescending {
		return label + " v"
	}
	return label + " ^"
}
