package opencode

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/adin/ai-dash/internal/config"
	_ "modernc.org/sqlite"
)

func createTestDB(t *testing.T) string {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "opencode.db")
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

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

	_, err = db.Exec(`CREATE TABLE message (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		data TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create message table: %v", err)
	}

	now := time.Now().UnixMilli()
	_, err = db.Exec(
		`INSERT INTO session (id, project_id, slug, directory, title, version, time_created, time_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"ses_test123",
		"proj1",
		"cool-slug",
		"/home/user/myproject",
		"Fix the bug",
		"1.0.0",
		now-60000,
		now,
	)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO message (id, session_id, time_created, time_updated, data)
		VALUES (?, ?, ?, ?, ?)`,
		"msg_1",
		"ses_test123",
		now-60000,
		now,
		`{"role":"user","model":{"providerID":"anthropic","modelID":"claude-sonnet-4-6"}}`,
	)
	if err != nil {
		t.Fatalf("insert message: %v", err)
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
	if s.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want claude-sonnet-4-6", s.Model)
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

	result, err := New(config.Config{OpencodePath: dbFile}).Discover()
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
	args := New(config.Config{}).ResumeArgs("ses_abc", "/home/user/project")
	// cd /home/user/project && opencode -s ses_abc
	if len(args) != 6 || args[0] != "cd" || args[3] != "opencode" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestResumeArgsNoDir(t *testing.T) {
	args := New(config.Config{}).ResumeArgs("ses_abc", "")
	// opencode -s ses_abc
	if len(args) != 3 || args[0] != "opencode" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestDbPathConfigOverride(t *testing.T) {
	s := New(config.Config{OpencodePath: "/custom/path.db"})
	if got := s.dbPath(); got != "/custom/path.db" {
		t.Errorf("dbPath() = %q, want /custom/path.db", got)
	}
}

func TestDbPathDefault(t *testing.T) {
	t.Setenv("AIDASH_OPENCODE_DB", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/testdata")
	got := (Source{}).dbPath()
	want := "/tmp/testdata/opencode/opencode.db"
	if got != want {
		t.Errorf("dbPath() = %q, want %q", got, want)
	}
}

func TestDbPathDefaultHome(t *testing.T) {
	t.Setenv("AIDASH_OPENCODE_DB", "")
	t.Setenv("XDG_DATA_HOME", "")
	got := (Source{}).dbPath()
	home, _ := os.UserHomeDir()
	var want string
	if runtime.GOOS == "darwin" {
		want = filepath.Join(home, "Library", "Application Support", "opencode", "opencode.db")
	} else {
		want = filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	}
	if got != want {
		t.Errorf("dbPath() = %q, want %q", got, want)
	}
}

func TestDbPathPrefersExistingMacFallbackOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only path preference")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")

	linuxStyle := filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	if err := os.MkdirAll(filepath.Dir(linuxStyle), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(linuxStyle, []byte("db"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if got := (Source{}).dbPath(); got != linuxStyle {
		t.Fatalf("dbPath() = %q, want %q", got, linuxStyle)
	}
}

func TestDefaultDBPathsReturnsTwo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	paths := defaultDBPaths()
	if len(paths) != 2 {
		t.Fatalf("expected 2 candidate paths, got %d: %v", len(paths), paths)
	}
	if paths[0] == paths[1] {
		t.Fatalf("paths should differ, got %q twice", paths[0])
	}
}
