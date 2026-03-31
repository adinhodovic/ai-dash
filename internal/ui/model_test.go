package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources/shared"
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

	m.filters.tool = "claude"
	filtered = m.filteredSessions()
	if len(filtered) != 2 {
		t.Errorf("tool=claude: expected 2, got %d", len(filtered))
	}

	m.filters.tool = ""
	m.filters.status = "completed"
	filtered = m.filteredSessions()
	if len(filtered) != 2 {
		t.Errorf("status=completed: expected 2, got %d", len(filtered))
	}

	m.filters.status = ""
	m.filters.project = "beta"
	filtered = m.filteredSessions()
	if len(filtered) != 1 {
		t.Errorf("project=beta: expected 1, got %d", len(filtered))
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

func TestCycleForward(t *testing.T) {
	m := testModel()
	if m.focus != focusList {
		t.Fatal("should start at focusList")
	}
	// Tab order: Sessions -> Projects
	m.cycleForward()
	if m.focus != focusFilters {
		t.Errorf("after 1 tab: got %d, want focusFilters(%d)", m.focus, focusFilters)
	}
	m.cycleForward()
	if m.focus != focusList {
		t.Errorf("after 2 tab: got %d, want focusList(%d)", m.focus, focusList)
	}
}

func TestCycleBackward(t *testing.T) {
	m := testModel()
	m.cycleBackward()
	if m.focus != focusFilters {
		t.Errorf("backward from list: got %d, want focusFilters(%d)", m.focus, focusFilters)
	}
}

func TestCycleSkipsSearch(t *testing.T) {
	m := testModel()
	// Cycle through all positions 10 times
	for i := 0; i < 30; i++ {
		m.cycleForward()
		if m.focus == focusSearch {
			t.Fatal("tab should never land on focusSearch")
		}
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
	m.filters.tool = "claude"
	m.searchInput.SetValue("test")
	m = sendKey(m, "c")
	if m.filters.tool != "" {
		t.Error("filters should be cleared")
	}
	if m.searchInput.Value() != "" {
		t.Error("search should be cleared")
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
	m.filters.tool = "nonexistent"
	view := m.View()
	if view.Content == "" {
		t.Error("no matches should still render")
	}
}

func TestSyncTableKeepsSelectedCursor(t *testing.T) {
	m := testModel()
	m = resize(m, 120, 40)
	m.sessionTable.SetCursor(2)
	filtered := m.filteredSessions()
	m.syncTable(filtered)
	if got := m.sessionTable.Cursor(); got != 2 {
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
	firstOverview := m.overviewTable.Height()
	firstDetail := m.detailTable.Height()
	firstRelated := m.relatedTable.Height()

	m.sessionTable.SetCursor(1)
	m.syncAllTables(filtered)

	if got := m.overviewTable.Height(); got != firstOverview {
		t.Fatalf("overview height = %d, want %d", got, firstOverview)
	}
	if got := m.detailTable.Height(); got != firstDetail {
		t.Fatalf("detail height = %d, want %d", got, firstDetail)
	}
	if got := m.relatedTable.Height(); got != firstRelated {
		t.Fatalf("related height = %d, want %d", got, firstRelated)
	}
}

func TestDetailPaneSectionHeightsAreStable(t *testing.T) {
	summary, detail, related := detailPaneSectionHeights(40)
	if summary != 2 {
		t.Fatalf("summary height = %d, want 2", summary)
	}
	if related != 6 {
		t.Fatalf("related height = %d, want 6", related)
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
