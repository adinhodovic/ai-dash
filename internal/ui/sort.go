package ui

import "fmt"

func (m *Model) cycleSortForward() {
	m.sortField = nextSortField(m.sortField)
	m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
}

func (m *Model) cycleSortBackward() {
	m.sortField = prevSortField(m.sortField)
	m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
}

func (m *Model) toggleSortDirection() {
	m.sortDescending = !m.sortDescending
	m.statusMessage = fmt.Sprintf("Sort: %s", m.sortLabel())
}
