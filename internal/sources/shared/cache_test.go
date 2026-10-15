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
	file := FileStat{Path: "/tmp/a.jsonl", Size: 10, ModTime: time.Unix(100, 0)}
	parses := 0
	parse := func(FileStat) (session.Session, bool) {
		parses++
		return session.Session{ID: "s1", Summary: "parsed", TranscriptPath: file.Path}, true
	}

	got := c.Sessions("claude", []FileStat{file}, parse, nil)
	if parses != 1 || len(got) != 1 || got[0].Summary != "parsed" {
		t.Fatalf("first scan parses=%d sessions=%#v", parses, got)
	}

	parses = 0
	got = c.Sessions("claude", []FileStat{file}, parse, func(s *session.Session, _ time.Time) {
		s.Summary = "refreshed"
	})
	if parses != 0 || len(got) != 1 || got[0].Summary != "refreshed" {
		t.Fatalf("cache hit parses=%d sessions=%#v", parses, got)
	}

	// A hit must not write the refreshed status back. The stored row is the
	// parse result; status that depends on the clock is derived on read.
	parses = 0
	got = c.Sessions("claude", []FileStat{file}, parse, nil)
	if parses != 0 || got[0].Summary != "parsed" {
		t.Fatalf("stored summary = %q, parses=%d", got[0].Summary, parses)
	}
}

func TestSessionsReparsesWhenSizeChanges(t *testing.T) {
	c := openTestCache(t)
	file := FileStat{Path: "/tmp/a.jsonl", Size: 10, ModTime: time.Unix(100, 0)}
	n := 0
	parse := func(FileStat) (session.Session, bool) {
		n++
		return session.Session{ID: "s1", Summary: "v" + string(rune('0'+n))}, true
	}
	c.Sessions("claude", []FileStat{file}, parse, nil)
	file.Size = 11
	got := c.Sessions("claude", []FileStat{file}, parse, nil)
	if n != 2 || got[0].Summary != "v2" {
		t.Fatalf("n=%d summary=%q", n, got[0].Summary)
	}
}

func TestSessionsDropsDeletedFiles(t *testing.T) {
	c := openTestCache(t)
	a := FileStat{Path: "/tmp/a.jsonl", Size: 1, ModTime: time.Unix(1, 0)}
	b := FileStat{Path: "/tmp/b.jsonl", Size: 1, ModTime: time.Unix(1, 0)}
	parse := func(f FileStat) (session.Session, bool) {
		return session.Session{ID: f.Path}, true
	}
	c.Sessions("pi", []FileStat{a, b}, parse, nil)
	got := c.Sessions("pi", []FileStat{b}, parse, nil)
	if len(got) != 1 || got[0].ID != b.Path {
		t.Fatalf("sessions = %#v", got)
	}
	// The deleted path must be parsed again if it reappears, not served stale.
	parses := 0
	parse = func(f FileStat) (session.Session, bool) {
		parses++
		return session.Session{ID: "new"}, true
	}
	got = c.Sessions("pi", []FileStat{a}, parse, nil)
	if parses != 1 || got[0].ID != "new" {
		t.Fatalf("reappeared parses=%d sessions=%#v", parses, got)
	}
}

func TestSessionsParseFailureRemovesStaleRow(t *testing.T) {
	c := openTestCache(t)
	file := FileStat{Path: "/tmp/a.jsonl", Size: 1, ModTime: time.Unix(1, 0)}
	c.Sessions("codex", []FileStat{file}, func(FileStat) (session.Session, bool) {
		return session.Session{ID: "ok"}, true
	}, nil)
	file.Size = 2
	got := c.Sessions("codex", []FileStat{file}, func(FileStat) (session.Session, bool) {
		return session.Session{}, false
	}, nil)
	if len(got) != 0 {
		t.Fatalf("expected no session, got %#v", got)
	}
	parses := 0
	got = c.Sessions("codex", []FileStat{file}, func(FileStat) (session.Session, bool) {
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
	file := FileStat{Path: "/tmp/a.jsonl", Size: 1, ModTime: time.Unix(1, 0)}
	c.Sessions("claude", []FileStat{file}, func(FileStat) (session.Session, bool) {
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
	got := c.Sessions("claude", []FileStat{file}, func(FileStat) (session.Session, bool) {
		parses++
		return session.Session{ID: "new"}, true
	}, nil)
	if parses != 1 || got[0].ID != "new" {
		t.Fatalf("parses=%d sessions=%#v", parses, got)
	}
}

func TestNilCacheParsesEveryFile(t *testing.T) {
	var c *Cache
	n := 0
	got := c.Sessions(
		"claude",
		[]FileStat{{Path: "/tmp/a.jsonl"}},
		func(FileStat) (session.Session, bool) {
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
	c, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
