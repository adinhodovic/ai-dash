package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	filtered := m.filteredSessions()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.autoCollapsed = msg.Width < collapseThreshold
		m.updateDetailCollapse()
		m.resizeTable(filtered)
		m.resizeSourceTable()
		m.syncAllTables(filtered)

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q":
				m.showHelp = false
			}
			return m, nil
		}
		if m.showSources {
			switch msg.String() {
			case "S", "esc", "q":
				m.showSources = false
			}
			return m, nil
		}
		if m.picker.active {
			var cmd tea.Cmd
			prevState := m.picker.list.FilterState()
			m.picker.list, cmd = m.picker.list.Update(msg)

			// Actively typing in the filter — let the list handle everything.
			if m.picker.list.FilterState() == list.Filtering || prevState == list.Filtering {
				return m, cmd
			}

			switch msg.String() {
			case "enter":
				if item, ok := m.picker.list.SelectedItem().(pickerItem); ok {
					if m.picker.label == "new-session" {
						if cmd := m.openNewSession(item.value); cmd != nil {
							m.picker.active = false
							return m, cmd
						}
					} else {
						m.applyFilterChange(item.value, m.picker.label)
					}
				}
				m.picker.active = false
				filtered = m.filteredSessions()
				m.syncAllTables(filtered)
			case "esc":
				m.picker.active = false
			}
			return m, cmd
		}
		if m.focus == focusSearch {
			return m.updateSearch(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = true
		case "tab":
			m.cycleForward()
		case "shift+tab":
			m.cycleBackward()
		case "/":
			m.focus = focusSearch
			m.searchInput.Focus()
		case "r":
			if cmd := m.openSelectedExternally(filtered); cmd != nil {
				return m, cmd
			}
		case "n":
			if m.meta.Config.AutoSelectTool && m.meta.Config.DefaultTool != "" {
				if cmd := m.openNewSession(m.meta.Config.DefaultTool); cmd != nil {
					return m, cmd
				}
			} else {
				m.picker.active = false
				m.showSources = false
				m.showHelp = false
				defTool := m.meta.Config.DefaultTool
				m.picker = newPicker(
					"new session (tool)",
					toolOptions(m.sessions),
					defTool,
					false,
				)
				m.picker.label = "new-session"
			}
		case "]":
			m.cycleSortForward()
			filtered = m.filteredSessions()
			m.syncAllTables(filtered)
		case "[":
			m.cycleSortBackward()
			filtered = m.filteredSessions()
			m.syncAllTables(filtered)
		case "=":
			m.toggleSortDirection()
			filtered = m.filteredSessions()
			m.syncAllTables(filtered)
		case "v":
			m.manualCollapse = !m.manualCollapse
			m.updateDetailCollapse()
			m.resizeRightTables(filtered)

		case "c":
			m.filters = filters{}
			m.searchInput.SetValue("")
			m.showSubagents = false
			maxSessionAge = m.meta.Config.DefaultAgeFilterDuration()
			m.sessionTable.SetCursor(0)
			m.statusMessage = "Cleared all filters"
			filtered = m.filteredSessions()
			m.syncAllTables(filtered)
		case "t":
			m.picker.active = false
			m.showSources = false
			m.showHelp = false
			m.picker = newPicker("tool", toolOptions(filtered), m.filters.tool, false)
		case "s":
			m.cycleSortForward()
			filtered = m.filteredSessions()
			m.syncAllTables(filtered)
		case "p":
			m.picker.active = false
			m.showSources = false
			m.showHelp = false
			m.picker = newPicker("project", projectOptions(filtered), m.filters.project, true)
		case "S":
			m.picker.active = false
			m.showHelp = false
			m.showSources = !m.showSources
		case "D":
			// Cycle through age presets
			next := agePresets[0]
			for i, preset := range agePresets {
				if preset == maxSessionAge && i+1 < len(agePresets) {
					next = agePresets[i+1]
					break
				}
				if preset == maxSessionAge && i+1 >= len(agePresets) {
					next = agePresets[0]
					break
				}
			}
			maxSessionAge = next
			m.statusMessage = fmt.Sprintf("Showing sessions from last %s", ageLabel(maxSessionAge))
			filtered = m.filteredSessions()
			m.sessionTable.SetCursor(0)
			m.syncAllTables(filtered)
		case "a":
			m.showSubagents = !m.showSubagents
			filtered = m.filteredSessions()
			m.sessionTable.SetCursor(0)
			m.syncAllTables(filtered)
			if m.showSubagents {
				m.statusMessage = "Showing subagent sessions"
			} else {
				m.statusMessage = "Hiding subagent sessions"
			}
		default:
			if m.focus == focusList {
				var cmd tea.Cmd
				m.sessionTable, cmd = m.sessionTable.Update(msg)
				m.resizeDetailTable(filtered)
				m.resizeRightTables(filtered)
				return m, cmd
			}
			if m.focus == focusFilters {
				var cmd tea.Cmd
				m.overviewTable, cmd = m.overviewTable.Update(msg)
				return m, cmd
			}
		}
	case statusMsg:
		m.statusMessage = msg.message
	case triggerReloadMsg:
		return m, tea.Batch(tickReload(), func() tea.Msg {
			discovery, _ := sources.Discover(m.meta.Config)
			sessions := append([]session.Session(nil), discovery.Sessions...)
			session.Sort(sessions)
			var err error
			return reloadMsg{sessions: sessions, discovery: discovery, err: err}
		})
	case reloadMsg:
		if msg.err == nil && len(msg.sessions) > 0 {
			prev := len(m.sessions)
			m.sessions = msg.sessions
			m.meta.Discovery = msg.discovery
			if len(m.sessions) != prev {
				m.statusMessage = fmt.Sprintf(
					"Reloaded: %d sessions (was %d)", len(m.sessions), prev,
				)
			}
			filtered = m.filteredSessions()
			m.syncAllTables(filtered)
		}
	}

	// Forward non-key messages to the picker list (e.g. FilterMatchesMsg).
	if m.picker.active {
		var cmd tea.Cmd
		m.picker.list, cmd = m.picker.list.Update(msg)
		return m, cmd
	}

	if m.focus != focusSearch {
		m.syncAfterChange(filtered)
	}
	return m, nil
}

func (m *Model) syncAfterChange(filtered []session.Session) {
	m.resizeDetailTable(filtered)
	m.resizeOverviewTable(filtered)
	m.resizeRightTables(filtered)
}

func (m *Model) syncAllTables(filtered []session.Session) {
	m.resizeTable(filtered)
	m.syncAfterChange(filtered)
}

func (m Model) updateSearch(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.focus = focusList
		m.searchInput.Blur()
		m.syncAllTables(m.filteredSessions())
	case "enter":
		m.focus = focusList
		m.searchInput.Blur()
		m.sessionTable.SetCursor(0)
		filtered := m.filteredSessions()
		m.syncAllTables(filtered)
		if strings.TrimSpace(m.searchQuery()) == "" {
			m.statusMessage = "Cleared search"
		} else {
			m.statusMessage = fmt.Sprintf("Applied search: %s", strings.TrimSpace(m.searchQuery()))
		}
	default:
		m.searchInput, cmd = m.searchInput.Update(msg)
		filtered := m.filteredSessions()
		m.resizeTable(filtered)
		m.resizeOverviewTable(filtered)
	}
	return m, cmd
}
