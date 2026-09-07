package theme

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/adinhodovic/ai-dash/internal/session"
	uilayout "github.com/adinhodovic/ai-dash/internal/ui/layout"
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
	ColorWarn      = Nord13
	ColorSuccess   = Nord14
	// ColorInfo is deliberately Nord9, not Nord8 (ColorHighlight) — keeping
	// distinct semantic colors from purely decorative ones (see ColorSelectBg)
	// avoids accidental foreground==background collisions when a row is
	// selected.
	ColorInfo     = Nord9
	ColorAccent   = Nord15
	ColorLimit    = Nord12
	ColorMatchFg  = Nord0
	ColorMatchBg  = Nord13
	ColorHeaderFg = Nord6
	ColorHeaderBg = Nord2
	ColorBadgeFg  = Nord0
	ColorBadgeBg  = Nord13
	ColorBorder   = Nord3
	ColorActive   = Nord6
	ColorSubtle   = Nord2
	// ColorSelectBg is a subtle lift off the base background (Nord's own
	// "selection" shade), not an inverted bright block — the rest of the
	// palette assumes light text on a dark background, and a bright color
	// like Nord8 there makes every existing foreground color low-contrast
	// (light-on-light) the moment something is selected.
	ColorSelectFg = Nord6
	ColorSelectBg = Nord3
	ColorHelpDesc = Nord5
	ColorHelpSep  = Nord3
)

type Styles struct {
	Frame        lipgloss.Style
	Header       lipgloss.Style
	Muted        lipgloss.Style
	Highlight    lipgloss.Style
	Match        lipgloss.Style
	Selected     lipgloss.Style
	Badge        lipgloss.Style
	FilterActive lipgloss.Style
	Error        lipgloss.Style
	Panel        lipgloss.Style
	Active       lipgloss.Style
	Subpanel     lipgloss.Style
	Titlebar     lipgloss.Style
	Rule         lipgloss.Style
	Overlay      lipgloss.Style
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
		// FilterActive marks "this filter is applied" — deliberately not
		// Badge's warning-toned Nord13 (that's for search-match highlights
		// and literal warnings), but the same accent color used everywhere
		// else in the UI for "this is the active/selected thing".
		FilterActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorMatchFg)).
			Background(lipgloss.Color(ColorHighlight)),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorError)).
			Bold(true),
		Rule: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBorder)),
		Overlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorActive)).
			Padding(uilayout.PanePadding, uilayout.PanePadding),
	}
}

func TableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		Foreground(lipgloss.Color(ColorStrong)).
		PaddingBottom(1)
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

// StatusStyle colors a session's plain CurrentState label. Bold is
// intentionally reserved for attention rows (see AttentionStyle/ReasonStyle),
// so none of these cases use it — that's what makes a flagged row visually
// distinct from a normal one, not just differently colored.
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case string(session.StateAborted):
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted))
	case string(session.StateToolCall):
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorAccent))
	case string(session.StateWaiting):
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarn))
	case string(session.StateRunning):
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorInfo))
	case string(session.StateMaxTokens):
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLimit))
	case string(session.StateDone):
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText))
	}
}

// StateGlyph returns the icon for a session's plain CurrentState label,
// mirroring StatusStyle's cases one-for-one so the icon and its color can
// never drift apart from each other.
func StateGlyph(status string) string {
	switch status {
	case string(session.StateAborted):
		return Error
	case string(session.StateToolCall):
		return Tool
	case string(session.StateWaiting):
		return Clock
	case string(session.StateRunning):
		return Active
	case string(session.StateMaxTokens):
		return Token
	case string(session.StateDone):
		return Success
	default:
		return Inactive
	}
}

// AttentionStyle is the "!" icon's style — reserved exclusively as the one
// consistent alert color across the whole UI, regardless of the underlying
// reason (that's ReasonStyle's job).
func AttentionStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError)).Bold(true)
}

// ReasonStyle colors the attention label text (e.g. "waiting 45m") by why a
// session was flagged, so different reasons stay visually distinguishable
// even though they share the same AttentionStyle icon.
func ReasonStyle(reason session.AttentionReason) lipgloss.Style {
	switch reason {
	case session.AttentionMaxTokens:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLimit)).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarn)).Bold(true)
	}
}
