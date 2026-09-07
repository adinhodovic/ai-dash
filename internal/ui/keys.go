package ui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up                key.Binding
	Down              key.Binding
	GoTop             key.Binding
	GoBottom          key.Binding
	PageUp            key.Binding
	PageDown          key.Binding
	Help              key.Binding
	Search            key.Binding
	SortNext          key.Binding
	SortPrev          key.Binding
	SortToggle        key.Binding
	Tool              key.Binding
	Sort              key.Binding
	Project           key.Binding
	ResumeSession     key.Binding
	RenameSession     key.Binding
	ToggleDetails     key.Binding
	ToggleDetailExtra key.Binding
	NewSession        key.Binding
	AgeRange          key.Binding
	ToggleAgents      key.Binding
	ToggleAttention   key.Binding
	ToggleActiveOnly  key.Binding
	Sources           key.Binding
	Clear             key.Binding
	Quit              key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		GoTop:    key.NewBinding(key.WithKeys("g"), key.WithHelp("g/G", "top/bottom")),
		GoBottom: key.NewBinding(key.WithKeys("G")),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),
		Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Search:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		SortNext:      key.NewBinding(key.WithKeys("]"), key.WithHelp("[/]", "sort")),
		SortPrev:      key.NewBinding(key.WithKeys("[")),
		SortToggle:    key.NewBinding(key.WithKeys("="), key.WithHelp("=", "sort dir")),
		Tool:          key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "tool")),
		Sort:          key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Project:       key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "project")),
		ResumeSession: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "resume")),
		RenameSession: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "rename")),
		ToggleDetails: key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "details")),
		ToggleDetailExtra: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "more info"),
		),
		NewSession:   key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new session")),
		AgeRange:     key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "age range")),
		ToggleAgents: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "agents")),
		ToggleAttention: key.NewBinding(
			key.WithKeys("!"),
			key.WithHelp("!", "attention"),
		),
		ToggleActiveOnly: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "active only"),
		),
		Sources: key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sources")),
		Clear:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k keyMap) shortHelpForFocus(focus focusArea) []key.Binding {
	base := []key.Binding{k.Up, k.Down, k.Help}
	switch focus {
	case focusList:
		base = append(base, k.ResumeSession, k.RenameSession, k.NewSession)
	}
	return append(base, k.Quit)
}

// helpSection is a titled group of bindings for the "?" overlay — labeled,
// so it reads as "here's what Navigate/Filter/Sort mean" instead of an
// unlabeled grid of key/description pairs.
type helpSection struct {
	title    string
	bindings []key.Binding
}

// helpColumns lays helpSections out in two columns, so the overlay stays
// roughly as compact as the old unlabeled grid despite section headers
// adding a line each.
func (k keyMap) helpColumns() [2][]helpSection {
	return [2][]helpSection{
		{
			{"Navigate", []key.Binding{k.Up, k.Down, k.GoTop, k.PageUp, k.PageDown}},
			{"Session", []key.Binding{
				k.ResumeSession, k.RenameSession, k.NewSession,
				k.ToggleDetails, k.ToggleDetailExtra,
			}},
			{"General", []key.Binding{k.Help, k.Sources, k.Quit}},
		},
		{
			{"Filter", []key.Binding{
				k.Search, k.Tool, k.Project, k.AgeRange,
				k.ToggleAgents, k.ToggleAttention, k.ToggleActiveOnly, k.Clear,
			}},
			{"Sort", []key.Binding{k.Sort, k.SortNext, k.SortToggle}},
		},
	}
}
