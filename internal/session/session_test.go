package session

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		name string
		s    Session
		want string
	}{
		{"normalized current state", Session{CurrentState: "tool call"}, "tool call"},
		{"active fallback", Session{Status: "active"}, "running"},
		{"completed", Session{Status: "completed"}, "done"},
		{"aborted", Session{Status: "aborted"}, "aborted"},
		{"unknown", Session{}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StatusLabel(tt.s); got != tt.want {
				t.Fatalf("StatusLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAttention(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		s    Session
		want AttentionReason
	}{
		{
			"waiting",
			Session{CurrentState: "waiting", Status: "active", EndedAt: now},
			AttentionWaiting,
		},
		{
			"max tokens",
			Session{CurrentState: "max tokens", Status: "active", EndedAt: now},
			AttentionMaxTokens,
		},
		{
			"old waiting session is abandoned, not flagged",
			Session{CurrentState: "waiting", Status: "active", EndedAt: now.Add(-3 * time.Hour)},
			AttentionNone,
		},
		{
			"old max tokens session is abandoned, not flagged",
			Session{CurrentState: "max tokens", Status: "active", EndedAt: now.Add(-3 * time.Hour)},
			AttentionNone,
		},
		{
			"stalled while active",
			Session{CurrentState: "running", Status: "active", EndedAt: now.Add(-45 * time.Minute)},
			AttentionStalled,
		},
		{
			"active but recent",
			Session{CurrentState: "running", Status: "active", EndedAt: now.Add(-1 * time.Minute)},
			AttentionNone,
		},
		{
			"abandoned past the stale ceiling",
			Session{CurrentState: "running", Status: "active", EndedAt: now.Add(-3 * time.Hour)},
			AttentionNone,
		},
		{
			"aborted stays none even if old",
			Session{CurrentState: "aborted", Status: "aborted", EndedAt: now.Add(-time.Hour)},
			AttentionNone,
		},
		{
			"completed stays none even if old",
			Session{CurrentState: "done", Status: "completed", EndedAt: now.Add(-time.Hour)},
			AttentionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Attention(tt.s); got != tt.want {
				t.Fatalf("Attention() = %q, want %q", got, tt.want)
			}
			if want := tt.want != AttentionNone; NeedsAttention(tt.s) != want {
				t.Fatalf("NeedsAttention() = %v, want %v", NeedsAttention(tt.s), want)
			}
		})
	}
}

func TestIsLive(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		s    Session
		want bool
	}{
		{
			"active and recent",
			Session{Status: "active", EndedAt: now.Add(-1 * time.Minute)},
			true,
		},
		{
			"active but stalled past staleAfter",
			Session{Status: "active", EndedAt: now.Add(-45 * time.Minute)},
			false,
		},
		{
			"completed but touched within the last 15m still counts",
			Session{Status: "completed", EndedAt: now.Add(-5 * time.Minute)},
			true,
		},
		{
			"aborted but touched within the last 15m still counts",
			Session{Status: "aborted", EndedAt: now.Add(-10 * time.Minute)},
			true,
		},
		{
			"completed and old is not live",
			Session{Status: "completed", EndedAt: now.Add(-time.Hour)},
			false,
		},
		{
			"falls back to StartedAt when EndedAt is zero",
			Session{Status: "completed", StartedAt: now.Add(-time.Minute)},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLive(tt.s); got != tt.want {
				t.Fatalf("IsLive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyRenames(t *testing.T) {
	sessions := []Session{
		{ID: "1", Tool: "claude", Summary: "original"},
		{ID: "2", Tool: "codex", Summary: "keep"},
	}

	ApplyRenames(sessions, map[string]string{"claude/1": "renamed"})

	if sessions[0].Summary != "renamed" {
		t.Fatalf("renamed summary = %q, want renamed", sessions[0].Summary)
	}
	if sessions[1].Summary != "keep" {
		t.Fatalf("untouched summary = %q, want keep", sessions[1].Summary)
	}
}

func TestLoadSaveRenames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session-renames.json")
	want := map[string]string{"claude/1": "renamed"}

	if err := SaveRenames(path, want); err != nil {
		t.Fatalf("SaveRenames() error = %v", err)
	}
	got, err := LoadRenames(path)
	if err != nil {
		t.Fatalf("LoadRenames() error = %v", err)
	}
	if got["claude/1"] != want["claude/1"] {
		t.Fatalf("loaded rename = %q, want %q", got["claude/1"], want["claude/1"])
	}
}
