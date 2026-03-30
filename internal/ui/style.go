package ui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

// Nord palette — single source of truth.
// https://www.nordtheme.com/docs/colors-and-palettes
const (
	// Polar Night — dark base tones
	nord0  = "#2e3440"
	nord1  = "#3b4252"
	nord2  = "#434c5e"
	nord3  = "#4c566a"

	// Snow Storm — light text tones
	nord4  = "#d8dee9"
	nord5  = "#e5e9f0"
	nord6  = "#eceff4"

	// Frost — blue accents
	nord7  = "#8fbcbb"
	nord8  = "#88c0d0"
	nord9  = "#81a1c1"
	nord10 = "#5e81ac"

	// Aurora — status/accent colors
	nord11 = "#bf616a" // red
	nord12 = "#d08770" // orange
	nord13 = "#ebcb8b" // yellow
	nord14 = "#a3be8c" // green
	nord15 = "#b48ead" // purple
)

// Semantic color mapping.
const (
	colorText      = nord5
	colorMuted     = nord4
	colorStrong    = nord6
	colorHighlight = nord8
	colorError     = nord11
	colorMatchFg   = nord0
	colorMatchBg   = nord13
	colorHeaderFg  = nord6
	colorHeaderBg  = nord2
	colorBadgeFg   = nord0
	colorBadgeBg   = nord13 // yellow — stands out as "filter active"
	colorBorder    = nord3
	colorActive    = nord6
	colorSubtle    = nord2
	colorSelectFg  = nord0
	colorSelectBg  = nord8
	colorHelpDesc  = nord5
	colorHelpSep   = nord3
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
			Foreground(lipgloss.Color(colorHeaderFg)).
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
			Background(lipgloss.Color(colorHeaderBg)),
		badge: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBadgeFg)).
			Background(lipgloss.Color(colorBadgeBg)),
		error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError)).
			Bold(true),
		rule: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBorder)),
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
		Bold(true).
		Foreground(lipgloss.Color(colorStrong))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(colorSelectFg)).
		Background(lipgloss.Color(colorSelectBg)).
		Bold(false)
	s.Cell = s.Cell.Padding(0, 1)
	return s
}

// applyHelpStyles applies the palette to the help component.
func applyHelpStyles(h *help.Model) {
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHighlight)).Bold(true)
	h.Styles.FullKey = h.Styles.ShortKey
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHelpDesc))
	h.Styles.FullDesc = h.Styles.ShortDesc
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHelpSep))
	h.Styles.FullSeparator = h.Styles.ShortSeparator
}
