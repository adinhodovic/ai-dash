package ui

import (
	"testing"
	"time"

	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources/shared"
)

func TestAttentionLabel(t *testing.T) {
	now := time.Now()
	s := session.Session{
		ID:           "1",
		Tool:         "claude",
		CurrentState: "waiting",
		Status:       "active",
		EndedAt:      now.Add(-45 * time.Minute),
	}
	got := attentionLabel(s)
	want := "waiting 45m"
	if got != want {
		t.Fatalf("attentionLabel() = %q, want %q", got, want)
	}

	notFlagged := session.Session{Status: "completed", CurrentState: "done"}
	if got := attentionLabel(notFlagged); got != "" {
		t.Fatalf("attentionLabel() for non-attention session = %q, want empty", got)
	}
}

func TestSeenAttentionSeededAtStartup(t *testing.T) {
	now := time.Now()
	waiting := session.Session{
		ID:           "1",
		Tool:         "claude",
		CurrentState: "waiting",
		Status:       "active",
		EndedAt:      now,
	}
	m := NewModel(Options{
		Sessions:  []session.Session{waiting},
		Discovery: shared.Discovery{},
		Version:   "test",
	})
	if !m.seenAttention[session.RenameKey(waiting)] {
		t.Fatal("session already needing attention at startup should be seeded as seen")
	}
}

func TestToggleAttentionMarksSeen(t *testing.T) {
	now := time.Now()
	waiting := session.Session{
		ID:           "1",
		Tool:         "claude",
		CurrentState: "waiting",
		Status:       "active",
		EndedAt:      now,
	}
	m := NewModel(Options{
		Sessions:  []session.Session{},
		Discovery: shared.Discovery{},
		Version:   "test",
	})
	m.sessions = []session.Session{waiting}
	if m.seenAttention[session.RenameKey(waiting)] {
		t.Fatal("session added after startup should not be pre-seeded as seen")
	}
	m = sendKey(m, "!")
	if !m.seenAttention[session.RenameKey(waiting)] {
		t.Fatal("toggling the attention filter should mark visible attention sessions as seen")
	}
}
