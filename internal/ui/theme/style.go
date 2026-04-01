package theme

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

const (
	Nord0  = "#2e3440"
	Nord1  = "#3b4252"
	Nord2  = "#434c5e"
	Nord3  = "#4c566a"
	Nord4  = "#d8dee9"
	Nord5  = "#e5e9f0"
	Nord6  = "#eceff4"
	Nord7  = "#8fbcbb"
	Nord8  = "#88c0d0"
	Nord9  = "#81a1c1"
	Nord10 = "#5e81ac"
	Nord11 = "#bf616a"
	Nord12 = "#d08770"
	Nord13 = "#ebcb8b"
	Nord14 = "#a3be8c"
	Nord15 = "#b48ead"
)

const (
	ColorText      = Nord5
	ColorMuted     = Nord4
	ColorStrong    = Nord6
	ColorHighlight = Nord8
	ColorError     = Nord11
	ColorMatchFg   = Nord0
	ColorMatchBg   = Nord13
	ColorHeaderFg  = Nord6
	ColorHeaderBg  = Nord2
	ColorBadgeFg   = Nord0
	ColorBadgeBg   = Nord13
	ColorBorder    = Nord3
	ColorActive    = Nord6
	ColorSubtle    = Nord2
	ColorSelectFg  = Nord0
	ColorSelectBg  = Nord8
	ColorHelpDesc  = Nord5
	ColorHelpSep   = Nord3
)

type Styles struct {
	Frame     lipgloss.Style
	Header    lipgloss.Style
	Muted     lipgloss.Style
	Highlight lipgloss.Style
	Match     lipgloss.Style
	Selected  lipgloss.Style
	Badge     lipgloss.Style
	Error     lipgloss.Style
	Panel     lipgloss.Style
	Active    lipgloss.Style
	Subpanel  lipgloss.Style
	Titlebar  lipgloss.Style
	Rule      lipgloss.Style
	Overlay   lipgloss.Style
}

func NewStyles() Styles {
	return Styles{
		Frame: lipgloss.NewStyle(),
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorHeaderFg)).
			Background(lipgloss.Color(ColorHeaderBg)),
		Muted:     lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted)),
		Highlight: lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHighlight)).Bold(true),
		Match: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMatchFg)).
			Background(lipgloss.Color(ColorMatchBg)).
			Bold(true),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorStrong)).
			Bold(true),
		Panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(ColorBorder)),
		Active: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(ColorActive)),
		Subpanel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(ColorSubtle)),
		Titlebar: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorStrong)).
			Background(lipgloss.Color(ColorHeaderBg)),
		Badge: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBadgeFg)).
			Background(lipgloss.Color(ColorBadgeBg)),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorError)).
			Bold(true),
		Rule: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBorder)),
		Overlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorActive)).
			Padding(1, 2),
	}
}

func TableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		Foreground(lipgloss.Color(ColorStrong))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(ColorSelectFg)).
		Background(lipgloss.Color(ColorSelectBg)).
		Bold(false)
	s.Cell = s.Cell.Padding(0, 1)
	return s
}

func ApplyHelpStyles(h *help.Model) {
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHighlight)).Bold(true)
	h.Styles.FullKey = h.Styles.ShortKey
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHelpDesc))
	h.Styles.FullDesc = h.Styles.ShortDesc
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHelpSep))
	h.Styles.FullSeparator = h.Styles.ShortSeparator
}
