package ui

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources/shared"
)

func testSessions() []session.Session {
	now := time.Now()
	return []session.Session{
		{
			ID:        "1",
			Tool:      "claude",
			Project:   "alpha",
			Status:    "active",
			StartedAt: now,
			Summary:   "working on tests",
		},
		{
			ID:        "2",
			Tool:      "codex",
			Project:   "alpha",
			Status:    "completed",
			StartedAt: now.Add(-time.Hour),
			EndedAt:   now,
			Summary:   "refactored auth",
		},
		{
			ID:        "3",
			Tool:      "opencode",
			Project:   "beta",
			Status:    "completed",
			StartedAt: now.Add(-2 * time.Hour),
			EndedAt:   now.Add(-time.Hour),
			Summary:   "fixed bug",
		},
		{
			ID:        "4",
			Tool:      "claude",
			Project:   "gamma",
			Status:    "aborted",
			StartedAt: now.Add(-3 * time.Hour),
			EndedAt:   now.Add(-2 * time.Hour),
			Summary:   "abandoned approach",
		},
	}
}

func testModel() Model {
	return NewModel(Options{
		Sessions:  testSessions(),
		Discovery: shared.Discovery{},
		Version:   "test",
	})
}

func resize(m Model, w, h int) Model {
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(Model)
}

func sendKey(m Model, k string) Model {
	updated, _ := m.Update(tea.KeyPressMsg{Code: rune(k[0]), Text: k})
	return updated.(Model)
}

func sendNamedKey(m Model, k string) Model {
	updated, _ := m.Update(tea.KeyPressMsg{Text: k})
	return updated.(Model)
}

func TestNewModel(t *testing.T) {
	m := testModel()
	if len(m.sessions) != 4 {
		t.Fatalf("expected 4 sessions, got %d", len(m.sessions))
	}
	if m.focus != focusList {
		t.Errorf("initial focus should be focusList, got %d", m.focus)
	}
	if m.sortField != session.SortUpdated {
		t.Errorf("initial sort should be updated, got %v", m.sortField)
	}
	if !m.sortDescending {
		t.Error("initial sort should be descending")
	}
}

func TestFilteredSessions(t *testing.T) {
	m := testModel()

	filtered := m.filteredSessions()
	if len(filtered) != 4 {
		t.Fatalf("no filters: expected 4, got %d", len(filtered))
	}

	m.filters.tools = []string{"claude"}
	filtered = m.filteredSessions()
	if len(filtered) != 2 {
		t.Errorf("tool=claude: expected 2, got %d", len(filtered))
	}

	m.filters.tools = nil
	m.filters.projects = []string{"beta"}
	filtered = m.filteredSessions()
	if len(filtered) != 1 {
		t.Errorf("project=beta: expected 1, got %d", len(filtered))
	}

	m.filters.projects = nil
	m.filters.tools = []string{"claude", "codex"}
	filtered = m.filteredSessions()
	if len(filtered) != 3 {
		t.Errorf("tool in [claude,codex]: expected 3, got %d", len(filtered))
	}
}

func TestSearchFilter(t *testing.T) {
	m := testModel()
	m.searchInput.SetValue("bug")
	filtered := m.filteredSessions()
	if len(filtered) != 1 {
		t.Errorf("search 'bug': expected 1, got %d", len(filtered))
	}
	if filtered[0].ID != "3" {
		t.Errorf("search 'bug': expected session 3, got %s", filtered[0].ID)
	}
}

func TestSortCycling(t *testing.T) {
	m := testModel()
	m = resize(m, 120, 40)

	initial := m.sortField
	m = sendKey(m, "]")
	if m.sortField == initial {
		t.Error("sort field should change after ]")
	}
	m = sendKey(m, "=")
	if m.sortDescending {
		t.Error("sort direction should toggle to ascending")
	}
}

