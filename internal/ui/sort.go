package ui

import "fmt"

var projectSortFields = []string{"last", "project", "sessions"}

func (m *Model) cycleSortForward() {
	if m.focus == focusFilters {
		m.projSortField = nextInSlice(m.projSortField, projectSortFields)
		m.statusMessage = fmt.Sprintf("Projects sort: %s", m.projSortField)
	} else {
		m.sortField = nextSortField(m.sortField)
		m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
	}
}

func (m *Model) cycleSortBackward() {
	if m.focus == focusFilters {
		m.projSortField = prevInSlice(m.projSortField, projectSortFields)
		m.statusMessage = fmt.Sprintf("Projects sort: %s", m.projSortField)
	} else {
		m.sortField = prevSortField(m.sortField)
		m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
	}
}

func (m *Model) toggleSortDirection() {
	if m.focus == focusFilters {
		m.projSortDesc = !m.projSortDesc
		m.statusMessage = fmt.Sprintf("Projects sort: %s", m.projSortField)
	} else {
		m.sortDescending = !m.sortDescending
		m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
	}
}

func nextInSlice(current string, options []string) string {
	for i, v := range options {
		if v == current {
			return options[(i+1)%len(options)]
		}
	}
	return options[0]
}

func prevInSlice(current string, options []string) string {
	for i, v := range options {
		if v == current {
			return options[(i-1+len(options))%len(options)]
		}
	}
	return options[0]
}
