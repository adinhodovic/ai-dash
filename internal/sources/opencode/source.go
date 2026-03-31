package opencode

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/adin/ai-dash/internal/config"
	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources/shared"
)

type Source struct {
	pathOverride string
}

var _ shared.SessionProvider = Source{}

func New(cfg config.Config) Source {
	return Source{pathOverride: cfg.SourcePath("opencode", "")}
}

func (Source) Name() string { return "opencode" }

func (s Source) Discover() (shared.Result, error) {
	dbPath := s.dbPath()
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

func (Source) ResumeArgs(sessionID, projectDir string) []string {
	cmd := "opencode -s " + sessionID
	if projectDir != "" {
		return []string{"sh", "-c", "cd " + projectDir + " && " + cmd}
	}
	return []string{cmd}
}

func (Source) NewSessionArgs(projectDir string) []string {
	return []string{"sh", "-c", "cd " + projectDir + " && opencode"}
}

func (s Source) dbPath() string {
	if s.pathOverride != "" {
		return s.pathOverride
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
	defer func() { _ = db.Close() }()

	rows, err := db.Query(`
		SELECT s.id, COALESCE(s.parent_id,''), s.slug, s.directory, s.title, s.version,
		       COALESCE(s.summary_additions,0), COALESCE(s.summary_deletions,0), COALESCE(s.summary_files,0),
		       s.time_created, s.time_updated,
		       COALESCE((
		           SELECT json_extract(m.data, '$.model.modelID')
		           FROM message m WHERE m.session_id = s.id
		           ORDER BY m.time_created ASC LIMIT 1
		       ), '')
		FROM session s
		ORDER BY s.time_updated DESC
	`)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	var sessions []session.Session
	for rows.Next() {
		var (
			id, parentID, slug, dir, title, version, modelID string
			additions, deletions, files                      int
			created, updated                                 int64
		)
		if err := rows.Scan(
			&id, &parentID, &slug, &dir, &title, &version,
			&additions, &deletions, &files,
			&created, &updated, &modelID,
		); err != nil {
			continue
		}
		startedAt := time.UnixMilli(created)
		endedAt := time.UnixMilli(updated)
		status := "completed"
		if time.Since(endedAt) < 5*time.Minute {
			status = "active"
		}

		model := modelID
		if model == "" {
			model = version
		}

		meta := map[string]string{
			"app_version": version,
		}
		if additions+deletions > 0 {
			meta["changes"] = fmt.Sprintf("+%d -%d in %d files", additions, deletions, files)
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
			Model:     model,
			Summary:   title,
			Tags:      []string{"opencode"},
			Meta:      meta,
		})
	}
	return sessions
}
