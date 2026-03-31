package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/sahilm/fuzzy"

	"github.com/adin/ai-dash/internal/config"
	"github.com/adin/ai-dash/internal/presets"
	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources"
)

const collapseThreshold = 110

type focusArea int

const (
	focusList focusArea = iota
	focusFilters
	focusSearch
	focusDetail
)

type Options struct {
	Sessions       []session.Session
	Discovery      sources.Discovery
	Config         config.Config
	Err            error
	Version        string
	BuildTimestamp string
}

type filters struct {
	tool    string
	status  string
	project string
}

type Model struct {
	sessions        []session.Session
	width           int
	height          int
	err             error
	styles          styles
	meta            Options
	filters         filters
	focus           focusArea
	detailCollapsed bool
	autoCollapsed   bool
	manualCollapse  bool
	statusMessage   string
	searchInput     textinput.Model
	sessionTable    table.Model
	sourceTable     table.Model
	relatedTable    table.Model
	detailTable     table.Model
	overviewTable   table.Model
	help            help.Model
	keys            keyMap
	presetStore     presets.Store
	sortField       session.SortField
	sortDescending  bool
	projSortField   string
	projSortDesc    bool
	projectPaths    []string // raw project paths matching overviewTable rows
	showHelp        bool
	showSources     bool
	showSubagents   bool
	picker          filterPicker
}

func NewModel(opts Options) Model {
	input := textinput.New()
	input.Placeholder = "search sessions"
	input.CharLimit = 120
	input.Prompt = ""
	input.Blur()

	reloadInterval = opts.Config.PollDuration()
	maxSessionAge = opts.Config.DefaultAgeFilterDuration()
	if presets := opts.Config.AgeDurations(); len(presets) > 0 {
		agePresets = presets
	}
	terminalCmd = opts.Config.Terminal

	store, err := presets.Load()
	m := Model{
		sessions:     opts.Sessions,
		err:          opts.Err,
		styles:       newStyles(),
		meta:         opts,
		searchInput:  input,
		sessionTable: newSessionTable(),
		sourceTable:  newSourceTable(),
		relatedTable: newRelatedTable(),
		detailTable:  newTable([]table.Column{{Title: "", Width: 10}, {Title: "", Width: 30}}),
		overviewTable: newTable(
			[]table.Column{{Title: "Metric", Width: 16}, {Title: "Value", Width: 20}},
		),
		help: func() help.Model {
			h := help.New()
			applyHelpStyles(&h)
			return h
		}(),
		keys:           defaultKeyMap(),
		presetStore:    store,
		sortField:      session.SortUpdated,
		sortDescending: true,
		projSortField:  "last",
		projSortDesc:   true,
	}
	if err != nil {
		m.statusMessage = fmt.Sprintf("preset load error: %v", err)
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickReload())
}

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
		case "o":
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
		case "f":
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
		case "w":
			m.savePreset(filtered)
		case "r":
			m.restorePreset(filtered)
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
			sessions, err := sources.LoadDefaultSessions(discovery)
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

func (m *Model) updateDetailCollapse() {
	m.detailCollapsed = m.autoCollapsed || m.manualCollapse
}

// cycleForward/cycleBackward skip focusSearch — search is only entered via '/'.
var tabbableFoci = []focusArea{focusList, focusFilters}

func (m *Model) cycleForward() {
	for i, f := range tabbableFoci {
		if f == m.focus {
			m.focus = tabbableFoci[(i+1)%len(tabbableFoci)]
			return
		}
	}
	m.focus = focusList
}

func (m *Model) cycleBackward() {
	for i, f := range tabbableFoci {
		if f == m.focus {
			m.focus = tabbableFoci[(i-1+len(tabbableFoci))%len(tabbableFoci)]
			return
		}
	}
	m.focus = focusList
}

func (m Model) filteredSessions() []session.Session {
	filtered := make([]session.Session, 0, len(m.sessions))
	query := strings.ToLower(strings.TrimSpace(m.searchQuery()))
	cutoff := time.Now().Add(-maxSessionAge)
	for _, s := range m.sessions {
		if lastActive(s).Before(cutoff) {
			continue
		}
		if m.filters.tool != "" && s.Tool != m.filters.tool {
			continue
		}
		if m.filters.status != "" && s.Status != m.filters.status {
			continue
		}
		if m.filters.project != "" && s.Project != m.filters.project {
			continue
		}
		if !m.showSubagents && s.ParentID != "" {
			continue
		}
		if query != "" && !matchesQuery(s, query) {
			continue
		}
		filtered = append(filtered, s)
	}
	session.SortBy(filtered, m.sortField, m.sortDescending)
	return filtered
}

func matchesQuery(s session.Session, query string) bool {
	fields := []string{
		s.Tool,
		s.Project,
		s.Repo,
		s.Branch,
		s.Status,
		s.Model,
		s.Summary,
		strings.Join(s.Tags, " "),
	}
	// Try exact substring first (fast path).
	lower := strings.ToLower(strings.Join(fields, " "))
	if strings.Contains(lower, query) {
		return true
	}
	// Fall back to fuzzy, but require a decent score.
	matches := fuzzy.Find(query, []string{lower})
	return len(matches) > 0 && matches[0].Score > len(query)*2
}

func (m Model) currentProject(filtered []session.Session) string {
	sel := m.sessionTable.Cursor()
	if len(filtered) > 0 && sel < len(filtered) {
		return filtered[sel].Project
	}
	if m.filters.project != "" {
		return m.filters.project
	}
	return ""
}

func (m Model) searchQuery() string { return m.searchInput.Value() }

var maxSessionAge time.Duration

var agePresets = []time.Duration{
	time.Hour,
	24 * time.Hour,
	3 * 24 * time.Hour,
	7 * 24 * time.Hour,
	14 * 24 * time.Hour,
	30 * 24 * time.Hour,
}

func ageLabel(d time.Duration) string {
	hours := int(d.Hours())
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dd", hours/24)
}

type statusMsg struct{ message string }

type reloadMsg struct {
	sessions  []session.Session
	discovery sources.Discovery
	err       error
}

var reloadInterval time.Duration

func tickReload() tea.Cmd {
	return tea.Tick(reloadInterval, func(_ time.Time) tea.Msg {
		return triggerReloadMsg{}
	})
}

type triggerReloadMsg struct{}
