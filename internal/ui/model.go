package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/sahilm/fuzzy"

	"github.com/adin/ai-dash/internal/config"
	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources"
	"github.com/adin/ai-dash/internal/ui/theme"
	uiutil "github.com/adin/ai-dash/internal/ui/util"
)

const collapseThreshold = 110

type focusArea int

const (
	focusList focusArea = iota
	focusFilters
	focusSearch
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
	project string
}

type Model struct {
	sessions        []session.Session
	width           int
	height          int
	err             error
	styles          theme.Styles
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
	m := Model{
		sessions:     opts.Sessions,
		err:          opts.Err,
		styles:       theme.NewStyles(),
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
			theme.ApplyHelpStyles(&h)
			return h
		}(),
		keys:           defaultKeyMap(),
		sortField:      session.SortUpdated,
		sortDescending: true,
		projSortField:  "last",
		projSortDesc:   true,
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickReload())
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
		if uiutil.LastActive(s).Before(cutoff) {
			continue
		}
		if m.filters.tool != "" && s.Tool != m.filters.tool {
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
