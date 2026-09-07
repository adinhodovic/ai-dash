package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/adinhodovic/ai-dash/internal/session"
	uilayout "github.com/adinhodovic/ai-dash/internal/ui/layout"
	"github.com/adinhodovic/ai-dash/internal/ui/overlay"
	"github.com/adinhodovic/ai-dash/internal/ui/theme"
	uiutil "github.com/adinhodovic/ai-dash/internal/ui/util"
	uiviews "github.com/adinhodovic/ai-dash/internal/ui/views"
)

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return altView("")
	}
	if m.err != nil && len(m.sessions) == 0 {
		return altView(m.styles.Error.Render(m.err.Error()))
	}

	filtered := m.filteredSessions()

	// Layout budget: searchBox(4) + content(variable) + footer(1) = m.height
	contentH := uilayout.ContentHeight(m.height)
	// The search box is a fixed-height bordered box (see TopBar), so it's
	// always exactly 4 lines regardless of how long its content is —
	// overflow is truncated per-line inside TopBar itself.
	top := m.renderTopBar(filtered)
	footer := " " + m.renderFooter()

	if len(m.sessions) == 0 {
		body := uiviews.EmptySessions(m.styles, m.width, contentH, "No sessions loaded.")
		return altView(uiviews.Page(top, body, footer))
	}
	if len(filtered) == 0 {
		msg := "No matches. Press c to clear or / to search."
		if m.showAttentionOnly {
			msg = "All caught up — no sessions need attention right now."
		}
		body := uiviews.NoMatches(
			m.styles,
			m.width,
			contentH,
			m.styles.Muted.Render(msg),
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

	botH := uilayout.BottomPaneHeight(m.height)
	leftW := max(40, m.width*70/100)
	rightW := m.width - leftW

	content := uiviews.MainDashboard(
		m.styles,
		m.focus == focusList,
		leftW,
		rightW,
		botH,
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
		page = overlay.Help(
			m.width,
			m.height,
			m.styles.Overlay,
			m.styles.Header,
			m.styles.Muted,
			m.renderHelpBody(),
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
	header := renderSessionHeader(m, m.sessionViewport.Width())
	return header + "\n\n" + m.sessionViewport.View()
}

func (m Model) renderDetailPane() string {
	return uiviews.DetailPane(
		m.styles,
		m.width,
		m.selectedSummary(),
		m.detailTable.View(),
	)
}

func (m Model) selectedSummary() string {
	filtered := m.filteredSessions()
	sel := m.sessionCursor
	if len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		return ""
	}
	return filtered[sel].Summary
}

func (m Model) renderCollapsedPreview(filtered []session.Session) string {
	sel := m.sessionCursor
	if len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		return m.styles.Muted.PaddingLeft(2).Render("No session selected")
	}
	s := filtered[sel]
	status := s.Status
	if reason := session.Attention(s); reason != session.AttentionNone {
		status = theme.AttentionStyle().Render(theme.Attention+" ") +
			theme.ReasonStyle(reason).Render(attentionLabel(s))
	}
	parts := []string{
		m.styles.Highlight.Render(s.Project),
		s.Tool,
		status,
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
	return m.styles.Subpanel.
		Padding(0, uilayout.PanePadding).
		Width(max(40, m.width-2-2*uilayout.PanePadding)).
		Render(content)
}

func (m Model) renderTopBar(filtered []session.Session) string {
	var searchPart string
	switch {
	case m.renaming:
		searchPart = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Nord13)).Bold(true).Render("Rename ") +
			m.renameInput.View()
	case m.focus == focusSearch:
		searchPart = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Nord13)).Bold(true).Render(theme.Search+" ") +
			m.searchInput.View()
	case strings.TrimSpace(m.searchQuery()) == "":
		searchPart = m.styles.Muted.Render(theme.Search + " press / to search")
	default:
		prefix := m.styles.Highlight.Render(theme.Search + " ")
		term := m.styles.Selected.Underline(true).Render(strings.TrimSpace(m.searchQuery()))
		searchPart = prefix + term
	}
	searchLine := searchPart

	var active int
	activeTools := map[string]bool{}
	for _, s := range m.sessions {
		if session.IsLive(s) {
			active++
			activeTools[uiutil.Capitalize(s.Tool)] = true
		}
	}
	results := fmt.Sprintf("%s %d/%d sessions", theme.Session, len(filtered), len(m.sessions))
	if active > 0 {
		tools := make([]string, 0, len(activeTools))
		for t := range activeTools {
			tools = append(tools, t)
		}
		slices.Sort(tools)
		results += fmt.Sprintf(" · %d active (%s)", active, strings.Join(tools, ", "))
	}
	resultsText := m.styles.Highlight.Render(results)

	strip := uiviews.TopStrip(m.styles, m.width, m.meta.Version, resultsText)
	bar := uiviews.TopBar(m.styles, m.width, searchLine, m.filterAndShortcutsLine())
	return lipgloss.JoinVertical(lipgloss.Left, strip, "", bar)
}

