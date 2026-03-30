package claude

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources/shared"
)

type Source struct{}

func New() Source { return Source{} }

func (Source) Name() string { return "claude" }

func (Source) Discover() (shared.Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return shared.Result{}, err
	}

	statePath := shared.EnvOrDefault("AIDASH_CLAUDE_STATE", filepath.Join(home, ".claude.json"))
	settingsPath := shared.EnvOrDefault("AIDASH_CLAUDE_SETTINGS", filepath.Join(home, ".claude/settings.json"))
	projectsPath := shared.EnvOrDefault("AIDASH_CLAUDE_PROJECTS_DIR", filepath.Join(home, ".claude/projects"))

	result := shared.Result{
		Sources: []shared.Source{
			shared.NewSource("claude", "json", statePath, "Claude Code global state"),
			shared.NewSource("claude", "json", settingsPath, "Claude Code user settings"),
			shared.NewSource("claude", "jsonl", projectsPath, "Claude Code transcripts"),
		},
	}

	transcripts, err := discoverTranscripts(projectsPath)
	result.Transcripts = transcripts
	result.Sessions = importTranscriptSessions(transcripts)
	return result, err
}

func (Source) ImportSessions(result shared.Result) ([]session.Session, error) {
	if len(result.Sessions) > 0 {
		return append([]session.Session(nil), result.Sessions...), nil
	}
	return importTranscriptSessions(result.Transcripts), nil
}

func (Source) ResumeArgs(sessionID string) []string {
	return []string{"claude", "--resume", sessionID}
}

func discoverTranscripts(root string) ([]shared.TranscriptFile, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	var transcripts []shared.TranscriptFile
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		transcripts = append(transcripts, shared.TranscriptFile{
			Tool:    "claude",
			Path:    path,
			Project: projectNameFromPath(root, path),
			ModTime: info.ModTime(),
		})
		return nil
	})
	return transcripts, err
}

func projectNameFromPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) == 0 {
		return ""
	}
	return sanitizeProjectName(parts[0])
}

func sanitizeProjectName(value string) string {
	original := strings.Trim(value, "-")
	if original == "" {
		return "unknown"
	}
	if strings.HasPrefix(value, "-") || strings.Contains(original, "-home-") || strings.Contains(original, "-workspace-") {
		parts := strings.Split(original, "-")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if last != "" {
				return last
			}
		}
	}
	return original
}

func importTranscriptSessions(transcripts []shared.TranscriptFile) []session.Session {
	parsed := shared.ImportGenericSessions("claude", transcriptPaths(transcripts))
	if len(parsed) > 0 {
		return parsed
	}
	sessions := make([]session.Session, 0, len(transcripts))
	for i, transcript := range transcripts {
		project := transcript.Project
		if project == "" {
			project = "unknown"
		}
		sessions = append(sessions, session.Session{
			ID:             fmt.Sprintf("%s-%03d", transcript.Tool, i+1),
			Tool:           transcript.Tool,
			Project:        project,
			Status:         "discovered",
			StartedAt:      transcript.ModTime,
			EndedAt:        transcript.ModTime,
			Summary:        "Auto-discovered transcript file from documented local storage.",
			TranscriptPath: transcript.Path,
			Tags:           []string{"auto-discovered"},
		})
	}
	return sessions
}

func transcriptPaths(transcripts []shared.TranscriptFile) []string {
	paths := make([]string, 0, len(transcripts))
	for _, transcript := range transcripts {
		paths = append(paths, transcript.Path)
	}
	return paths
}
