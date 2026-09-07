package ui

import (
	"fmt"
	"slices"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/adinhodovic/ai-dash/internal/ui/theme"
	uiutil "github.com/adinhodovic/ai-dash/internal/ui/util"
)

type filterPicker struct {
	active bool
	label  string // "tool", "project", "new-session"
	multi  bool   // toggle-in-place checkboxes vs. pick-one-and-close
	list   list.Model
}

type pickerItem struct {
	value    string
	display  string
	checkbox bool
	checked  bool
}

func (i pickerItem) Title() string {
	if !i.checkbox {
		return i.display
	}
	mark := "[ ]"
	if i.checked {
		mark = "[x]"
	}
	return mark + " " + i.display
}
func (i pickerItem) Description() string { return "" }
func (i pickerItem) FilterValue() string { return i.display }

// newPicker builds a filter picker. In multi mode every option is a
// checkbox toggled in place (no "(all)" entry — clearing is what `c`
// already does); otherwise it's a single pick-and-close list.
func newPicker(
	label string,
	options []string,
	selected []string,
	multi, searchable bool,
) filterPicker {
	selectedSet := make(map[string]bool, len(selected))
	for _, v := range selected {
		selectedSet[v] = true
	}
	if multi {
		options = slices.DeleteFunc(slices.Clone(options), func(s string) bool { return s == "" })
	}
	items := make([]list.Item, 0, len(options))
	cursorIdx := 0
	for i, opt := range options {
		display := opt
		if display == "" {
			display = "(all)"
		} else {
			display = uiutil.CleanProjectName(display)
		}
		if selectedSet[opt] {
			cursorIdx = i
		}
		items = append(
			items,
			pickerItem{value: opt, display: display, checkbox: multi, checked: selectedSet[opt]},
		)
	}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorStrong)).
		Padding(0, 0, 0, 2)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorSelectFg)).
		Background(lipgloss.Color(theme.ColorSelectBg)).
		Padding(0, 0, 0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color(theme.ColorActive))
	delegate.Styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorMuted)).
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
		Foreground(lipgloss.Color(theme.ColorHeaderFg)).
		Background(lipgloss.Color(theme.ColorHeaderBg)).
		Padding(0, 1)
	l.Styles.ActivePaginationDot = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorActive))
	l.Styles.InactivePaginationDot = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorBorder))
	l.Styles.NoItems = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorMuted)).
		Padding(0, 0, 0, 2)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(searchable)
	l.SetShowFilter(searchable)
	l.DisableQuitKeybindings()
	l.Select(cursorIdx)
	return filterPicker{active: true, label: label, multi: multi, list: l}
}

// toggleFilterValue adds or removes value from the tool/project filter set,
// live — the picker stays open so multiple values can be checked in one
// pass instead of one apply-and-reopen cycle per value.
func (m *Model) toggleFilterValue(value, label string) {
	if m.focus != focusList {
		return
	}
	switch label {
	case "tool":
		m.filters.tools = toggleInSlice(m.filters.tools, value)
	case "project":
		m.filters.projects = toggleInSlice(m.filters.projects, value)
	}
	m.sessionCursor = 0
}

func toggleInSlice(values []string, value string) []string {
	if i := slices.Index(values, value); i >= 0 {
		return slices.Delete(slices.Clone(values), i, i+1)
	}
	return append(slices.Clone(values), value)
}

// clearFilter empties the tool/project filter set entirely.
func (m *Model) clearFilter(label string) {
	switch label {
	case "tool":
		m.filters.tools = nil
	case "project":
		m.filters.projects = nil
	}
	m.sessionCursor = 0
	m.statusMessage = fmt.Sprintf("Cleared %s filter", label)
}
