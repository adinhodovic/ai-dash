package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/ui/theme"
	uiutil "github.com/adinhodovic/ai-dash/internal/ui/util"
)

// Each session occupies sessionRowLines of content — a context line leading
// with the project, then a trailing cluster of metadata (tool, last active,
// status) — and a bold title (the summary), gh-dash's extended-title pattern
// (repo/context first, title second) applied without gh-dash's own custom
// table (see sessions.go for why). The title gets up to titleMaxLines lines
// of its own — long summaries wrap instead of being cut to one line — but no
// more than that, so a long summary can't grow into where the next session's
// context line (and its Tool zone) would otherwise start; short summaries
// are blank-padded up to the same height so every row's context line lands
// at a fixed, predictable offset regardless of content. Then a blank
// sessionRowGap line separates it from the next session. sessionRowStride is
// the total distance from one row's top to the next.
const (
	titleMaxLines    = 2
	sessionRowLines  = 1 + titleMaxLines
	sessionRowGap    = 1
	sessionRowStride = sessionRowLines + sessionRowGap
)

// Fixed zone widths for line 1, so fields land in the same column across
// every row instead of drifting with each other's content length — the
// "columns" a real table would give you, without needing one. The header
// line (renderSessionHeader) uses these same widths so labels line up with
// the data below them.
const (
	rowGutterWidth  = 2 // selection indicator
	toolZoneWidth   = 11
	timeZoneWidth   = 16
	statusZoneWidth = 18
	zoneGap         = 2
	// metaGap separates the path (identity) from the trailing metadata
	// cluster (tool/time/status) — wider than zoneGap so the two groups
	// read as distinct, not just four same-spaced fields in a row.
	metaGap = 4
)

// projectZoneWidth is whatever's left after the trailing metadata cluster
// and its gaps — the one flexible zone, since paths vary widely in length.
func projectZoneWidth(contentWidth int) int {
	used := toolZoneWidth + timeZoneWidth + statusZoneWidth + 2*zoneGap + metaGap
	return max(10, contentWidth-used)
}

