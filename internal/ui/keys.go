package ui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
)

type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	GoTop         key.Binding
	GoBottom      key.Binding
	PageUp        key.Binding
	PageDown      key.Binding
	Focus         key.Binding
	Help          key.Binding
	Search        key.Binding
	SortNext      key.Binding
	SortPrev      key.Binding
	SortToggle    key.Binding
	Tool          key.Binding
	Sort          key.Binding
	Project       key.Binding
	SavePreset    key.Binding
	LoadPreset    key.Binding
	ResumeSession   key.Binding
	ToggleDetails key.Binding
	NewSession    key.Binding
	AgeRange      key.Binding
	ToggleAgents  key.Binding
	Sources       key.Binding
	Clear         key.Binding
	Quit          key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		GoTop:         key.NewBinding(key.WithKeys("g"), key.WithHelp("g/G", "top/bottom")),
		GoBottom:      key.NewBinding(key.WithKeys("G")),
		PageUp:        key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up")),
		PageDown:      key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn", "page down")),
		Focus:         key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus")),
		Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Search:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		SortNext:      key.NewBinding(key.WithKeys("]"), key.WithHelp("[/]", "sort")),
		SortPrev:      key.NewBinding(key.WithKeys("[")),
		SortToggle:    key.NewBinding(key.WithKeys("="), key.WithHelp("=", "sort dir")),
		Tool:          key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "tool")),
		Sort:          key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Project:       key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "project")),
		SavePreset:    key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "save preset")),
		LoadPreset:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "load preset")),
		ResumeSession:   key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "resume")),
		ToggleDetails: key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "details")),
		NewSession:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new session")),
		AgeRange:      key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "age range")),
		ToggleAgents:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "agents")),
		Sources:       key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sources")),
		Clear:         key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
		Quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Help, k.Focus, k.Search, k.Tool, k.Sort, k.Project, k.Clear, k.Quit}
}

func (k keyMap) shortHelpForFocus(focus focusArea) []key.Binding {
	base := []key.Binding{k.Up, k.Down, k.Help, k.Focus, k.Search}
	switch focus {
	case focusList:
		base = append(base, k.ResumeSession, k.NewSession)
	case focusFilters:
		base = append(base, k.NewSession)
	}
	return append(base, k.Tool, k.Sort, k.Project, k.AgeRange, k.Clear, k.Quit)
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.GoTop, k.PageUp, k.PageDown},
		{k.Help, k.Focus, k.Search, k.ResumeSession, k.NewSession, k.ToggleDetails, k.Clear, k.Quit},
		{k.Tool, k.Sort, k.Project, k.AgeRange, k.ToggleAgents, k.Sources},
		{k.SortNext, k.SortToggle, k.SavePreset, k.LoadPreset},
	}
}

type detailItem struct {
	title string
	desc  string
}

func newTable(columns []table.Column) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
	)
	t.SetStyles(tableStyles())
	return t
}

func newSessionTable() table.Model {
	return newTable([]table.Column{
		{Title: "Tool", Width: 7},
		{Title: "Project", Width: 16},
		{Title: "Status", Width: 10},
		{Title: "Started", Width: 16},
		{Title: "Summary", Width: 36},
	})
}

func newSourceTable() table.Model {
	return newTable([]table.Column{
		{Title: "Tool", Width: 8},
		{Title: "Kind", Width: 12},
		{Title: "Status", Width: 10},
		{Title: "Path", Width: 40},
	})
}

func newRelatedTable() table.Model {
	return newTable([]table.Column{
		{Title: "Tool", Width: 8},
		{Title: "Project", Width: 14},
		{Title: "Relation", Width: 12},
		{Title: "Started", Width: 16},
		{Title: "Summary", Width: 28},
	})
}
