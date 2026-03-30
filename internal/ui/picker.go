package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"

	"github.com/adin/ai-dash/internal/presets"
	"github.com/adin/ai-dash/internal/session"
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
	height := min(len(items)+4, 20)
	l := list.New(items, list.NewDefaultDelegate(), 40, height)
	l.Title = fmt.Sprintf("Filter: %s", label)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(searchable)
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
	m.selected = 0
	m.statusMessage = fmt.Sprintf("Updated %s filter", label)
}

func (m *Model) savePreset(filtered []session.Session) {
	project := m.currentProject(filtered)
	if project == "" {
		return
	}
	m.presetStore.Projects[project] = presets.Preset{
		Tool:    m.filters.tool,
		Status:  m.filters.status,
		Project: m.filters.project,
		Search:  strings.TrimSpace(m.searchQuery()),
	}
	if err := presets.Save(m.presetStore); err != nil {
		m.statusMessage = fmt.Sprintf("preset save error: %v", err)
		return
	}
	m.statusMessage = fmt.Sprintf("Saved preset for %s", project)
}

func (m *Model) restorePreset(filtered []session.Session) {
	project := m.currentProject(filtered)
	if project == "" {
		return
	}
	preset, ok := m.presetStore.Projects[project]
	if !ok {
		m.statusMessage = fmt.Sprintf("No saved preset for %s", project)
		return
	}
	m.filters = filters{tool: preset.Tool, status: preset.Status, project: preset.Project}
	m.searchInput.SetValue(preset.Search)
	m.selected = 0
	m.statusMessage = fmt.Sprintf("Restored preset for %s", project)
}