// filterAndShortcutsLine shows what's currently filtered, as chips — the
// keys that change each one used to be spelled out right here, but that's
// what "?" is for, and having both just competed for attention with the
// thing that actually matters: what's applied right now.
func (m Model) filterAndShortcutsLine() string {
	prefix := m.styles.Highlight.Render(theme.Filter + " ")
	return prefix + m.filterChips()
}

// filterChips renders what's currently narrowing the list. Filters actually
// changed from their default get a bold accent-colored chip — the thing
// that answers "what's applied" at a glance; filters still sitting at their
// default get a plain muted chip if they're worth showing as context at
// all (a default that's simply "off," like subagents, isn't — there's
// nothing to say about it until it's turned on).
func (m Model) filterChips() string {
	var active, info []string
	activeChip := m.styles.FilterActive.Padding(0, 1).MarginRight(2)
	infoChip := m.styles.Muted.Padding(0, 1).MarginRight(2)

	if m.showAttentionOnly {
		active = append(active, activeChip.Render(theme.Attention+" needs attention"))
	} else if n := len(attentionKeys(m.sessions)); n > 0 {
		active = append(
			active,
			theme.AttentionStyle().Bold(true).Padding(0, 1).MarginRight(2).
				Render(fmt.Sprintf("%s %d need attention", theme.Attention, n)),
		)
	}
	if m.showActiveOnly {
		active = append(active, activeChip.Render(theme.Active+" active only"))
	}
	if m.filters.tool != "" {
		active = append(active, activeChip.Render(theme.Tool+" "+m.filters.tool))
	}
	if m.filters.project != "" {
		active = append(active, activeChip.Render(theme.Project+" "+uiutil.CleanProjectName(m.filters.project)))
	}
	if m.showSubagents {
		active = append(active, activeChip.Render(theme.Parent+" incl. subagents"))
	}
	if maxSessionAge != m.meta.Config.DefaultAgeFilterDuration() {
		active = append(active, activeChip.Render(theme.Clock+" last "+ageLabel(maxSessionAge)))
	} else {
		info = append(info, infoChip.Render(theme.Clock+" last "+ageLabel(maxSessionAge)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, append(active, info...)...)
}

// renderHelpBody renders the "?" overlay as labeled sections (Navigate,
// Session, Filter, Sort, General) in two columns, rather than bubbles'
// stock FullHelpView — an unlabeled grid of key/description pairs that
// doesn't say what any group of keys is actually for.
func (m Model) renderHelpBody() string {
	const helpKeyWidth = 8
	renderColumn := func(sections []helpSection) string {
		var blocks []string
		for _, sec := range sections {
			lines := []string{m.styles.Highlight.Render(sec.title)}
			for _, b := range sec.bindings {
				h := b.Help()
				if h.Key == "" {
					continue
				}
				lines = append(
					lines,
					"  "+m.styles.Selected.Render(padField(h.Key, helpKeyWidth))+m.styles.Muted.Render(h.Desc),
				)
			}
			blocks = append(blocks, lipgloss.JoinVertical(lipgloss.Left, lines...))
		}
		return strings.Join(blocks, "\n\n")
	}
	columns := m.keys.helpColumns()
	left := lipgloss.NewStyle().MarginRight(4).Render(renderColumn(columns[0][:]))
	right := renderColumn(columns[1][:])
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) sortLabel() string {
	dir := "asc"
	if m.sortDescending {
		dir = "desc"
	}
	return fmt.Sprintf("%s %s", m.sortField, dir)
}
