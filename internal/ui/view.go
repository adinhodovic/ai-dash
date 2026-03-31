package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/ui/icon"
)

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return altView("")
	}
	if m.err != nil && len(m.sessions) == 0 {
		return altView(m.styles.error.Render(m.err.Error()))
	}

	filtered := m.filteredSessions()

	// Layout budget: topBar(1) + content(variable) + footer(1) = m.height
	contentH := contentHeight(m.height)
	top := " " + ansi.Truncate(m.renderTopBar(filtered), m.width-2, "")
	footer := " " + m.renderFooter(filtered)

	if len(m.sessions) == 0 {
		body := renderPane(
			m.styles.panel,
			m.styles.header,
			"Sessions",
			"No sessions loaded.",
			m.width,
			contentH,
		)
		return altView(m.composePageStr(top, body, footer))
	}
	if len(filtered) == 0 {
		body := renderPane(
			m.styles.panel,
			m.styles.header,
			"Sessions",
			m.styles.muted.Render("No matches. Press c to clear or / to search."),
			m.width,
			contentH,
		)
		return altView(m.composePageStr(top, body, footer))
	}

	if m.detailCollapsed {
		previewH := 2
		tableH := contentH - previewH
		tablePane := renderPane(
			panelStyle(m.styles, m.focus == focusList),
			m.styles.header,
			"Sessions",
			m.renderSessionPane(filtered),
			m.width,
			tableH,
		)
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
		content := lipgloss.JoinVertical(lipgloss.Left, tLines...)
		return altView(m.composePageStr(top, content, footer))
	}

	// Layout: top row = projects, bottom row = sessions + details
	topH := topPaneHeight(m.height)
	botH := bottomPaneHeight(m.height)
	leftW := max(40, m.width*70/100)
	rightW := m.width - leftW

	projW := leftW
	overviewW := rightW
	projPane := renderPane(
		panelStyle(m.styles, m.focus == focusFilters),
		m.styles.header,
		"Projects",
		m.overviewTable.View(),
		projW,
		topH,
	)
	overviewPane := renderPane(
		m.styles.panel,
		m.styles.header,
		"Overview",
		m.renderOverviewStats(filtered, overviewW, topH),
		overviewW,
		topH,
	)
	projects := lipgloss.JoinHorizontal(lipgloss.Top, projPane, overviewPane)
	sessions := renderPane(
		panelStyle(m.styles, m.focus == focusList),
		m.styles.header,
		"Sessions",
		m.renderSessionPane(filtered),
		leftW,
		botH,
	)
	detail := renderPane(
		panelStyle(m.styles, m.focus == focusDetail),
		m.styles.header,
		"Details",
		m.renderDetailPane(),
		rightW,
		botH,
	)

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
	line := h.ShortHelpView(m.keys.shortHelpForFocus(m.focus))
	// Ensure footer is exactly one line so layout math stays consistent.
	if i := strings.Index(line, "\n"); i >= 0 {
		line = line[:i]
	}
	return line
}

func (m Model) overlayPicker(_ string) string {
	width := max(40, m.width*50/100)
	height := min(m.picker.list.Height()+6, m.height-4)
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
	title := m.styles.header.PaddingLeft(1).
		PaddingRight(1).
		MarginBottom(1).
		Render("Keyboard Shortcuts")
	helpText := lipgloss.NewStyle().MarginBottom(1).Render(h.FullHelpView(m.keys.FullHelp()))
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		helpText,
		m.styles.muted.Render("Press ? to close"),
	)
	overlay := m.styles.overlay.Width(width).Height(height).Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}

func (m Model) renderSessionPane(filtered []session.Session) string {
	return m.sessionTable.View()
}

func (m Model) renderDetailPane() string {
	detailW := m.width - m.width*70/100
	innerW := max(10, detailW-2) // subtract pane border
	divider := m.styles.muted.Render(strings.Repeat("─", max(1, innerW)))

	summary := m.selectedSummary()
	if len(summary) > 500 {
		summary = summary[:497] + "..."
	}
	summaryLabel := m.styles.highlight.Render("Summary")
	summaryText := lipgloss.NewStyle().Width(innerW).Render(summary)

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
	sel := m.sessionTable.Cursor()
	if len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		return ""
	}
	return filtered[sel].Summary
}

func (m Model) renderCollapsedPreview(filtered []session.Session) string {
	sel := m.sessionTable.Cursor()
	if len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		return m.styles.muted.PaddingLeft(2).Render("No session selected")
	}
	s := filtered[sel]
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
	spacer := lipgloss.NewStyle().MarginRight(2).Render
	styledParts := make([]string, len(parts))
	for i, p := range parts {
		styledParts[i] = spacer(p)
	}
	line := lipgloss.JoinHorizontal(lipgloss.Top, styledParts...)
	summary := s.Summary
	if len(summary) > 60 {
		summary = summary[:57] + "..."
	}
	content := lipgloss.JoinVertical(lipgloss.Left, line, m.styles.muted.Render(summary))
	return m.styles.subpanel.Width(max(40, m.width-2)).Render(content)
}

func spacer() detailItem { return detailItem{"", ""} }

