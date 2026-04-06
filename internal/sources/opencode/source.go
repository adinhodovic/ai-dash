package opencode

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

func (Source) ResumeArgs(sessionID, projectDir string) []string {
	q := shared.ShellQuote
	if projectDir != "" {
		return []string{"cd", q(projectDir), "&&", "opencode", "-s", q(sessionID)}
	}
	return []string{"opencode", "-s", q(sessionID)}
}

func (Source) NewSessionArgs(projectDir string) []string {
	q := shared.ShellQuote
	return []string{"cd", q(projectDir), "&&", "opencode"}
}

func (s Source) dbPath() string {
	if s.pathOverride != "" {
		return s.pathOverride
	}
	if dataDir := os.Getenv("XDG_DATA_HOME"); dataDir != "" {
		return filepath.Join(dataDir, "opencode", "opencode.db")
	}
	paths := defaultDBPaths()
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func defaultDBPaths() []string {
	home, _ := os.UserHomeDir()
	var paths []string
	if home != "" {
		if runtime.GOOS == "darwin" {
			paths = append(paths,
				filepath.Join(home, "Library", "Application Support", "opencode", "opencode.db"),
				filepath.Join(home, ".local", "share", "opencode", "opencode.db"),
			)
		} else {
			paths = append(paths,
				filepath.Join(home, ".local", "share", "opencode", "opencode.db"),
				filepath.Join(home, "Library", "Application Support", "opencode", "opencode.db"),
			)
		}
	}
	return paths
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
		       ), ''),
		       COALESCE((
		           SELECT json_extract(m.data, '$.role')
		           FROM message m WHERE m.session_id = s.id
		           ORDER BY m.time_created DESC LIMIT 1
		       ), ''),
		       COALESCE((
		           SELECT json_extract(m.data, '$.finish')
		           FROM message m WHERE m.session_id = s.id
		           ORDER BY m.time_created DESC LIMIT 1
		       ), ''),
		       COALESCE((
		           SELECT json_extract(m.data, '$.time.completed')
		           FROM message m WHERE m.session_id = s.id
		           ORDER BY m.time_created DESC LIMIT 1
		       ), 0),
		       COALESCE((
		           SELECT json_extract(m.data, '$.error.name')
		           FROM message m WHERE m.session_id = s.id
		           ORDER BY m.time_created DESC LIMIT 1
		       ), ''),
		       COALESCE((
		           SELECT json_extract(m.data, '$.finish')
		           FROM message m
		           WHERE m.session_id = s.id
		             AND json_extract(m.data, '$.role') = 'assistant'
		             AND COALESCE(json_extract(m.data, '$.time.completed'), 0) != 0
		           ORDER BY m.time_created DESC LIMIT 1
		       ), ''),
		       COALESCE((
		           SELECT json_extract(p.data, '$.type')
		           FROM part p
		           WHERE p.message_id = (
		               SELECT m.id
		               FROM message m WHERE m.session_id = s.id
		               ORDER BY m.time_created DESC LIMIT 1
		           )
		           ORDER BY p.time_created DESC LIMIT 1
		       ), ''),
		       COALESCE((
		           SELECT json_extract(p.data, '$.state.status')
		           FROM part p
		           WHERE p.message_id = (
		               SELECT m.id
		               FROM message m WHERE m.session_id = s.id
		               ORDER BY m.time_created DESC LIMIT 1
		           )
		           ORDER BY p.time_created DESC LIMIT 1
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
			latestRole, latestFinish, latestError            string
			lastCompletedFinish                              string
			latestPartType, latestPartStatus                 string
			additions, deletions, files                      int
			created, updated, latestCompleted                int64
		)
		if err := rows.Scan(
			&id, &parentID, &slug, &dir, &title, &version,
			&additions, &deletions, &files,
			&created, &updated, &modelID,
			&latestRole, &latestFinish, &latestCompleted, &latestError, &lastCompletedFinish,
			&latestPartType, &latestPartStatus,
		); err != nil {
			continue
		}
		startedAt := time.UnixMilli(created)
		endedAt := time.UnixMilli(updated)
		status, currentState := openCodeStatus(
			endedAt,
			latestRole,
			latestFinish,
			latestCompleted,
			latestError,
			lastCompletedFinish,
			latestPartType,
			latestPartStatus,
		)

		model := modelID
		if model == "" {
			model = version
		}

		meta := map[string]string{
			"app_version": version,
		}
		if currentState != "" {
			meta["current_state_source"] = openCodeStatusSource(
				latestRole,
				latestFinish,
				latestCompleted,
				latestError,
				lastCompletedFinish,
				latestPartType,
				latestPartStatus,
			)
		}
		if additions+deletions > 0 {
			meta["changes"] = fmt.Sprintf("+%d -%d in %d files", additions, deletions, files)
		}

		sessions = append(sessions, session.Session{
			ID:           id,
			ParentID:     parentID,
			Slug:         slug,
			Tool:         "opencode",
			Project:      dir,
			Repo:         dir,
			Status:       status,
			CurrentState: currentState,
			StartedAt:    startedAt,
			EndedAt:      endedAt,
			Model:        model,
			Summary:      title,
			Tags:         []string{"opencode"},
			Meta:         meta,
		})
	}
	return sessions
}

func openCodeStatus(
	endedAt time.Time,
	latestRole, latestFinish string,
	latestCompleted int64,
	latestError string,
	lastCompletedFinish string,
	latestPartType, latestPartStatus string,
) (string, string) {
	if latestError == "MessageAbortedError" {
		return string(session.StatusAborted), string(session.StateAborted)
	}
	if latestRole == "assistant" && latestCompleted == 0 {
		if latestPartType == "tool" && latestPartStatus == "running" {
			return string(session.StatusActive), string(session.StateToolCall)
		}
		switch lastCompletedFinish {
		case "tool-calls":
			return string(session.StatusActive), string(session.StateToolCall)
		case "stop":
			return string(session.StatusActive), string(session.StateWaiting)
		}
		return string(session.StatusActive), string(session.StateRunning)
	}

	switch latestFinish {
	case "tool-calls":
		return string(session.StatusActive), string(session.StateToolCall)
	case "stop":
		return string(session.StatusActive), string(session.StateWaiting)
	}

	if time.Since(endedAt) < 5*time.Minute {
		return string(session.StatusActive), string(session.StateRunning)
	}
	return string(session.StatusCompleted), string(session.StateDone)
}

func openCodeStatusSource(
	latestRole, latestFinish string,
	latestCompleted int64,
	latestError string,
	lastCompletedFinish string,
	latestPartType, latestPartStatus string,
) string {
	if latestError == "MessageAbortedError" {
		return "message.error.name=MessageAbortedError"
	}
	if latestRole == "assistant" && latestCompleted == 0 {
		if latestPartType == "tool" && latestPartStatus == "running" {
			return "part.type=tool + part.state.status=running"
		}
		switch lastCompletedFinish {
		case "tool-calls":
			return "message.pending + previous finish=tool-calls"
		case "stop":
			return "message.pending + previous finish=stop"
		default:
			return "message.pending"
		}
	}
	if latestFinish != "" {
		return fmt.Sprintf("message.finish=%s", latestFinish)
	}
	return "session.time_updated heuristic"
}
