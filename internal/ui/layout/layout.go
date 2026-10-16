package layout

// PanePadding is the horizontal inset applied inside every bordered pane, so
// text never sits flush against the border.
const PanePadding = 2

// ContentHeight returns the available height for panes (total minus the top
// area and the footer). The top area is: TopStrip (one line, no border) + a
// margin line + TopBar (border(2), Search and Filters sharing one line, so
// 3 total) = 1 + 1 + 3 = 5, plus the footer(1) = 6.
func ContentHeight(termHeight int) int {
	return max(4, termHeight-6)
}

// BottomPaneHeight is the height available to the Sessions/Details row.
// There's no pane above it — that space is now the full content height.
func BottomPaneHeight(termHeight int) int {
	return ContentHeight(termHeight)
}

// PaneBodyHeight returns the usable height inside a bordered pane: just the
// border (2) — panes no longer carry a title line, so there's nothing else
// to subtract.
func PaneBodyHeight(paneHeight int) int {
	return max(1, paneHeight-2)
}
