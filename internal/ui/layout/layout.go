package layout

// PanePadding is the horizontal inset applied inside every bordered pane, so
// text never sits flush against the border.
const PanePadding = 2

// ContentHeight returns the available height for panes (total minus top bar and footer).
func ContentHeight(termHeight int) int {
	// JoinVertical layout: top(1) + content + footer(1) = termHeight.
	return max(4, termHeight-2)
}

func TopPaneHeight(termHeight int) int {
	return max(4, ContentHeight(termHeight)*28/100)
}

func BottomPaneHeight(termHeight int) int {
	return max(4, ContentHeight(termHeight)-TopPaneHeight(termHeight))
}

func PaneBodyHeight(paneHeight int) int {
	return max(1, paneHeight-4)
}
