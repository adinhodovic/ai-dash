package overlay

import "charm.land/lipgloss/v2"

func Picker(
	width, height, listHeight int,
	style, mutedStyle lipgloss.Style,
	listView, hint string,
) string {
	overlayW := max(40, width*50/100)
	body := listView
	if hint != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, listView, mutedStyle.MarginTop(1).Render(hint))
	}
	overlayH := min(listHeight+8, height-4)
	box := style.Width(overlayW).Height(overlayH).Render(body)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func Sources(
	width, height int,
	overlayStyle, headerStyle, mutedStyle lipgloss.Style,
	tableView string,
) string {
	overlayW := max(40, width*70/100)
	overlayH := min(12, height-6)
	title := headerStyle.MarginBottom(1).Render("Sources")
	hint := mutedStyle.MarginTop(1).Render("Press S or Esc to close")
	body := lipgloss.JoinVertical(lipgloss.Left, title, tableView, hint)
	box := overlayStyle.Width(overlayW).Height(overlayH).Render(body)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func Help(
	width, height int,
	overlayStyle, headerStyle, mutedStyle lipgloss.Style,
	helpView string,
) string {
	title := headerStyle.MarginBottom(1).Render("Keyboard Shortcuts")
	helpText := lipgloss.NewStyle().MarginBottom(1).Render(helpView)
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		helpText,
		mutedStyle.Render("Press ? to close"),
	)
	// Grow with the content, since configured actions add a column.
	overlayW := lipgloss.Width(body) + overlayStyle.GetHorizontalFrameSize()
	overlayH := lipgloss.Height(body) + overlayStyle.GetVerticalFrameSize()
	overlayW = max(30, min(max(60, overlayW), width-4))
	overlayH = max(4, min(max(20, overlayH), height-2))
	box := overlayStyle.Width(overlayW).Height(overlayH).MaxHeight(overlayH).Render(body)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
