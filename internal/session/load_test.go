package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileSortsSessionsDescending(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	content := `{
		"sessions": [
			{"id":"older","tool":"codex","project":"a","status":"completed","started_at":"2026-03-28T10:00:00Z"},
			{"id":"newer","tool":"claude","project":"b","status":"active","started_at":"2026-03-29T10:00:00Z"}
		]
	}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	sessions, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load file: %v", err)
	}

	if got := sessions[0].ID; got != "newer" {
		t.Fatalf("expected newest session first, got %q", got)
	}
}

func TestLoadDefaultSessionsLoadsFromWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	content := `{
		"sessions": [
			{"id":"fallback","tool":"claude","project":"demo","status":"completed","started_at":"2026-03-29T12:00:00Z"}
		]
	}`
	path := filepath.Join(dir, "sessions.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write sessions file: %v", err)
	}

	sessions, err := LoadDefaultSessions()
	if err != nil {
		t.Fatalf("load default sessions: %v", err)
	}

	if len(sessions) != 1 || sessions[0].ID != "fallback" {
		t.Fatalf("expected fallback session, got %#v", sessions)
	}
}

func TestLoadDefaultSessionsErrorsWhenNoFilesExist(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	_, err = LoadDefaultSessions()
	if err == nil {
		t.Fatalf("expected error when no session files exist")
	}
}
