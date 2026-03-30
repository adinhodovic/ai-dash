package opencode

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func createTestDB(t *testing.T) string {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "opencode.db")
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE session (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		parent_id TEXT,
		slug TEXT NOT NULL,
		directory TEXT NOT NULL,
		title TEXT NOT NULL,
		version TEXT NOT NULL,
		share_url TEXT,
		summary_additions INTEGER,
		summary_deletions INTEGER,
		summary_files INTEGER,
		summary_diffs TEXT,
		revert TEXT,
		permission TEXT,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		time_compacting INTEGER,
		time_archived INTEGER,
		workspace_id TEXT
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	now := time.Now().UnixMilli()
	_, err = db.Exec(`INSERT INTO session (id, project_id, slug, directory, title, version, time_created, time_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"ses_test123", "proj1", "cool-slug", "/home/user/myproject", "Fix the bug", "1.0.0", now-60000, now)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	return dbFile
}

func TestLoadFromDB(t *testing.T) {
	dbFile := createTestDB(t)
	sessions := loadFromDB(dbFile)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	s := sessions[0]
	if s.ID != "ses_test123" {
		t.Errorf("id = %q", s.ID)
	}
	if s.Tool != "opencode" {
		t.Errorf("tool = %q", s.Tool)
	}
	if s.Project != "/home/user/myproject" {
		t.Errorf("project = %q", s.Project)
	}
	if s.Summary != "Fix the bug" {
		t.Errorf("summary = %q", s.Summary)
	}
	if s.StartedAt.IsZero() || s.EndedAt.IsZero() {
		t.Error("timestamps should be set")
	}
}

func TestDiscoverUsesEnvOverride(t *testing.T) {
	dbFile := createTestDB(t)
	t.Setenv("AIDASH_OPENCODE_DB", dbFile)

	result, err := New().Discover()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(result.Sources) == 0 {
		t.Fatal("expected sources")
	}
	if result.Sources[0].Path != dbFile {
		t.Errorf("source path = %q, want %q", result.Sources[0].Path, dbFile)
	}
	if len(result.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(result.Sessions))
	}
}

func TestLoadFromDBMissing(t *testing.T) {
	sessions := loadFromDB("/nonexistent/path.db")
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions for missing db, got %d", len(sessions))
	}
}

func TestResumeArgs(t *testing.T) {
	args := New().ResumeArgs("ses_abc")
	if len(args) != 3 || args[0] != "opencode" || args[1] != "-s" || args[2] != "ses_abc" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestDbPathEnvOverride(t *testing.T) {
	t.Setenv("AIDASH_OPENCODE_DB", "/custom/path.db")
	if got := dbPath(); got != "/custom/path.db" {
		t.Errorf("dbPath() = %q", got)
	}
}

func TestDbPathDefault(t *testing.T) {
	t.Setenv("AIDASH_OPENCODE_DB", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/testdata")
	got := dbPath()
	want := "/tmp/testdata/opencode/opencode.db"
	if got != want {
		t.Errorf("dbPath() = %q, want %q", got, want)
	}
}

func TestDbPathDefaultHome(t *testing.T) {
	t.Setenv("AIDASH_OPENCODE_DB", "")
	t.Setenv("XDG_DATA_HOME", "")
	got := dbPath()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	if got != want {
		t.Errorf("dbPath() = %q, want %q", got, want)
	}
}
