package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/adin/ai-dash/internal/session"
	uilayout "github.com/adin/ai-dash/internal/ui/layout"
	"github.com/adin/ai-dash/internal/ui/overlay"
	"github.com/adin/ai-dash/internal/ui/theme"
	uiutil "github.com/adin/ai-dash/internal/ui/util"
	uiviews "github.com/adin/ai-dash/internal/ui/views"
)

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return altView("")
	}
	if m.err != nil && len(m.sessions) == 0 {
		return altView(m.styles.Error.Render(m.err.Error()))
	}

	filtered := m.filteredSessions()

	// Layout budget: topBar(1) + content(variable) + footer(1) = m.height
	contentH := uilayout.ContentHeight(m.height)
	top := " " + m.renderTopBar(filtered)
	footer := " " + m.renderFooter()

	if len(m.sessions) == 0 {
		body := uiviews.EmptySessions(m.styles, m.width, contentH, "No sessions loaded.")
		return altView(uiviews.Page(top, body, footer))
	}
	if len(filtered) == 0 {
		body := uiviews.NoMatches(
			m.styles,
			m.width,
			contentH,
			m.styles.Muted.Render("No matches. Press c to clear or / to search."),
		)
		return altView(uiviews.Page(top, body, footer))
	}

	if m.detailCollapsed {
		previewH := 2
		tableH := contentH - previewH
		content := uiviews.CollapsedSessions(
			m.styles,
			m.width,
			contentH,
			tableH,
			m.focus == focusList,
			m.renderSessionPane(),
			m.renderCollapsedPreview(filtered),
		)
		return altView(uiviews.Page(top, content, footer))
	}

	// Layout: top row = projects, bottom row = sessions + details
	topH := uilayout.TopPaneHeight(m.height)
	botH := uilayout.BottomPaneHeight(m.height)
	leftW := max(40, m.width*70/100)
	rightW := m.width - leftW

	content := uiviews.MainDashboard(
		m.styles,
		m.focus == focusFilters,
		m.focus == focusList,
		leftW,
		rightW,
		topH,
		botH,
		m.overviewTable.View(),
		m.renderOverviewStats(filtered),
		m.renderSessionPane(),
		m.renderDetailPane(),
	)

	page := uiviews.Page(top, content, footer)
	if m.picker.active {
		page = overlay.Picker(
			m.width,
			m.height,
			m.picker.list.Height(),
			m.styles.Overlay,
			m.picker.list.View(),
		)
	}
	if m.showSources {
		page = overlay.Sources(
			m.width,
			m.height,
			m.styles.Overlay,
			m.styles.Header,
			m.styles.Muted,
			m.sourceTable.View(),
		)
	}
	if m.showHelp {
		h := m.help
		h.SetWidth(max(30, min(60, m.width-4)) - 6)
		page = overlay.Help(
			m.width,
			m.height,
			m.styles.Overlay,
			m.styles.Header,
			m.styles.Muted,
			h.FullHelpView(m.keys.FullHelp()),
		)
	}
	return altView(page)
}

func altView(s string) tea.View {
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func (m Model) renderFooter() string {
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

func (m Model) renderSessionPane() string {
	return m.sessionTable.View()
}

func (m Model) renderDetailPane() string {
	return uiviews.DetailPane(
		m.styles,
		m.width,
		m.selectedSummary(),
		m.detailTable.View(),
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
		return m.styles.Muted.PaddingLeft(2).Render("No session selected")
	}
	s := filtered[sel]
	parts := []string{
		m.styles.Highlight.Render(s.Project),
		s.Tool,
		s.Status,
		uiutil.ValueOrUnknown(s.Model),
		uiutil.DurationLabel(s),
	}
	if s.CostUSD > 0 {
		parts = append(parts, uiutil.FormatCost(s.CostUSD))
	}
	if s.TokensIn+s.TokensOut > 0 {
		parts = append(parts, uiutil.FormatTokens(s.TokensIn, s.TokensOut))
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
	content := lipgloss.JoinVertical(lipgloss.Left, line, m.styles.Muted.Render(summary))
	return m.styles.Subpanel.Width(max(40, m.width-2)).Render(content)
}

func (m Model) renderTopBar(filtered []session.Session) string {
	return uiviews.TopBar(
		m.styles,
		m.width,
		m.focus == focusSearch,
		m.searchInput.View(),
		strings.TrimSpace(m.searchQuery()),
		len(filtered),
		len(m.sessions),
		m.filterChips(),
	)
}

func (m Model) filterChips() string {
	var chips []string
	chip := m.styles.Badge.Padding(0, 1).MarginRight(1)
	chips = append(chips, chip.Render("last "+ageLabel(maxSessionAge)))
	if m.filters.tool != "" {
		chips = append(chips, chip.Render(m.filters.tool))
	}
	if m.filters.project != "" {
		chips = append(chips, chip.Render(uiutil.CleanProjectName(m.filters.project)))
	}
	if !m.showSubagents {
		chips = append(chips, chip.Render("no subagents"))
	}
	chips = append(chips, m.styles.Muted.Render("c to clear"))
	return lipgloss.JoinHorizontal(lipgloss.Top, chips...)
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
		return label + " " + theme.SortDesc
	}
	return label + " " + theme.SortAsc
}

func (m Model) projSortHeader(label, field string) string {
	if m.projSortField != field {
		return label
	}
	if m.projSortDesc {
		return label + " " + theme.SortDesc
	}
	return label + " " + theme.SortAsc
}