// padField truncates or right-pads s to exactly width runes, so fixed-width
// zones stay aligned regardless of content length.
func padField(s string, width int) string {
	s = uiutil.Truncate(s, width)
	if pad := width - len([]rune(s)); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// renderSessionHeader labels each zone (mirroring a real table's column
// headers) and marks whichever one the list is currently sorted by with a
// direction arrow — the sort-cycling keys (s/[/]/=) act on these same
// fields, so the header doubles as a legend for what they do.
func renderSessionHeader(m Model, width int) string {
	contentWidth := max(10, width-rowGutterWidth)
	label := func(text string, field session.SortField, fieldWidth int) string {
		if m.sortField == field {
			arrow := theme.SortAsc
			if m.sortDescending {
				arrow = theme.SortDesc
			}
			return m.styles.Highlight.Render(padField(text+" "+arrow, fieldWidth))
		}
		return m.styles.Muted.Render(padField(text, fieldWidth))
	}
	gap := m.styles.Muted.Render(strings.Repeat(" ", zoneGap))
	metaGapStr := m.styles.Muted.Render(strings.Repeat(" ", metaGap))
	zones := []string{
		label("Session", session.SortProject, projectZoneWidth(contentWidth)),
		label("Tool", session.SortTool, toolZoneWidth),
		label("Last Active", session.SortUpdated, timeZoneWidth),
		label("Status", session.SortStatus, statusZoneWidth),
	}
	joined := zones[0] + metaGapStr + strings.Join(zones[1:], gap)
	return strings.Repeat(" ", rowGutterWidth) + ansi.Truncate(joined, contentWidth, "…")
}

// renderSessionRow builds one session's 2-line block, truncated to width.
//
// Selection is shown as a background color, but — unlike wrapping an
// already-colored composition in an outer background style afterward, which
// broke (each fragment's own ANSI reset also cleared the outer background
// the instant it fired, leaving the highlight visible only in the gaps
// between colored fragments) — every fragment here bakes its own
// Background(...) in alongside its Foreground(...) when selected. Each
// fragment is a single self-contained Render call, so there's no outer wrap
// for an inner reset to fight with, and every field keeps its real color on
// a fully-highlighted row.
func renderSessionRow(m Model, s session.Session, selected bool, width int) string {
	status := uiutil.SessionStatusLabel(s)
	reason := session.Attention(s)
	var icon, statusText string
	var statusStyle lipgloss.Style
	if reason != session.AttentionNone {
		icon = theme.Attention
		statusText = attentionLabel(s)
		statusStyle = theme.ReasonStyle(reason)
	} else {
		icon = theme.StateGlyph(status)
		statusText = status
		statusStyle = theme.StatusStyle(status)
	}

	dim := m.styles.Muted
	titleStyle := m.styles.Selected
	matchStyle := m.styles.Match
	gapStyle := lipgloss.NewStyle()
	gutterStyle := lipgloss.NewStyle()
	if selected {
		bg := lipgloss.Color(theme.ColorSelectBg)
		statusStyle = statusStyle.Background(bg)
		dim = dim.Background(bg)
		titleStyle = titleStyle.Background(bg)
		gapStyle = gapStyle.Background(bg)
		gutterStyle = gutterStyle.Foreground(lipgloss.Color(theme.ColorSelectFg)).Background(bg)
		// Leave matchStyle's own background (its highlight color) alone even
		// when selected — it needs to stay visually distinct from the row's
		// selection background, not blend into it.
	}

	contentWidth := max(10, width-rowGutterWidth)
	gap := gapStyle.Render(strings.Repeat(" ", zoneGap))
	metaGapStr := gapStyle.Render(strings.Repeat(" ", metaGap))
	projectZone := dim.Render(
		padField(
			theme.Project+" "+uiutil.CleanProjectName(s.Project),
			projectZoneWidth(contentWidth),
		),
	)
	toolZone := dim.Render(padField(theme.Tool+" "+uiutil.Capitalize(s.Tool), toolZoneWidth))
	timeZone := dim.Render(
		padField(theme.Clock+" "+uiutil.TimeAgo(uiutil.LastActive(s)), timeZoneWidth),
	)
	statusZone := statusStyle.Render(padField(icon+" "+statusText, statusZoneWidth))
	context := projectZone + metaGapStr + strings.Join(
		[]string{toolZone, timeZone, statusZone},
		gap,
	)

	// The title sits under the Session zone only — it shouldn't run out past
	// where the Tool column starts on the line above it.
	titleWidth := projectZoneWidth(contentWidth)
	title := uiutil.CleanSummary(s.Summary)
	gutter := gutterStyle.Render("  ")
	if selected {
		gutter = gutterStyle.Bold(true).Render("▌ ")
	}
	// Selection highlight still fills the row's full width — only the text
	// itself is held back from the Tool column, not the highlight behind it,
	// so a selected row reads as one solid block across all its lines.
	trailingFill := gapStyle.Render(strings.Repeat(" ", max(0, contentWidth-titleWidth)))

	lines := make([]string, 0, 1+titleMaxLines)
	lines = append(lines, gutter+ansi.Truncate(context, contentWidth, "…"))
	for _, titleLine := range wrapTitle(title, titleWidth) {
		rendered := renderTitleLine(titleLine, m.searchQuery(), titleStyle, matchStyle, titleWidth)
		lines = append(lines, gutter+rendered+trailingFill)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// wrapTitle word-wraps title into exactly titleMaxLines lines (padding with
// blanks if it's shorter), so every row's title occupies the same fixed
// height regardless of content — the thing that lets sessionRowStride stay a
// constant instead of varying per row. Content beyond titleMaxLines lines is
// cut, with an ellipsis marking that it was.
func wrapTitle(title string, width int) []string {
	wrapped := wordWrap(title, width)
	lines := make([]string, titleMaxLines)
	for i := range lines {
		if i < len(wrapped) {
			lines[i] = wrapped[i]
		}
	}
	if len(wrapped) > titleMaxLines {
		lines[titleMaxLines-1] = capLine(lines[titleMaxLines-1], width)
	}
	return lines
}

// wordWrap greedily packs words onto lines no wider than width, breaking a
// single word longer than width across lines rather than overflowing it.
func wordWrap(text string, width int) []string {
	width = max(1, width)
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := ""
	for _, w := range words {
		for len([]rune(w)) > width {
			if cur != "" {
				lines = append(lines, cur)
				cur = ""
			}
			r := []rune(w)
			lines = append(lines, string(r[:width]))
			w = string(r[width:])
		}
		switch {
		case cur == "":
			cur = w
		case len([]rune(cur))+1+len([]rune(w)) <= width:
			cur += " " + w
		default:
			lines = append(lines, cur)
			cur = w
		}
	}
	return append(lines, cur)
}

// capLine marks a line as cut short (there was more text past it) by
// swapping its final rune for an ellipsis, rather than just appending one
// and risking pushing the line past width.
func capLine(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		r = r[:max(0, width-1)]
	}
	return string(r) + "…"
}

// renderTitleLine renders one already width-bounded line of the title,
// highlighting the substring that matches the active search query (if any)
// in matchStyle instead of base — so a search result shows *why* it
// matched, not just that it did. Falls back to a plain base-styled render
// when there's no query or no literal substring match (e.g. the result only
// matched via fuzzy scoring, or via a field other than the summary).
// Manually pads to width in base's own style afterward, rather than relying
// on lipgloss's Width() auto-padding, since splitting the text into two
// differently-styled fragments reintroduces the same nested-ANSI-reset risk
// Width() padding would otherwise paper over.
func renderTitleLine(line, query string, base, match lipgloss.Style, width int) string {
	query = strings.TrimSpace(query)
	var rendered string
	if query != "" {
		if idx := strings.Index(strings.ToLower(line), strings.ToLower(query)); idx >= 0 {
			before, matched, after := line[:idx], line[idx:idx+len(query)], line[idx+len(query):]
			rendered = base.Render(before) + match.Render(matched) + base.Render(after)
		}
	}
	if rendered == "" {
		rendered = base.Render(line)
	}
	if pad := width - len([]rune(line)); pad > 0 {
		rendered += base.Render(strings.Repeat(" ", pad))
	}
	return rendered
}

// followCursor returns the Y-offset that keeps the selected session's block
// visible within the viewport, scrolling the minimum amount needed — up if
// its top line is above the window, down if its bottom line is below it,
// unchanged otherwise. Pure function so the scroll math is easy to test.
func followCursor(cursor, count, viewportHeight, yOffset int) int {
	if count <= 0 || viewportHeight <= 0 {
		return 0
	}
	top := cursor * sessionRowStride
	bottom := top + sessionRowLines - 1
	if top < yOffset {
		return top
	}
	if bottom >= yOffset+viewportHeight {
		return bottom - viewportHeight + 1
	}
	return yOffset
}

// moveSessionCursor clamps the cursor into range, re-renders the list with
// the new selection, and scrolls to keep it visible.
func moveSessionCursor(m *Model, delta int, filtered []session.Session) {
	if len(filtered) == 0 {
		m.sessionCursor = 0
		return
	}
	m.sessionCursor = max(0, min(len(filtered)-1, m.sessionCursor+delta))
	m.syncTable(filtered)
}