func (m Model) detailItems(s session.Session) []detailItem {
	// Identity
	items := []detailItem{
		{icon.Tool + " Tool", s.Tool},
		{icon.Project + " Project", cleanProjectName(s.Project)},
		{icon.Active + " Status", s.Status},
		{icon.Model + " Model", valueOrUnknown(s.Model)},
	}

	// Time
	items = append(items, spacer())
	items = append(items,
		detailItem{icon.Clock + " Active", timeAgo(lastActive(s))},
		detailItem{icon.Clock + " Started", s.StartedAt.Format("2006-01-02 15:04:05")},
		detailItem{icon.Clock + " Ended", session.EndedLabel(s.EndedAt, s.Status)},
		detailItem{icon.Clock + " Duration", durationLabel(s)},
	)

	// Source
	var sourceItems []detailItem
	if s.Repo != "" {
		sourceItems = append(sourceItems, detailItem{icon.Repo + " Repo", shortenPath(s.Repo)})
	}
	if s.Branch != "" {
		sourceItems = append(sourceItems, detailItem{icon.Branch + " Branch", s.Branch})
	}
	if len(sourceItems) > 0 {
		items = append(items, spacer())
		items = append(items, sourceItems...)
	}

	// Usage
	var usageItems []detailItem
	if s.TokensIn+s.TokensOut > 0 {
		usageItems = append(
			usageItems,
			detailItem{icon.Token + " Tokens", formatTokens(s.TokensIn, s.TokensOut)},
		)
	}
	if s.CostUSD > 0 {
		usageItems = append(usageItems, detailItem{icon.Cost + " Cost", formatCost(s.CostUSD)})
	}
	if len(usageItems) > 0 {
		items = append(items, spacer())
		items = append(items, usageItems...)
	}

	// IDs
	items = append(items, spacer())
	if s.Slug != "" {
		items = append(items, detailItem{icon.Session + " Slug", s.Slug})
	}
	if s.ParentID != "" {
		items = append(items, detailItem{icon.Parent + " Parent", s.ParentID})
	}
	if len(s.Tags) > 0 {
		items = append(items, detailItem{icon.Tag + " Tags", strings.Join(s.Tags, ", ")})
	}
	items = append(items, detailItem{icon.ID + " ID", s.ID})

	// Metadata
	skip := map[string]bool{"model": true, "branch": true, "version": true}
	var metaItems []detailItem
	for k, v := range s.Meta {
		if !skip[k] {
			metaItems = append(metaItems, detailItem{humanizeKey(k), v})
		}
	}
	if len(metaItems) > 0 {
		items = append(items, spacer())
		items = append(items, detailItem{icon.Meta + " Metadata", ""})
		items = append(items, metaItems...)
	}
	return items
}

func (m Model) renderTopBar(filtered []session.Session) string {
	sep := m.styles.muted.Padding(0, 1).Render("│")
	var searchPart string
	if m.focus == focusSearch {
		searchPart = lipgloss.NewStyle().
			Foreground(lipgloss.Color(nord13)).
			Bold(true).
			Render(icon.Search+" ") +
			m.searchInput.View()
	} else {
		search := strings.TrimSpace(m.searchQuery())
		if search == "" {
			searchPart = m.styles.muted.Render(icon.Search + " all")
		} else {
			prefix := m.styles.highlight.Render(icon.Search + " ")
			term := m.styles.selected.Underline(true).Render(search)
			searchPart = prefix + term
		}
	}
	parts := []string{
		searchPart,
		m.styles.selected.Render(fmt.Sprintf("%d/%d", len(filtered), len(m.sessions))),
	}
	if chips := m.filterChips(); chips != "" {
		parts = append(parts, chips)
	}
	left := strings.Join(parts, sep)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(nord6)).
		Background(lipgloss.Color(nord10)).
		Padding(0, 2).
		Render("AI Dash")
	return lipgloss.NewStyle().Width(m.width - 2).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			left,
			lipgloss.NewStyle().
				Width(m.width-2-lipgloss.Width(left)-lipgloss.Width(title)).
				Render(""),
			title,
		),
	)
}

func (m Model) filterChips() string {
	var chips []string
	chip := m.styles.badge.Padding(0, 1).MarginRight(1)
	chips = append(chips, chip.Render("last "+ageLabel(maxSessionAge)))
	if m.filters.tool != "" {
		chips = append(chips, chip.Render(m.filters.tool))
	}
	if m.filters.project != "" {
		chips = append(chips, chip.Render(cleanProjectName(m.filters.project)))
	}
	if !m.showSubagents {
		chips = append(chips, chip.Render("no subagents"))
	}
	chips = append(chips, m.styles.muted.Render("c to clear"))
	return lipgloss.JoinHorizontal(lipgloss.Top, chips...)
}

func (m Model) sortLabel() string {
	dir := "asc"
	if m.sortDescending {
		dir = "desc"
	}
	return fmt.Sprintf("%s %s", m.sortField, dir)
}

func (m Model) sortHeader(label string, field session.SortField, _ int) string {
	if m.sortField != field {
		return label
	}
	if m.sortDescending {
		return label + " " + icon.SortDesc
	}
	return label + " " + icon.SortAsc
}

func (m Model) projSortHeader(label, field string, _ int) string {
	if m.projSortField != field {
		return label
	}
	if m.projSortDesc {
		return label + " " + icon.SortDesc
	}
	return label + " " + icon.SortAsc
}
