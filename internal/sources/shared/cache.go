package shared

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "modernc.org/sqlite"

	"github.com/adinhodovic/ai-dash/internal/session"
)

// cacheVersion is the stored parser output, not just the struct shape.
// Cached files are not reread unless their size or mtime changes, so bump this when a parser starts producing different sessions; OpenCache drops the table.
const cacheVersion = 2

// Cache is a derived index of parsed sessions. Transcript bodies stay in the
// files Claude, pi, and Codex already write. The database is disposable:
// delete it and the next scan rebuilds it.
type Cache struct {
	db *sql.DB
}

type cacheRow struct {
	path    string
	size    int64
	mtime   int64
	session session.Session
}

// DefaultCachePath is the disposable index of parsed sessions.
// Linux: ~/.cache/ai-dash/sessions.db. macOS: ~/Library/Caches/ai-dash/sessions.db.
func DefaultCachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		return "sessions.db"
	}
	return filepath.Join(dir, "ai-dash", "sessions.db")
}

// OpenCache opens the session index, creating it if needed.
// One connection: SQLite state is per connection, and this process has a single poller.
// WAL plus synchronous=NORMAL is the right mode for a local cache.
// busy_timeout and immediate transactions keep a second ai-dash from failing the write.
func OpenCache(path string) (*Cache, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := path + "?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=on" +
		"&_busy_timeout=5000&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	c := &Cache{db: db}
	if err := c.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return c, nil
}

func (c *Cache) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

func (c *Cache) migrate() error {
	var version int
	if err := c.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version != cacheVersion {
		if _, err := c.db.Exec("DROP TABLE IF EXISTS transcripts"); err != nil {
			return err
		}
	}
	_, err := c.db.Exec(`
		CREATE TABLE IF NOT EXISTS transcripts (
			path TEXT PRIMARY KEY,
			tool TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			mtime_ns INTEGER NOT NULL,
			session_json TEXT NOT NULL
		) STRICT
	`)
	if err != nil {
		return err
	}
	_, err = c.db.Exec("PRAGMA user_version = " + strconv.Itoa(cacheVersion))
	return err
}

// Sessions returns one session per file. Unchanged files are served from the
// cache. parse runs only on a miss, or for every file when c is nil. refresh
// runs only on a hit, so status that depends on the clock can move without
// rereading the transcript. A cache that cannot be read falls back to parsing.
func (c *Cache) Sessions(
	tool string,
	files []TranscriptFile,
	parse func(TranscriptFile) (session.Session, bool),
	refreshStatus func(*session.Session, time.Time),
) []session.Session {
	var cached map[string]cacheRow
	if c != nil {
		cached, _ = c.load(tool)
	}

	out := make([]session.Session, 0, len(files))
	var upserts []cacheRow
	for _, f := range files {
		mtime := f.ModTime.UnixNano()
		if hit, ok := cached[f.Path]; ok && hit.size == f.Size && hit.mtime == mtime {
			s := hit.session
			if refreshStatus != nil {
				refreshStatus(&s, f.ModTime)
			}
			out = append(out, s)
			delete(cached, f.Path)
			continue
		}
		s, ok := parse(f)
		if !ok {
			continue
		}
		out = append(out, s)
		upserts = append(upserts, cacheRow{
			path:    f.Path,
			size:    f.Size,
			mtime:   mtime,
			session: s,
		})
		delete(cached, f.Path)
	}
	if c == nil {
		return out
	}
	deleted := make([]string, 0, len(cached))
	for path := range cached {
		deleted = append(deleted, path)
	}
	_ = c.save(tool, upserts, deleted)
	return out
}

func (c *Cache) load(tool string) (map[string]cacheRow, error) {
	rows, err := c.db.Query(
		`SELECT path, size_bytes, mtime_ns, session_json FROM transcripts WHERE tool = ?`,
		tool,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[string]cacheRow{}
	for rows.Next() {
		var raw string
		var item cacheRow
		if err := rows.Scan(&item.path, &item.size, &item.mtime, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &item.session); err != nil {
			continue
		}
		out[item.path] = item
	}
	return out, rows.Err()
}

func (c *Cache) save(tool string, upserts []cacheRow, deleted []string) error {
	if len(upserts) == 0 && len(deleted) == 0 {
		return nil
	}
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, item := range upserts {
		raw, err := json.Marshal(item.session)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO transcripts (path, tool, size_bytes, mtime_ns, session_json)
			 VALUES (?, ?, ?, ?, ?)`,
			item.path, tool, item.size, item.mtime, string(raw),
		); err != nil {
			return err
		}
	}
	for _, path := range deleted {
		if _, err := tx.Exec(
			`DELETE FROM transcripts WHERE tool = ? AND path = ?`,
			tool, path,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
