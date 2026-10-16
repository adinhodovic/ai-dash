package shared

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/adinhodovic/ai-dash/internal/session"
)

func TestSessionsSkipsUnchangedFile(t *testing.T) {
	c := openTestCache(t)
	file := TranscriptFile{Path: "/tmp/a.jsonl", Size: 10, ModTime: time.Unix(100, 0)}
	parses := 0
	parse := func(TranscriptFile) (session.Session, bool) {
		parses++
		return session.Session{ID: "s1", Summary: "parsed", TranscriptPath: file.Path}, true
	}

	got := c.Sessions("claude", []TranscriptFile{file}, parse, nil)
	if parses != 1 || len(got) != 1 || got[0].Summary != "parsed" {
		t.Fatalf("first scan parses=%d sessions=%#v", parses, got)
	}

	parses = 0
	refresh := func(s *session.Session, _ time.Time) { s.Summary = "refreshed" }
	got = c.Sessions("claude", []TranscriptFile{file}, parse, refresh)
	if parses != 0 || len(got) != 1 || got[0].Summary != "refreshed" {
		t.Fatalf("cache hit parses=%d sessions=%#v", parses, got)
	}

	// A hit must not write the refreshed status back. The stored row is the
	// parse result; status that depends on the clock is derived on read.
	parses = 0
	got = c.Sessions("claude", []TranscriptFile{file}, parse, nil)
	if parses != 0 || got[0].Summary != "parsed" {
		t.Fatalf("stored summary = %q, parses=%d", got[0].Summary, parses)
	}
}

func TestSessionsReparsesWhenSizeChanges(t *testing.T) {
	c := openTestCache(t)
	file := TranscriptFile{Path: "/tmp/a.jsonl", Size: 10, ModTime: time.Unix(100, 0)}
	n := 0
	parse := func(TranscriptFile) (session.Session, bool) {
		n++
		return session.Session{ID: "s1", Summary: "v" + string(rune('0'+n))}, true
	}
	c.Sessions("claude", []TranscriptFile{file}, parse, nil)
	file.Size = 11
	got := c.Sessions("claude", []TranscriptFile{file}, parse, nil)
	if n != 2 || got[0].Summary != "v2" {
		t.Fatalf("n=%d summary=%q", n, got[0].Summary)
	}
}

func TestSessionsDropsDeletedFiles(t *testing.T) {
	c := openTestCache(t)
	a := TranscriptFile{Path: "/tmp/a.jsonl", Size: 1, ModTime: time.Unix(1, 0)}
	b := TranscriptFile{Path: "/tmp/b.jsonl", Size: 1, ModTime: time.Unix(1, 0)}
	parse := func(f TranscriptFile) (session.Session, bool) {
		return session.Session{ID: f.Path}, true
	}
	c.Sessions("pi", []TranscriptFile{a, b}, parse, nil)
	got := c.Sessions("pi", []TranscriptFile{b}, parse, nil)
	if len(got) != 1 || got[0].ID != b.Path {
		t.Fatalf("sessions = %#v", got)
	}
	// The deleted path must be parsed again if it reappears, not served stale.
	parses := 0
	parse = func(f TranscriptFile) (session.Session, bool) {
		parses++
		return session.Session{ID: "new"}, true
	}
	got = c.Sessions("pi", []TranscriptFile{a}, parse, nil)
	if parses != 1 || got[0].ID != "new" {
		t.Fatalf("reappeared parses=%d sessions=%#v", parses, got)
	}
}

func TestSessionsParseFailureRemovesStaleRow(t *testing.T) {
	c := openTestCache(t)
	file := TranscriptFile{Path: "/tmp/a.jsonl", Size: 1, ModTime: time.Unix(1, 0)}
	c.Sessions("codex", []TranscriptFile{file}, func(TranscriptFile) (session.Session, bool) {
		return session.Session{ID: "ok"}, true
	}, nil)
	file.Size = 2
	got := c.Sessions(
		"codex",
		[]TranscriptFile{file},
		func(TranscriptFile) (session.Session, bool) {
			return session.Session{}, false
		},
		nil,
	)
	if len(got) != 0 {
		t.Fatalf("expected no session, got %#v", got)
	}
	parses := 0
	got = c.Sessions("codex", []TranscriptFile{file}, func(TranscriptFile) (session.Session, bool) {
		parses++
		return session.Session{ID: "again"}, true
	}, nil)
	if parses != 1 || got[0].ID != "again" {
		t.Fatalf("parses=%d sessions=%#v", parses, got)
	}
}

func TestOpenDropsCacheWhenVersionChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.db")
	c := mustOpen(t, path)
	file := TranscriptFile{Path: "/tmp/a.jsonl", Size: 1, ModTime: time.Unix(1, 0)}
	c.Sessions("claude", []TranscriptFile{file}, func(TranscriptFile) (session.Session, bool) {
		return session.Session{ID: "old"}, true
	}, nil)
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version = 0"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	c = mustOpen(t, path)
	defer func() { _ = c.Close() }()
	parses := 0
	got := c.Sessions(
		"claude",
		[]TranscriptFile{file},
		func(TranscriptFile) (session.Session, bool) {
			parses++
			return session.Session{ID: "new"}, true
		},
		nil,
	)
	if parses != 1 || got[0].ID != "new" {
		t.Fatalf("parses=%d sessions=%#v", parses, got)
	}
}

func TestNilCacheParsesEveryFile(t *testing.T) {
	var c *Cache
	n := 0
	got := c.Sessions(
		"claude",
		[]TranscriptFile{{Path: "/tmp/a.jsonl"}},
		func(TranscriptFile) (session.Session, bool) {
			n++
			return session.Session{ID: "x"}, true
		},
		nil,
	)
	if n != 1 || len(got) != 1 {
		t.Fatalf("n=%d sessions=%#v", n, got)
	}
}

func openTestCache(t *testing.T) *Cache {
	t.Helper()
	c := mustOpen(t, filepath.Join(t.TempDir(), "cache.db"))
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func mustOpen(t *testing.T, path string) *Cache {
	t.Helper()
	c, err := OpenCache(path)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