func TestClearFilters(t *testing.T) {
	m := testModel()
	m = resize(m, 120, 40)
	m.filters.tools = []string{"claude"}
	m.searchInput.SetValue("test")
	m = sendKey(m, "c")
	if len(m.filters.tools) != 0 {
		t.Error("filters should be cleared")
	}
	if m.searchInput.Value() != "" {
		t.Error("search should be cleared")
	}
}

func TestProjectPickerEnterAfterSearchSelectsItem(t *testing.T) {
	m := testModel()
	m = resize(m, 120, 40)
	m = sendKey(m, "p")
	m.picker.list.SetFilterText("beta")
	m.picker.list.SetFilterState(list.Filtering)

	m = sendNamedKey(m, "enter")

	if !m.picker.active {
		t.Fatal("multi-select picker should remain open after selecting a value")
	}
	if len(m.filters.projects) != 1 || m.filters.projects[0] != "beta" {
		t.Fatalf("projects = %v, want [beta]", m.filters.projects)
	}
}

func TestProjectPickerSearchUpdatesSelectedCheckbox(t *testing.T) {
	m := testModel()
	m = resize(m, 120, 40)
	m = sendKey(m, "p")
	m.picker.list.SetFilterText("beta")

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updated.(Model)
	if cmd != nil {
		if msg := cmd(); msg != nil {
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
	}

	item, ok := m.picker.list.SelectedItem().(pickerItem)
	if !ok || !item.checked {
		t.Fatalf("selected item = %#v, want checked beta", m.picker.list.SelectedItem())
	}
}

func TestProjectPickerNoMatchEnterDoesNotSelectItem(t *testing.T) {
	m := testModel()
	m = resize(m, 120, 40)
	m = sendKey(m, "p")
	m.picker.list.SetFilterText("does-not-exist")
	m.picker.list.SetFilterState(list.Filtering)

	m = sendNamedKey(m, "enter")

	if len(m.filters.projects) != 0 {
		t.Fatalf("projects = %v, want no selection", m.filters.projects)
	}
}

func TestNewSessionPickerOmitsAllOption(t *testing.T) {
	m := testModel()
	m = resize(m, 120, 40)
	m = sendKey(m, "n")

	item, ok := m.picker.list.SelectedItem().(pickerItem)
	if !ok || item.value == "" {
		t.Fatalf("selected item = %#v, want a tool", m.picker.list.SelectedItem())
	}
}

func TestFilterPickerShowsOptionsOutsideCurrentFilter(t *testing.T) {
	m := testModel()
	m.filters.tools = []string{"claude"}
	m = resize(m, 120, 40)
	m = sendKey(m, "t")

	values := make(map[string]bool)
	for _, item := range m.picker.list.VisibleItems() {
		values[item.(pickerItem).value] = true
	}
	if !values["codex"] {
		t.Fatalf("tool options = %v, want codex despite active claude filter", values)
	}
}

func TestToolPickerSupportsMultipleSelections(t *testing.T) {
	m := testModel()
	m = resize(m, 120, 40)
	m = sendKey(m, "t")

	// Multi-select pickers omit the empty "all" option; select claude and codex.
	m.picker.list.Select(0)
	m = sendNamedKey(m, "enter")
	m.picker.list.Select(1)
	m = sendNamedKey(m, "enter")

	if len(m.filters.tools) != 2 || !slices.Contains(m.filters.tools, "claude") ||
		!slices.Contains(m.filters.tools, "codex") {
		t.Fatalf("tools = %v, want claude and codex", m.filters.tools)
	}
}

func TestRenameSession(t *testing.T) {
	m := testModel()
	m.meta.RenamesPath = filepath.Join(t.TempDir(), "session-renames.json")
	m = resize(m, 120, 40)
	m = sendKey(m, "R")
	if !m.renaming {
		t.Fatal("expected rename mode")
	}
	m.renameInput.SetValue("renamed session")
	m = sendNamedKey(m, "enter")
	if m.renaming {
		t.Fatal("rename mode should close after save")
	}
	filtered := m.filteredSessions()
	if got := filtered[m.sessionCursor].Summary; got != "renamed session" {
		t.Fatalf("selected summary = %q, want renamed session", got)
	}
	if got := m.meta.Renames[session.RenameKey(filtered[m.sessionCursor])]; got != "renamed session" {
		t.Fatalf("stored rename = %q, want renamed session", got)
	}
}

func TestRenameShortcutWorksWhileSearching(t *testing.T) {
	m := testModel()
	m = resize(m, 120, 40)
	m = sendKey(m, "/")

	m = sendNamedKey(m, "ctrl+r")

	if !m.renaming {
		t.Fatal("ctrl+r should start renaming while search is focused")
	}
}

func TestViewNoPanic(t *testing.T) {
	// Test that View doesn't panic with various states
	sizes := [][2]int{{0, 0}, {80, 24}, {200, 50}, {40, 10}}
	for _, size := range sizes {
		m := testModel()
		m = resize(m, size[0], size[1])
		view := m.View()
		_ = view // just checking no panic
	}
}

func TestViewFitsTerminal(t *testing.T) {
	sizes := [][2]int{{80, 24}, {120, 40}, {200, 50}}
	for _, size := range sizes {
		w, h := size[0], size[1]
		m := testModel()
		m = resize(m, w, h)
		view := m.View()
		lines := len(splitLines(view.Content))
		if lines > h {
			t.Errorf("View at %dx%d: %d lines > terminal height %d", w, h, lines, h)
		}
	}
}

func TestViewEmptySessions(t *testing.T) {
	m := NewModel(Options{Sessions: nil, Version: "test"})
	m = resize(m, 80, 24)
	view := m.View()
	if view.Content == "" {
		t.Error("empty sessions should still render")
	}
}

func TestViewNoMatchingFilters(t *testing.T) {
	m := testModel()
	m = resize(m, 80, 24)
	m.filters.tools = []string{"nonexistent"}
	view := m.View()
	if view.Content == "" {
		t.Error("no matches should still render")
	}
}

func TestSyncTableKeepsSelectedCursor(t *testing.T) {
	m := testModel()
	m = resize(m, 120, 40)
	m.sessionCursor = 2
	filtered := m.filteredSessions()
	m.syncTable(filtered)
	if got := m.sessionCursor; got != 2 {
		t.Fatalf("cursor = %d, want 2", got)
	}
}

func TestLayoutHeightsStayStableAcrossSelection(t *testing.T) {
	now := time.Now()
	m := NewModel(Options{
		Sessions: []session.Session{
			{
				ID:        "1",
				Tool:      "claude",
				Project:   "alpha",
				Status:    "active",
				StartedAt: now,
				Summary:   "short",
			},
			{
				ID:        "2",
				Tool:      "codex",
				Project:   "alpha",
				Status:    "completed",
				StartedAt: now.Add(-time.Hour),
				EndedAt:   now,
				Summary:   "a much longer summary that should not change pane heights even when the selected session has more detail fields",
				Repo:      "/tmp/repo",
				Branch:    "main",
				Slug:      "slug",
				ParentID:  "parent",
				Tags:      []string{"one", "two"},
			},
		},
		Version: "test",
	})
	m = resize(m, 120, 40)
	filtered := m.filteredSessions()
	firstDetail := m.detailTable.Height()

	m.sessionCursor = 1
	m.syncAllTables(filtered)

	if got := m.detailTable.Height(); got != firstDetail {
		t.Fatalf("detail height = %d, want %d", got, firstDetail)
	}
}

func TestDetailPaneSectionHeightsAreStable(t *testing.T) {
	summary, detail := detailPaneSectionHeights(40)
	if summary != 2 {
		t.Fatalf("summary height = %d, want 2", summary)
	}
	if detail < 3 {
		t.Fatalf("detail height = %d, want at least 3", detail)
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
