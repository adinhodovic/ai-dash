package ui

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
		got := truncate(tt.value, tt.limit)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.value, tt.limit, got, tt.want)
		}
	}
}

func TestTruncateForCell(t *testing.T) {
	if got := truncateForCell("", 10); got != "-" {
		t.Errorf("empty input should be dash, got %q", got)
	}
	if got := truncateForCell("  hello  ", 10); got != "hello" {
		t.Errorf("should trim whitespace, got %q", got)
	}
}

func TestCleanProjectName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"/home/adin/oss/ai-dash", "~/oss/ai-dash"},
		{"~/myproject", "~/myproject"},
		{"", "unknown"},
		{"  ", "unknown"},
		{"dotfiles", "dotfiles"},
		{"oss-ai-dash", "oss-ai-dash"},
	}
	for _, tt := range tests {
		got := cleanProjectName(tt.input)
		if got != tt.want {
			t.Errorf("cleanProjectName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCleanSummary(t *testing.T) {
	if got := cleanSummary(""); got != "-" {
		t.Errorf("empty should be dash, got %q", got)
	}
	if got := cleanSummary("req_abc123"); got != "Imported session" {
		t.Errorf("request ID should be cleaned, got %q", got)
	}
	if got := cleanSummary("a1b2c3d4-e5f6-7890-abcd-ef1234567890"); got != "Imported session" {
		t.Errorf("UUID should be cleaned, got %q", got)
	}
	if got := cleanSummary("fix the bug"); got != "fix the bug" {
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
		got := formatCost(tt.cost)
		if got != tt.want {
			t.Errorf("formatCost(%v) = %q, want %q", tt.cost, got, tt.want)
		}
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1500, "1,500"},
		{1000000, "1,000,000"},
	}
	for _, tt := range tests {
		got := formatCount(tt.n)
		if got != tt.want {
			t.Errorf("formatCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	if got := formatTokens(0, 0); got != "n/a" {
		t.Errorf("zero tokens should be n/a, got %q", got)
	}
	got := formatTokens(1500, 500)
	if got != "1,500 in / 500 out" {
		t.Errorf("formatTokens(1500, 500) = %q", got)
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
		{"30 seconds", session.Session{Status: "done", StartedAt: now, EndedAt: now.Add(30 * time.Second)}, "30s"},
		{"5 minutes", session.Session{Status: "done", StartedAt: now, EndedAt: now.Add(5*time.Minute + 30*time.Second)}, "5m 30s"},
		{"2 hours", session.Session{Status: "done", StartedAt: now, EndedAt: now.Add(2*time.Hour + 15*time.Minute)}, "2h 15m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := durationLabel(tt.s)
			if got != tt.want {
				t.Errorf("durationLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContentHeight(t *testing.T) {
	tests := []struct {
		termH, want int
	}{
		{24, 22}, // 24 - 2
		{10, 8},  // 10 - 2
		{3, 4},   // clamped
		{0, 4},   // clamped
	}
	for _, tt := range tests {
		got := contentHeight(tt.termH)
		if got != tt.want {
			t.Errorf("contentHeight(%d) = %d, want %d", tt.termH, got, tt.want)
		}
	}
}

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

func TestRelationLabel(t *testing.T) {
	parent := session.Session{ID: "parent-1", Project: "proj", Repo: "repo"}
	child := session.Session{ID: "child-1", ParentID: "parent-1", Project: "proj", Repo: "repo"}
	sibling := session.Session{ID: "sib-1", Project: "proj", Repo: "other"}
	unrelated := session.Session{ID: "other-1", Project: "other", Repo: "other"}

	if got := relationLabel(parent, child); got != "child" {
		t.Errorf("child relation = %q", got)
	}
	if got := relationLabel(child, parent); got != "parent" {
		t.Errorf("parent relation = %q", got)
	}
	if got := relationLabel(parent, sibling); got != "project" {
		t.Errorf("project relation = %q", got)
	}
	if got := relationLabel(parent, unrelated); got != "" {
		t.Errorf("unrelated should be empty, got %q", got)
	}
}

func TestValueOrUnknown(t *testing.T) {
	if got := valueOrUnknown(""); got != "unknown" {
		t.Errorf("empty = %q", got)
	}
	if got := valueOrUnknown("x"); got != "x" {
		t.Errorf("non-empty = %q", got)
	}
}
