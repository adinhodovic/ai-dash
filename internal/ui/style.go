package ui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

// Color palette — single source of truth.
const (
	colorMuted     = "246"
	colorStrong    = "230"
	colorHighlight = "81"
	colorError     = "203"
	colorMatchFg   = "16"
	colorMatchBg   = "221"
	colorHeaderBg  = "24"
	colorBadgeFg   = "24"
	colorBadgeBg   = "151"
	colorBorder    = "240"
	colorActive    = "81"
	colorSubtle    = "238"
	colorTitleBg   = "236"
	colorSelectFg  = "229"
	colorSelectBg  = "57"
)

type styles struct {
	frame     lipgloss.Style
	header    lipgloss.Style
	muted     lipgloss.Style
	highlight lipgloss.Style
	match     lipgloss.Style
	selected  lipgloss.Style
	badge     lipgloss.Style
	error     lipgloss.Style
	panel     lipgloss.Style
	active    lipgloss.Style
	subpanel  lipgloss.Style
	titlebar  lipgloss.Style
	rule      lipgloss.Style
	overlay   lipgloss.Style
}

func newStyles() styles {
	return styles{
		frame: lipgloss.NewStyle(),
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorStrong)).
			Background(lipgloss.Color(colorHeaderBg)),
		muted:     lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted)),
		highlight: lipgloss.NewStyle().Foreground(lipgloss.Color(colorHighlight)).Bold(true),
		match: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMatchFg)).
			Background(lipgloss.Color(colorMatchBg)).
			Bold(true),
		selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorStrong)).
			Bold(true),
		panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(colorBorder)),
		active: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(colorActive)),
		subpanel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(colorSubtle)),
		titlebar: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorStrong)).
			Background(lipgloss.Color(colorTitleBg)),
		badge: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBadgeFg)).
			Background(lipgloss.Color(colorBadgeBg)),
		error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError)).
			Bold(true),
		rule: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorHeaderBg)),
		overlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorActive)).
			Padding(1, 2),
	}
}

// tableStyles returns consistent table styles using the palette.
func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(colorSelectFg)).
		Background(lipgloss.Color(colorSelectBg)).
		Bold(false)
	s.Cell = s.Cell.Padding(0, 1)
	return s
}

// helpStyles returns consistent help key styles using the palette.
func applyHelpStyles(h *help.Model) {
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHighlight)).Bold(true)
	h.Styles.FullKey = h.Styles.ShortKey
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color(colorStrong))
	h.Styles.FullDesc = h.Styles.ShortDesc
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBorder))
	h.Styles.FullSeparator = h.Styles.ShortSeparator
}
