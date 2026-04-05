package util

import (
	"testing"
	"time"

	"github.com/adin/ai-dash/internal/session"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		value string
		limit int
		want  string
	}{
		{"hello world", 5, "hell~"},
		{"hi", 5, "hi"},
		{"hello", 5, "hello"},
		{"", 5, ""},
		{"abc", 0, "abc"},
		{"ab", 1, "a"},
	}
	for _, tt := range tests {
		got := Truncate(tt.value, tt.limit)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.value, tt.limit, got, tt.want)
		}
	}
}

func TestTruncateForCell(t *testing.T) {
	if got := TruncateForCell("", 10); got != "-" {
		t.Errorf("empty input should be dash, got %q", got)
	}
	if got := TruncateForCell("  hello  ", 10); got != "hello" {
		t.Errorf("should trim whitespace, got %q", got)
	}
}

func TestCleanProjectName(t *testing.T) {
	home := homeDir
	tests := []struct {
		input string
		want  string
	}{
		{home + "/oss/ai-dash", "~/oss/ai-dash"},
		{"~/myproject", "~/myproject"},
		{"", "unknown"},
		{"  ", "unknown"},
		{"dotfiles", "dotfiles"},
		{"oss-ai-dash", "oss-ai-dash"},
	}
	for _, tt := range tests {
		got := CleanProjectName(tt.input)
		if got != tt.want {
			t.Errorf("CleanProjectName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCleanSummary(t *testing.T) {
	if got := CleanSummary(""); got != "-" {
		t.Errorf("empty should be dash, got %q", got)
	}
	if got := CleanSummary("req_abc123"); got != "Imported session" {
		t.Errorf("request ID should be cleaned, got %q", got)
	}
	if got := CleanSummary("a1b2c3d4-e5f6-7890-abcd-ef1234567890"); got != "Imported session" {
		t.Errorf("UUID should be cleaned, got %q", got)
	}
	if got := CleanSummary("fix the bug"); got != "fix the bug" {
		t.Errorf("normal summary should pass through, got %q", got)
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		cost float64
		want string
	}{
		{0, "n/a"},
		{0.005, "$0.0050"},
		{1.50, "$1.50"},
		{100.0, "$100.00"},
	}
	for _, tt := range tests {
		got := FormatCost(tt.cost)
		if got != tt.want {
			t.Errorf("FormatCost(%v) = %q, want %q", tt.cost, got, tt.want)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	if got := FormatTokens(0, 0); got != "n/a" {
		t.Errorf("zero tokens should be n/a, got %q", got)
	}
	got := FormatTokens(1500, 500)
	if got != "1,500 in / 500 out" {
		t.Errorf("FormatTokens(1500, 500) = %q", got)
	}
}

func TestDurationLabel(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		s    session.Session
		want string
	}{
		{"active", session.Session{Status: "active"}, "running"},
		{"zero end", session.Session{Status: "done", StartedAt: now}, "unknown"},
		{
			"30 seconds",
			session.Session{Status: "done", StartedAt: now, EndedAt: now.Add(30 * time.Second)},
			"30s",
		},
		{
			"5 minutes",
			session.Session{
				Status:    "done",
				StartedAt: now,
				EndedAt:   now.Add(5*time.Minute + 30*time.Second),
			},
			"5m 30s",
		},
		{
			"2 hours",
			session.Session{
				Status:    "done",
				StartedAt: now,
				EndedAt:   now.Add(2*time.Hour + 15*time.Minute),
			},
			"2h 15m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DurationLabel(tt.s)
			if got != tt.want {
				t.Errorf("DurationLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSessionStatusLabel(t *testing.T) {
	tests := []struct {
		name string
		s    session.Session
		want string
	}{
		{"active default", session.Session{Status: "active"}, "running"},
		{"active waiting", session.Session{Status: "active", Meta: map[string]string{"stop_reason": "end_turn"}}, "waiting"},
		{"active tool call", session.Session{Status: "active", Meta: map[string]string{"stop_reason": "tool_use"}}, "tool call"},
		{"completed", session.Session{Status: "completed"}, "done"},
		{"aborted", session.Session{Status: "aborted"}, "aborted"},
		{"unknown", session.Session{}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SessionStatusLabel(tt.s); got != tt.want {
				t.Fatalf("SessionStatusLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRelationLabel(t *testing.T) {
	parent := session.Session{ID: "parent-1", Project: "proj", Repo: "repo"}
	child := session.Session{ID: "child-1", ParentID: "parent-1", Project: "proj", Repo: "repo"}
	sibling := session.Session{ID: "sib-1", Project: "proj", Repo: "other"}
	unrelated := session.Session{ID: "other-1", Project: "other", Repo: "other"}

	if got := RelationLabel(parent, child); got != "child" {
		t.Errorf("child relation = %q", got)
	}
	if got := RelationLabel(child, parent); got != "parent" {
		t.Errorf("parent relation = %q", got)
	}
	if got := RelationLabel(parent, sibling); got != "project" {
		t.Errorf("project relation = %q", got)
	}
	if got := RelationLabel(parent, unrelated); got != "" {
		t.Errorf("unrelated should be empty, got %q", got)
	}
}

func TestValueOrUnknown(t *testing.T) {
	if got := ValueOrUnknown(""); got != "unknown" {
		t.Errorf("empty = %q", got)
	}
	if got := ValueOrUnknown("x"); got != "x" {
		t.Errorf("non-empty = %q", got)
	}
}
