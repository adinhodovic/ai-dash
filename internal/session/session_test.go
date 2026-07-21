package session

import (
	"path/filepath"
	"testing"
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
