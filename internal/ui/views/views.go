package views

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	uilayout "github.com/adinhodovic/ai-dash/internal/ui/layout"
	"github.com/adinhodovic/ai-dash/internal/ui/theme"
)

func EmptySessions(styles theme.Styles, width, contentH int, message string) string {
	return renderPane(styles.Panel, message, width, contentH)
}

func NoMatches(styles theme.Styles, width, contentH int, message string) string {
	return renderPane(styles.Panel, message, width, contentH)
}

func CollapsedSessions(
	styles theme.Styles,
	width, contentH, tableH int,
	focusList bool,
	sessionPane, preview string,
) string {
	tablePane := renderPane(
		panelStyle(styles, focusList),
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
	focusList bool,
	leftW, rightW, botH int,
	sessionPane, detailPane string,
) string {
	sessions := renderPane(
		panelStyle(styles, focusList),
		sessionPane,
		leftW,
		botH,
	)
	details := renderPane(
		styles.Panel,
		detailPane,
		rightW,
		botH,
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, sessions, details)
}

func DetailPane(styles theme.Styles, width int, summary, detailTable string) string {
	detailW := width - width*70/100
	innerW := max(10, detailW-2-2*uilayout.PanePadding)
	divider := styles.Muted.Render(strings.Repeat("─", max(1, innerW)))
	if len(summary) > 500 {
		summary = summary[:497] + "..."
	}
	summaryLabel := styles.Highlight.Render("Summary")
	summaryText := lipgloss.NewStyle().Width(innerW).Render(summary)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		summaryLabel,
		summaryText,
		divider,
		detailTable,
	)
}

// TopStrip renders the one-line masthead above the search window: the
// results count on the left (it reads fine on its own — no "Results:"
// label needed), a small plain-text brand mark and the build version
// grouped together on the right. Pixel art was tried at three different
// sizes and never actually read as "AI Dash" at terminal scale — plain
// text is small, legible, and done.
func TopStrip(styles theme.Styles, width int, version, results string) string {
	logo := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Nord8)).Render("AI Dash")
	right := lipgloss.JoinHorizontal(lipgloss.Top, logo, "  ", styles.Muted.Render(version))

	innerW := max(10, width-2*uilayout.PanePadding)
	leftW := lipgloss.Width(results)
	rightMax := max(0, innerW-leftW-1) // leave room for at least one gap column
	if lipgloss.Width(right) > rightMax {
		versionW := max(0, rightMax-lipgloss.Width(logo)-2)
		right = lipgloss.JoinHorizontal(
			lipgloss.Top,
			logo,
			"  ",
			styles.Muted.Render(ansi.Truncate(version, versionW, "…")),
		)
	}
	gapW := max(0, innerW-leftW-lipgloss.Width(right))
	line := lipgloss.JoinHorizontal(
		lipgloss.Top,
		results,
		lipgloss.NewStyle().Width(gapW).Render(""),
		right,
	)

	return lipgloss.NewStyle().
		Padding(0, uilayout.PanePadding).
		Width(width).
		Render(ansi.Truncate(line, innerW, "…"))
}

// TopBar renders the search window: Search fixed-width on the left (so it
// doesn't slide around as you type), Filters pushed to the right edge of
// the window — the two ends of the same bar rather than crowded together
// right after Search.
func TopBar(styles theme.Styles, width int, searchLine, filtersLine string) string {
	innerW := max(10, width-2-2*uilayout.PanePadding)
	gapW := max(1, innerW-lipgloss.Width(searchLine)-lipgloss.Width(filtersLine))
	line := searchLine + strings.Repeat(" ", gapW) + filtersLine

	return styles.Panel.
		Padding(0, uilayout.PanePadding).
		Width(width).
		Height(1).
		Render(ansi.Truncate(line, innerW, "…"))
}

func Page(top, content, footer string) string {
	return lipgloss.JoinVertical(lipgloss.Left, top, content, footer)
}

func renderPane(border lipgloss.Style, body string, width, height int) string {
	inner := lipgloss.NewStyle().Padding(0, uilayout.PanePadding).Render(body)
	return border.Width(width).Height(height).MaxHeight(height).Render(inner)
}

func panelStyle(styles theme.Styles, active bool) lipgloss.Style {
	if active {
		return styles.Active
	}
	return styles.Panel
}
