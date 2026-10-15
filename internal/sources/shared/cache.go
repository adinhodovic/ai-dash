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

// cacheVersion is the stored session shape. Bump it when a parser starts
// emitting fields old rows do not have; Open drops the table.
const cacheVersion = 1

// FileStat is the identity of a transcript on disk. Size and mtime together
// catch appends even when the filesystem timestamp is coarse.
type FileStat struct {
	Path    string
	Size    int64
	ModTime time.Time
}

// Cache is a derived index of parsed sessions. Transcript bodies stay in the
// files Claude, pi, and Codex already write. The database is disposable:
// delete it and the next scan rebuilds it.
type Cache struct {
	db *sql.DB
}

type cachedRow struct {
	size    int64
	mtime   int64
	session session.Session
}

// DefaultPath is ~/.config/ai-dash/cache.db, next to the other ai-dash files.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return "cache.db"
	}
	return filepath.Join(dir, "ai-dash", "cache.db")
}

// Open opens the cache, creating it if needed. One connection: SQLite state
// is per connection, and this process has a single poller. WAL plus
// synchronous=NORMAL is the right mode for a local cache. busy_timeout and
// immediate transactions keep a second ai-dash from failing the write.
func Open(path string) (*Cache, error) {
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
		if _, err := c.db.Exec("DROP TABLE IF EXISTS file_cache"); err != nil {
			return err
		}
	}
	_, err := c.db.Exec(`
		CREATE TABLE IF NOT EXISTS file_cache (
			path TEXT PRIMARY KEY,
			tool TEXT NOT NULL,
			size INTEGER NOT NULL,
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
// rereading the transcript. Cache errors fall back to parsing.
func (c *Cache) Sessions(
	tool string,
	files []FileStat,
	parse func(FileStat) (session.Session, bool),
	refresh func(*session.Session, time.Time),
) []session.Session {
	if c == nil {
		return parseAll(files, parse)
	}
	cached, err := c.load(tool)
	if err != nil {
		return parseAll(files, parse)
	}

	out := make([]session.Session, 0, len(files))
	var upserts []stored
	seen := make(map[string]struct{}, len(files))
	for _, f := range files {
		if f.Path == "" {
			continue
		}
		if _, ok := seen[f.Path]; ok {
			continue
		}
		seen[f.Path] = struct{}{}

		mtime := f.ModTime.UnixNano()
		if row, ok := cached[f.Path]; ok && row.size == f.Size && row.mtime == mtime {
			s := row.session
			if refresh != nil {
				refresh(&s, f.ModTime)
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
		upserts = append(upserts, stored{
			path:    f.Path,
			size:    f.Size,
			mtime:   mtime,
			session: s,
		})
		delete(cached, f.Path)
	}

	deleted := make([]string, 0, len(cached))
	for path := range cached {
		deleted = append(deleted, path)
	}
	_ = c.save(tool, upserts, deleted)
	return out
}

type stored struct {
	path    string
	size    int64
	mtime   int64
	session session.Session
}

func parseAll(files []FileStat, parse func(FileStat) (session.Session, bool)) []session.Session {
	out := make([]session.Session, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, f := range files {
		if _, ok := seen[f.Path]; ok {
			continue
		}
		seen[f.Path] = struct{}{}
		if s, ok := parse(f); ok {
			out = append(out, s)
		}
	}
	return out
}

func (c *Cache) load(tool string) (map[string]cachedRow, error) {
	rows, err := c.db.Query(
		`SELECT path, size, mtime_ns, session_json FROM file_cache WHERE tool = ?`,
		tool,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[string]cachedRow{}
	for rows.Next() {
		var path, raw string
		var row cachedRow
		if err := rows.Scan(&path, &row.size, &row.mtime, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &row.session); err != nil {
			continue
		}
		out[path] = row
	}
	return out, rows.Err()
}

func (c *Cache) save(tool string, upserts []stored, deleted []string) error {
	if len(upserts) == 0 && len(deleted) == 0 {
		return nil
	}
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	insert, err := tx.Prepare(`
		INSERT INTO file_cache (path, tool, size, mtime_ns, session_json)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			tool = excluded.tool,
			size = excluded.size,
			mtime_ns = excluded.mtime_ns,
			session_json = excluded.session_json
	`)
	if err != nil {
		return err
	}
	defer func() { _ = insert.Close() }()

	for _, row := range upserts {
		raw, err := json.Marshal(row.session)
		if err != nil {
			return err
		}
		if _, err := insert.Exec(row.path, tool, row.size, row.mtime, string(raw)); err != nil {
			return err
		}
	}
	for _, path := range deleted {
		if _, err := tx.Exec(
			`DELETE FROM file_cache WHERE tool = ? AND path = ?`,
			tool,
			path,
		); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}
