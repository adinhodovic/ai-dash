package opencode

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources/shared"
)

type Source struct{}

func New() Source { return Source{} }

func (Source) Name() string { return "opencode" }

func (Source) Discover() (shared.Result, error) {
	dbPath := dbPath()
	return shared.Result{
		Sources: []shared.Source{
			shared.NewSource("opencode", "sqlite", dbPath, "OpenCode SQLite database"),
		},
		Sessions: loadFromDB(dbPath),
	}, nil
}

func (Source) ImportSessions(result shared.Result) ([]session.Session, error) {
	return append([]session.Session(nil), result.Sessions...), nil
}

func (Source) ResumeArgs(sessionID string) []string {
	return []string{"opencode", "-s", sessionID}
}

func dbPath() string {
	if override := os.Getenv("AIDASH_OPENCODE_DB"); override != "" {
		return override
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "opencode", "opencode.db")
}

func loadFromDB(path string) []session.Session {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return nil
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, COALESCE(parent_id,''), slug, directory, title, version,
		       time_created, time_updated
		FROM session
		ORDER BY time_updated DESC
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var sessions []session.Session
	for rows.Next() {
		var (
			id, parentID, slug, dir, title, version string
			created, updated                        int64
		)
		if err := rows.Scan(&id, &parentID, &slug, &dir, &title, &version, &created, &updated); err != nil {
			continue
		}
		startedAt := time.UnixMilli(created)
		endedAt := time.UnixMilli(updated)
		status := "completed"
		if time.Since(endedAt) < 5*time.Minute {
			status = "active"
		}

		sessions = append(sessions, session.Session{
			ID:        id,
			ParentID:  parentID,
			Slug:      slug,
			Tool:      "opencode",
			Project:   dir,
			Repo:      dir,
			Status:    status,
			StartedAt: startedAt,
			EndedAt:   endedAt,
			Model:     version,
			Summary:   title,
			Tags:      []string{"opencode"},
		})
	}
	return sessions
}
