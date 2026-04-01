package ui

import (
	"charm.land/bubbles/v2/table"
	"github.com/adin/ai-dash/internal/ui/theme"
)

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
	t.SetStyles(theme.TableStyles())
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
