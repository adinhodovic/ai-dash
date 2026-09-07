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
				item, ok := m.picker.list.SelectedItem().(pickerItem)
				if !ok {
					break
				}
				if m.picker.multi {
					// Toggle in place and keep the picker open — checking
					// several values shouldn't mean reopening it each time.
					m.toggleFilterValue(item.value, m.picker.label)
					item.checked = !item.checked
					m.picker.list.SetItem(m.picker.list.Index(), item)
					filtered = m.filteredSessions()
					m.syncAllTables(filtered)
					return m, cmd
				}
				if m.picker.label == "new-session" {
					if cmd := m.openNewSession(item.value); cmd != nil {
						m.picker.active = false
						return m, cmd
					}
				}
				m.picker.active = false
				filtered = m.filteredSessions()
				m.syncAllTables(filtered)
			case "esc":
				m.picker.active = false
			case "c":
				if m.picker.multi {
					m.clearFilter(m.picker.label)
					m.picker.active = false
					filtered = m.filteredSessions()
					m.syncAllTables(filtered)
				}
			}
			return m, cmd
		}
		if m.renaming {
			return m.updateRename(msg)
		}
		if m.focus == focusSearch {
			return m.updateSearch(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = true
		case "/":
			m.focus = focusSearch
			m.searchQueryBeforeEdit = m.searchInput.Value()
			m.searchInput.Focus()
		case "r":
			if cmd := m.openSelectedExternally(filtered); cmd != nil {
				return m, cmd
			}
		case "R":
			if m.focus == focusList {
				m.startRename(filtered)
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
				var selected []string
				if defTool != "" {
					selected = []string{defTool}
				}
				m.picker = newPicker(
					"new session (tool)",
					toolOptions(m.sessions),
					selected,
					false,
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
			m.resizeSourceTable()
		case "i":
			m.showDetailExtra = !m.showDetailExtra
			m.resizeDetailTable(filtered)
			if m.showDetailExtra {
				m.statusMessage = "Showing IDs & metadata"
			} else {
				m.statusMessage = "Hiding IDs & metadata"
			}

		case "c":
			m.filters = filters{}
			m.searchInput.SetValue("")
			m.showSubagents = false
			m.showAttentionOnly = false
			m.showActiveOnly = false
			maxSessionAge = m.meta.Config.DefaultAgeFilterDuration()
			m.sessionCursor = 0
			m.statusMessage = "Cleared all filters"
			filtered = m.filteredSessions()
			m.syncAllTables(filtered)
		case "t":
			m.picker.active = false
			m.showSources = false
			m.showHelp = false
			m.picker = newPicker("tool", toolOptions(filtered), m.filters.tools, true, false)
		case "s":
			m.cycleSortForward()
			filtered = m.filteredSessions()
			m.syncAllTables(filtered)
		case "p":
			m.picker.active = false
			m.showSources = false
			m.showHelp = false
			m.picker = newPicker("project", projectOptions(filtered), m.filters.projects, true, true)
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
			m.sessionCursor = 0
			m.syncAllTables(filtered)
		case "a":
			m.showSubagents = !m.showSubagents
			filtered = m.filteredSessions()
			m.sessionCursor = 0
			m.syncAllTables(filtered)
			if m.showSubagents {
				m.statusMessage = "Showing subagent sessions"
			} else {
				m.statusMessage = "Hiding subagent sessions"
			}
		case "!":
			m.showAttentionOnly = !m.showAttentionOnly
			if m.showAttentionOnly {
				for key := range attentionKeys(m.sessions) {
					m.seenAttention[key] = true
				}
			}
			filtered = m.filteredSessions()
			m.sessionCursor = 0
			m.syncAllTables(filtered)
			if m.showAttentionOnly {
				m.statusMessage = "Showing only sessions needing attention"
			} else {
				m.statusMessage = "Showing all sessions"
			}
		case "A":
			m.showActiveOnly = !m.showActiveOnly
			filtered = m.filteredSessions()
			m.sessionCursor = 0
			m.syncAllTables(filtered)
			if m.showActiveOnly {
				m.statusMessage = "Showing only active sessions"
			} else {
				m.statusMessage = "Showing all sessions"
			}
		case "up", "k":
			if m.focus == focusList {
				moveSessionCursor(&m, -1, filtered)
			}
		case "down", "j":
			if m.focus == focusList {
				moveSessionCursor(&m, 1, filtered)
			}
		case "pgup", "ctrl+u":
			if m.focus == focusList {
				page := max(1, m.sessionViewport.Height()/sessionRowStride)
				moveSessionCursor(&m, -page, filtered)
			}
		case "pgdown", "ctrl+d":
			if m.focus == focusList {
				page := max(1, m.sessionViewport.Height()/sessionRowStride)
				moveSessionCursor(&m, page, filtered)
			}
		case "g":
			if m.focus == focusList {
				moveSessionCursor(&m, -len(filtered), filtered)
			}
		case "G":
			if m.focus == focusList {
				moveSessionCursor(&m, len(filtered), filtered)
			}
		}
	case statusMsg:
		m.statusMessage = msg.message
	case triggerReloadMsg:
		return m, tea.Batch(tickReload(), func() tea.Msg {
			discovery, _ := sources.Discover(m.meta.Config)
			sessions := append([]session.Session(nil), discovery.Sessions...)
			session.ApplyRenames(sessions, m.meta.Renames)
			session.Sort(sessions)
			var err error
			return reloadMsg{sessions: sessions, discovery: discovery, err: err}
		})
	case reloadMsg:
		if msg.err == nil && len(msg.sessions) > 0 {
			prev := len(m.sessions)
			m.sessions = msg.sessions
			m.meta.Discovery = msg.discovery
			current := attentionKeys(m.sessions)
			for key := range current {
				current[key] = m.seenAttention[key]
			}
			m.seenAttention = current
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
	m.resizeSourceTable()
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
		m.searchInput.SetValue(m.searchQueryBeforeEdit)
		m.searchInput.Blur()
		m.syncAllTables(m.filteredSessions())
		m.statusMessage = "Search cancelled"
	case "enter":
		m.focus = focusList
		m.searchInput.Blur()
		m.sessionCursor = 0
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
	}
	return m, cmd
}

func (m *Model) startRename(filtered []session.Session) {
	sel := m.sessionCursor
	if len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		m.statusMessage = "No session selected"
		return
	}
	s := filtered[sel]
	m.renaming = true
	m.renamingKey = session.RenameKey(s)
	m.renameInput.SetValue(s.Summary)
	m.renameInput.Focus()
	m.showHelp = false
	m.showSources = false
	m.picker.active = false
}

func (m Model) updateRename(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.renaming = false
		m.renamingKey = ""
		m.renameInput.Blur()
		m.statusMessage = "Rename cancelled"
	case "enter":
		newSummary := strings.TrimSpace(m.renameInput.Value())
		if newSummary == "" {
			m.statusMessage = "Rename cannot be empty"
			return m, nil
		}
		if m.meta.Renames == nil {
			m.meta.Renames = map[string]string{}
		}
		previous, hadPrevious := m.meta.Renames[m.renamingKey]
		m.meta.Renames[m.renamingKey] = newSummary
		if err := session.SaveRenames(m.meta.RenamesPath, m.meta.Renames); err != nil {
			if hadPrevious {
				m.meta.Renames[m.renamingKey] = previous
			} else {
				delete(m.meta.Renames, m.renamingKey)
			}
			m.statusMessage = fmt.Sprintf("Rename not saved: %v", err)
			return m, nil
		}
		for i := range m.sessions {
			if session.RenameKey(m.sessions[i]) == m.renamingKey {
				m.sessions[i].Summary = newSummary
				break
			}
		}
		m.statusMessage = "Renamed session"
		m.renaming = false
		m.renamingKey = ""
		m.renameInput.Blur()
		m.syncAllTables(m.filteredSessions())
	default:
		m.renameInput, cmd = m.renameInput.Update(msg)
	}
	return m, cmd
}
