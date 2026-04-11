package ui

import (
	"testing"

	"github.com/adinhodovic/ai-dash/internal/session"
)

func TestNextPrevSortField(t *testing.T) {
	got := nextSortField(session.SortUpdated)
	if got != session.SortTool {
		t.Errorf("next after updated = %v, want tool", got)
	}
	got = nextSortField(session.SortTool)
	if got != session.SortStatus {
		t.Errorf("next after tool = %v, want status", got)
	}
	got = prevSortField(session.SortUpdated)
	if got != session.SortSummary {
		t.Errorf("prev before updated = %v, want summary", got)
	}
}
