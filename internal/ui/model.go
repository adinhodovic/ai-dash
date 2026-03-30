package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

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
	selected        int
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
	showHelp        bool
	showSources     bool
	picker          filterPicker
}

func NewModel(opts Options) Model {
	input := textinput.New()
	input.Placeholder = "search sessions"
	input.CharLimit = 120
	input.Prompt = "search> "
	input.Blur()

	reloadInterval = opts.Config.PollDuration()
	maxSessionAge = opts.Config.MaxAgeDuration()
	terminalCmd = opts.Config.Terminal

	store, err := presets.Load()
	m := Model{
		sessions:      opts.Sessions,
		err:           opts.Err,
		styles:        newStyles(),
		meta:          opts,
		searchInput:   input,
		sessionTable:  newSessionTable(),
		sourceTable:   newSourceTable(),
		relatedTable:  newRelatedTable(),
		detailTable:   newHeaderlessTable([]table.Column{{Title: "", Width: 10}, {Title: "", Width: 30}}),
		overviewTable: newTable([]table.Column{{Title: "Metric", Width: 16}, {Title: "Value", Width: 20}}),
		help: func() help.Model {
			h := help.New()
			applyHelpStyles(&h)
			return h
		}(),
		keys:           defaultKeyMap(),
		presetStore:    store,
		sortField:      session.SortUpdated,
		sortDescending: true,
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
	m.clampSelection(len(filtered))

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
			switch msg.String() {
			case "enter":
				if item, ok := m.picker.list.SelectedItem().(pickerItem); ok {
					m.applyFilterChange(item.value, m.picker.label)
				}
				m.picker.active = false
				filtered = m.filteredSessions()
				m.syncAllTables(filtered)
			case "esc":
				m.picker.active = false
			default:
				var cmd tea.Cmd
				m.picker.list, cmd = m.picker.list.Update(msg)
				return m, cmd
			}
			return m, nil
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
		case "enter", "o":
			if cmd := m.openSelectedExternally(filtered); cmd != nil {
				return m, cmd
			}
		case "]":
			m.sortField = nextSortField(m.sortField)
			m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
		case "[":
			m.sortField = prevSortField(m.sortField)
			m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
		case "=":
			m.sortDescending = !m.sortDescending
			m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
		case "v":
			m.manualCollapse = !m.manualCollapse
			m.updateDetailCollapse()
			m.resizeRightTables(filtered)

		case "c":
			m.filters = filters{}
			m.searchInput.SetValue("")
			m.selected = 0
			m.statusMessage = "Cleared filters and search"
			filtered = m.filteredSessions()
			m.syncAllTables(filtered)
		case "f":
			m.picker.active = false
			m.showSources = false
			m.showHelp = false
			m.picker = newPicker("tool", toolOptions(m.sessions), m.filters.tool, false)
		case "s":
			m.picker.active = false
			m.showSources = false
			m.showHelp = false
			m.picker = newPicker("status", statusOptions(m.sessions), m.filters.status, false)
		case "p":
			m.picker.active = false
			m.showSources = false
			m.showHelp = false
			m.picker = newPicker("project", projectOptions(m.sessions), m.filters.project, true)
		case "S":
			m.picker.active = false
			m.showHelp = false
			m.showSources = !m.showSources
		case "w":
			m.savePreset(filtered)
		case "r":
			m.restorePreset(filtered)
		default:
			if m.focus == focusList {
				var cmd tea.Cmd
				m.sessionTable, cmd = m.sessionTable.Update(msg)
				m.selected = m.sessionTable.Cursor()
				// Only sync dependent views, don't touch the table itself
				m.resizeDetailTable(filtered)
				m.resizeRightTables(filtered)
				return m, cmd
			}
			if m.focus == focusDetail {
				var cmd tea.Cmd
				m.detailTable, cmd = m.detailTable.Update(msg)
				return m, cmd
			}
			if m.focus == focusFilters {
				var cmd tea.Cmd
				m.relatedTable, cmd = m.relatedTable.Update(msg)
				if m.jumpToRelated(filtered) {
					filtered = m.filteredSessions()
				}
				m.syncAfterChange(filtered)
				return m, cmd
			}
		}
	case statusMsg:
		m.statusMessage = msg.message
	case triggerReloadMsg:
		return m, tea.Batch(tickReload(), func() tea.Msg {
			discovery, _ := sources.Discover()
			sessions, err := sources.LoadDefaultSessions(discovery)
			return reloadMsg{sessions: sessions, discovery: discovery, err: err}
		})
	case reloadMsg:
		if msg.err == nil && len(msg.sessions) > 0 {
			prev := len(m.sessions)
			m.sessions = msg.sessions
			m.meta.Discovery = msg.discovery
			if len(m.sessions) != prev {
				m.statusMessage = fmt.Sprintf("Reloaded: %d sessions (was %d)", len(m.sessions), prev)
			}
			filtered = m.filteredSessions()
			m.syncAllTables(filtered)
		}
	}

	m.syncAfterChange(filtered)
	return m, nil
}

func (m *Model) syncAfterChange(filtered []session.Session) {
	m.resizeDetailTable(filtered)
	m.resizeOverviewTable(filtered)
	m.resizeRightTables(filtered)
}

func (m *Model) syncAllTables(filtered []session.Session) {
	m.syncTable(filtered)
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
	case "enter":
		m.focus = focusList
		m.searchInput.Blur()
		m.selected = 0
		if strings.TrimSpace(m.searchQuery()) == "" {
			m.statusMessage = "Cleared search"
		} else {
			m.statusMessage = fmt.Sprintf("Applied search: %s", strings.TrimSpace(m.searchQuery()))
		}
	default:
		m.searchInput, cmd = m.searchInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) updateDetailCollapse() {
	m.detailCollapsed = m.autoCollapsed || m.manualCollapse
}

// cycleForward/cycleBackward skip focusSearch — search is only entered via '/'.
var tabbableFoci = []focusArea{focusList, focusDetail, focusFilters}

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

func (m *Model) clampSelection(length int) {
	if length == 0 {
		m.selected = 0
		return
	}
	if m.selected >= length {
		m.selected = length - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
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
		if query != "" && !matchesQuery(s, query) {
			continue
		}
		filtered = append(filtered, s)
	}
	session.SortBy(filtered, m.sortField, m.sortDescending)
	return filtered
}

func matchesQuery(s session.Session, query string) bool {
	fields := []string{s.ID, s.Tool, s.Project, s.Repo, s.Branch, s.Status, s.Model, s.Summary, strings.Join(s.Tags, " ")}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func (m Model) currentProject(filtered []session.Session) string {
	if len(filtered) > 0 && m.selected < len(filtered) {
		return filtered[m.selected].Project
	}
	if m.filters.project != "" {
		return m.filters.project
	}
	return ""
}

func (m Model) searchQuery() string { return m.searchInput.Value() }

var maxSessionAge time.Duration

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
