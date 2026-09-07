package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/sahilm/fuzzy"

	"github.com/adinhodovic/ai-dash/internal/config"
	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources"
	"github.com/adinhodovic/ai-dash/internal/ui/theme"
	uiutil "github.com/adinhodovic/ai-dash/internal/ui/util"
)

const collapseThreshold = 110

type focusArea int

const (
	focusList focusArea = iota
	focusSearch
)

type Options struct {
	Sessions       []session.Session
	Discovery      sources.Discovery
	Config         config.Config
	Err            error
	Version        string
	BuildTimestamp string
	Renames        map[string]string
	RenamesPath    string
}

type filters struct {
	tool    string
	project string
}

type Model struct {
	sessions              []session.Session
	width                 int
	height                int
	err                   error
	styles                theme.Styles
	meta                  Options
	filters               filters
	focus                 focusArea
	detailCollapsed       bool
	autoCollapsed         bool
	manualCollapse        bool
	statusMessage         string
	searchInput           textinput.Model
	searchQueryBeforeEdit string
	renameInput           textinput.Model
	renaming              bool
	renamingKey           string
	sessionViewport       viewport.Model
	sessionCursor         int
	sourceTable           table.Model
	detailTable           table.Model
	help                  help.Model
	keys                  keyMap
	sortField             session.SortField
	sortDescending        bool
	showHelp              bool
	showSources           bool
	showSubagents         bool
	showAttentionOnly     bool
	showActiveOnly        bool
	showDetailExtra       bool
	seenAttention         map[string]bool
	picker                filterPicker
}

// attentionKeys returns the RenameKey of every session currently needing
// attention, for tracking which ones have already been surfaced to the user.
func attentionKeys(sessions []session.Session) map[string]bool {
	keys := make(map[string]bool)
	for _, s := range sessions {
		if session.NeedsAttention(s) {
			keys[session.RenameKey(s)] = true
		}
	}
	return keys
}

func NewModel(opts Options) Model {
	input := textinput.New()
	input.Placeholder = "search sessions"
	input.CharLimit = 120
	input.Prompt = ""
	input.Blur()
	renameInput := textinput.New()
	renameInput.Placeholder = "rename selected session"
	renameInput.CharLimit = 200
	renameInput.Prompt = ""
	renameInput.Blur()
	renames := opts.Renames
	if renames == nil {
		renames = map[string]string{}
	}
	renamesPath := opts.RenamesPath
	if renamesPath == "" {
		renamesPath = session.DefaultRenamesPath()
	}

	reloadInterval = opts.Config.PollDuration()
	maxSessionAge = opts.Config.DefaultAgeFilterDuration()
	if presets := opts.Config.AgeDurations(); len(presets) > 0 {
		agePresets = presets
	}
	m := Model{
		sessions:        opts.Sessions,
		err:             opts.Err,
		styles:          theme.NewStyles(),
		meta:            opts,
		searchInput:     input,
		renameInput:     renameInput,
		sessionViewport: viewport.New(),
		sourceTable:     newSourceTable(),
		detailTable:     newTable([]table.Column{{Title: "", Width: 10}, {Title: "", Width: 30}}),
		help: func() help.Model {
			h := help.New()
			theme.ApplyHelpStyles(&h)
			return h
		}(),
		keys:           defaultKeyMap(),
		sortField:      session.SortUpdated,
		sortDescending: true,
		seenAttention:  attentionKeys(opts.Sessions),
	}
	m.meta.Renames = renames
	m.meta.RenamesPath = renamesPath
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickReload())
}

func (m *Model) updateDetailCollapse() {
	m.detailCollapsed = m.autoCollapsed || m.manualCollapse
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
		if m.showAttentionOnly && !session.NeedsAttention(s) {
			continue
		}
		if m.showActiveOnly && !session.IsLive(s) {
			continue
		}
		if query != "" && !matchesQuery(s, query) {
			continue
		}
		filtered = append(filtered, s)
	}
	session.SortBy(filtered, m.sortField, m.sortDescending)
	return prioritizeAttention(filtered)
}

// prioritizeAttention stably partitions sessions needing attention to the
// front, preserving whatever order the active sort already produced within
// each partition.
func prioritizeAttention(sessions []session.Session) []session.Session {
	attention := make([]session.Session, 0, len(sessions))
	rest := make([]session.Session, 0, len(sessions))
	for _, s := range sessions {
		if session.NeedsAttention(s) {
			attention = append(attention, s)
		} else {
			rest = append(rest, s)
		}
	}
	return append(attention, rest...)
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
	15 * time.Minute,
	time.Hour,
	24 * time.Hour,
	3 * 24 * time.Hour,
	7 * 24 * time.Hour,
	14 * 24 * time.Hour,
	30 * 24 * time.Hour,
}

func ageLabel(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
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
