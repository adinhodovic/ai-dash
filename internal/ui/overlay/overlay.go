package overlay

import "charm.land/lipgloss/v2"

func Picker(
	page string,
	width, height, listHeight int,
	style lipgloss.Style,
	listView string,
) string {
	overlayW := max(40, width*50/100)
	overlayH := min(listHeight+6, height-4)
	box := style.Width(overlayW).Height(overlayH).Render(listView)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func Sources(
	page string,
	width, height int,
	overlayStyle, headerStyle, mutedStyle lipgloss.Style,
	tableView string,
) string {
	overlayW := max(40, width*70/100)
	overlayH := min(12, height-6)
	title := headerStyle.PaddingLeft(1).PaddingRight(1).MarginBottom(1).Render("Sources")
	hint := mutedStyle.MarginTop(1).Render("Press S or Esc to close")
	body := lipgloss.JoinVertical(lipgloss.Left, title, tableView, hint)
	box := overlayStyle.Width(overlayW).Height(overlayH).Render(body)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func Help(
	page string,
	width, height int,
	overlayStyle, headerStyle, mutedStyle lipgloss.Style,
	helpView string,
) string {
	overlayW := max(30, min(60, width-4))
	overlayH := min(20, height-6)
	title := headerStyle.PaddingLeft(1).
		PaddingRight(1).
		MarginBottom(1).
		Render("Keyboard Shortcuts")
	helpText := lipgloss.NewStyle().MarginBottom(1).Render(helpView)
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		helpText,
		mutedStyle.Render("Press ? to close"),
	)
	box := overlayStyle.Width(overlayW).Height(overlayH).Render(body)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
