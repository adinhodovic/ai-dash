package claude

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/adin/ai-dash/internal/sources/shared"
)

func TestDiscoverFindsTranscriptSessions(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "repo-a")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	transcriptPath := filepath.Join(projectDir, "transcript.jsonl")
	if err := os.WriteFile(transcriptPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	t.Setenv("AIDASH_CLAUDE_PROJECTS_DIR", root)

	result, err := New().Discover()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(result.Transcripts) != 1 {
		t.Fatalf("expected 1 transcript, got %d", len(result.Transcripts))
	}
	if len(result.Sessions) != 1 || result.Sessions[0].Project != "repo-a" {
		t.Fatalf("expected imported repo-a session, got %#v", result.Sessions)
	}
}

func TestImportSessionsBuildsDiscoveredSession(t *testing.T) {
	modTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	result := shared.Result{Transcripts: []shared.TranscriptFile{{
		Tool:    "claude",
		Path:    "/tmp/transcript.jsonl",
		Project: "demo",
		ModTime: modTime,
	}}}

	sessions, err := New().ImportSessions(result)
	if err != nil {
		t.Fatalf("import sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].TranscriptPath != "/tmp/transcript.jsonl" || sessions[0].StartedAt != modTime {
		t.Fatalf("unexpected session: %#v", sessions[0])
	}
}
