package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/adin/ai-dash/internal/ui/theme"
)

func EmptySessions(styles theme.Styles, width, contentH int, message string) string {
	return renderPane(styles.Panel, styles.Header, "Sessions", message, width, contentH)
}

func NoMatches(styles theme.Styles, width, contentH int, message string) string {
	return renderPane(styles.Panel, styles.Header, "Sessions", message, width, contentH)
}

func CollapsedSessions(
	styles theme.Styles,
	width, contentH, tableH int,
	focusList bool,
	sessionPane, preview string,
) string {
	tablePane := renderPane(
		panelStyle(styles, focusList),
		styles.Header,
		"Sessions",
		sessionPane,
		width,
		tableH,
	)
	preview = ansi.Truncate(preview, width, "")
	tLines := strings.Split(tablePane, "\n")
	if len(tLines) > tableH {
		tLines = tLines[:tableH]
	}
	tLines = append(tLines, preview)
	if len(tLines) > contentH {
		tLines = tLines[:contentH]
	}
	return lipgloss.JoinVertical(lipgloss.Left, tLines...)
}

func MainDashboard(
	styles theme.Styles,
	focusFilters, focusList bool,
	leftW, rightW, topH, botH int,
	overviewTable, overviewStats, sessionPane, detailPane string,
) string {
	projPane := renderPane(
		panelStyle(styles, focusFilters),
		styles.Header,
		"Projects",
		overviewTable,
		leftW,
		topH,
	)
	statsPane := renderPane(
		styles.Panel,
		styles.Header,
		"Overview",
		overviewStats,
		rightW,
		topH,
	)
	projects := lipgloss.JoinHorizontal(lipgloss.Top, projPane, statsPane)
	sessions := renderPane(
		panelStyle(styles, focusList),
		styles.Header,
		"Sessions",
		sessionPane,
		leftW,
		botH,
	)
	details := renderPane(
		styles.Panel,
		styles.Header,
		"Details",
		detailPane,
		rightW,
		botH,
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		projects,
		lipgloss.JoinHorizontal(lipgloss.Top, sessions, details),
	)
}

func DetailPane(styles theme.Styles, width int, summary, detailTable, relatedTable string) string {
	detailW := width - width*70/100
	innerW := max(10, detailW-2)
	divider := styles.Muted.Render(strings.Repeat("─", max(1, innerW)))
	if len(summary) > 500 {
		summary = summary[:497] + "..."
	}
	summaryLabel := styles.Highlight.Render("Summary")
	summaryText := lipgloss.NewStyle().Width(innerW).Render(summary)
	relatedLabel := styles.Highlight.Render("Related Sessions")
	return lipgloss.JoinVertical(
		lipgloss.Left,
		summaryLabel,
		summaryText,
		divider,
		detailTable,
		divider,
		relatedLabel,
		relatedTable,
	)
}

func TopBar(
	styles theme.Styles,
	width int,
	searchFocused bool,
	searchInputView, searchQuery string,
	filteredCount, totalCount int,
	chips string,
) string {
	sep := styles.Muted.Padding(0, 1).Render("│")
	var searchPart string
	if searchFocused {
		searchPart = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Nord13)).
			Bold(true).
			Render(theme.Search+" ") + searchInputView
	} else if searchQuery == "" {
		searchPart = styles.Muted.Render(theme.Search + " all")
	} else {
		prefix := styles.Highlight.Render(theme.Search + " ")
		term := styles.Selected.Underline(true).Render(searchQuery)
		searchPart = prefix + term
	}
	parts := []string{
		searchPart,
		styles.Selected.Render(fmt.Sprintf("%d/%d", filteredCount, totalCount)),
	}
	if chips != "" {
		parts = append(parts, chips)
	}
	left := strings.Join(parts, sep)
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(theme.Nord6)).
		Background(lipgloss.Color(theme.Nord10)).
		Padding(0, 2).
		Render("AI Dash")
	return lipgloss.NewStyle().Width(width - 2).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			left,
			lipgloss.NewStyle().
				Width(width-2-lipgloss.Width(left)-lipgloss.Width(title)).
				Render(""),
			title,
		),
	)
}

func Page(top, content, footer string) string {
	return lipgloss.JoinVertical(lipgloss.Left, top, content, footer)
}

func renderPane(border, header lipgloss.Style, title, body string, width, height int) string {
	titleLine := header.PaddingRight(1).PaddingLeft(1).MarginBottom(1).Render(title)
	content := lipgloss.JoinVertical(lipgloss.Left, titleLine, body)
	return border.Width(width).Height(height).MaxHeight(height).Render(content)
}

func panelStyle(styles theme.Styles, active bool) lipgloss.Style {
	if active {
		return styles.Active
	}
	return styles.Panel
}
