package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverCandidateFilesWithPatterns(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		filepath.Join(root, "session-1.json"),
		filepath.Join(root, "history.jsonl"),
		filepath.Join(root, "config.json"),
	}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	matches, err := DiscoverCandidateFiles(root)
	if err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 candidate files, got %#v", matches)
	}
}

func TestImportGenericSessionsParsesJSONSession(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "session.json")
	content := `{"messages":[{"role":"user","content":"help"}],"model":"gpt-5-codex","summary":"Fix auth flow regression","status":"completed","started_at":"2026-03-29T12:00:00Z","ended_at":"2026-03-29T12:05:00Z"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write session file: %v", err)
	}

	sessions := ImportGenericSessions("codex", []string{path})
	if len(sessions) != 1 {
		t.Fatalf("expected one parsed session, got %#v", sessions)
	}
	if sessions[0].Model != "gpt-5-codex" || sessions[0].Status != "completed" {
		t.Fatalf("unexpected session: %#v", sessions[0])
	}
}
