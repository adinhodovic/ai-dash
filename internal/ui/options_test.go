package ui

import (
	"testing"

	"github.com/adin/ai-dash/internal/session"
)

func TestNextPrevSortField(t *testing.T) {
	got := nextSortField(session.SortUpdated)
	if got != session.SortTool {
		t.Errorf("next after updated = %v, want tool", got)
	}
	got = prevSortField(session.SortUpdated)
	if got != session.SortSummary {
		t.Errorf("prev before updated = %v, want summary", got)
	}
}
