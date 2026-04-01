package ui

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
)

type filterPicker struct {
	active bool
	label  string // "tool", "status", "project"
	list   list.Model
}

type pickerItem struct {
	value   string
	display string
}

func (i pickerItem) Title() string       { return i.display }
func (i pickerItem) Description() string { return "" }
func (i pickerItem) FilterValue() string { return i.display }

func newPicker(label string, options []string, current string, searchable bool) filterPicker {
	items := make([]list.Item, 0, len(options))
	selectedIdx := 0
	for i, opt := range options {
		display := opt
		if display == "" {
			display = "(all)"
		} else {
			display = cleanProjectName(display)
		}
		if opt == current {
			selectedIdx = i
		}
		items = append(items, pickerItem{value: opt, display: display})
	}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorStrong)).
		Padding(0, 0, 0, 2)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorSelectFg)).
		Background(lipgloss.Color(colorSelectBg)).
		Padding(0, 0, 0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color(colorActive))
	delegate.Styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted)).
		Padding(0, 0, 0, 2)

	maxH := 20
	if searchable {
		maxH = 30
	}
	height := max(10, min(len(items)+6, maxH))
	l := list.New(items, delegate, 50, height)
	l.Title = fmt.Sprintf("Filter: %s", label)
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 2)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorHeaderFg)).
		Background(lipgloss.Color(colorHeaderBg)).
		Padding(0, 1)
	l.Styles.ActivePaginationDot = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorActive))
	l.Styles.InactivePaginationDot = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorBorder))
	l.Styles.NoItems = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted)).
		Padding(0, 0, 0, 2)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(searchable)
	l.SetShowFilter(searchable)
	l.DisableQuitKeybindings()
	l.Select(selectedIdx)
	return filterPicker{active: true, label: label, list: l}
}

func (m *Model) applyFilterChange(value, label string) {
	if m.focus != focusFilters && m.focus != focusList {
		return
	}
	switch label {
	case "tool":
		m.filters.tool = value
	case "status":
		m.filters.status = value
	case "project":
		m.filters.project = value
	}
	m.sessionTable.SetCursor(0)
	m.statusMessage = fmt.Sprintf("Updated %s filter", label)
}
