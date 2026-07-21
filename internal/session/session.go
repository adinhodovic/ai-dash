package session

import (
	"cmp"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type Session struct {
	ID             string            `json:"id"`
	ParentID       string            `json:"parent_id"`
	Slug           string            `json:"slug"`
	Tool           string            `json:"tool"`
	Project        string            `json:"project"`
	Repo           string            `json:"repo"`
	Branch         string            `json:"branch"`
	Status         string            `json:"status"`
	CurrentState   string            `json:"current_state,omitempty"`
	StartedAt      time.Time         `json:"started_at"`
	EndedAt        time.Time         `json:"ended_at"`
	Model          string            `json:"model"`
	Summary        string            `json:"summary"`
	TranscriptPath string            `json:"transcript_path"`
	TokensIn       int               `json:"tokens_in"`
	TokensOut      int               `json:"tokens_out"`
	CostUSD        float64           `json:"cost_usd"`
	Tags           []string          `json:"tags"`
	Meta           map[string]string `json:"meta,omitempty"`
}

const renamesFile = "session-renames.json"

type SortField string

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusAborted   Status = "aborted"
)

type CurrentState string

const (
	StateRunning   CurrentState = "running"
	StateWaiting   CurrentState = "waiting"
	StateToolCall  CurrentState = "tool call"
	StateMaxTokens CurrentState = "max tokens"
	StateDone      CurrentState = "done"
	StateAborted   CurrentState = "aborted"
	StateUnknown   CurrentState = "unknown"
)

const (
	SortStarted SortField = "started"
	SortUpdated SortField = "updated"
	SortProject SortField = "project"
	SortTool    SortField = "tool"
	SortStatus  SortField = "status"
	SortSummary SortField = "summary"
)

func Sort(sessions []Session) {
	SortBy(sessions, SortStarted, true)
}

func RenameKey(s Session) string {
	return s.Tool + "/" + s.ID
}

func DefaultRenamesPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return renamesFile
	}
	return filepath.Join(dir, "ai-dash", renamesFile)
}

func LoadRenames(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	renames := map[string]string{}
	if err := json.Unmarshal(data, &renames); err != nil {
		return nil, err
	}
	return renames, nil
}

func SaveRenames(path string, renames map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(renames, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func ApplyRenames(sessions []Session, renames map[string]string) {
	for i := range sessions {
		if summary := strings.TrimSpace(renames[RenameKey(sessions[i])]); summary != "" {
			sessions[i].Summary = summary
		}
	}
}

func SortBy(sessions []Session, field SortField, descending bool) {
	slices.SortFunc(sessions, func(a, b Session) int {
		c := compareSessions(a, b, field)
		if descending {
			return -c
		}
		return c
	})
}

func compareSessions(a, b Session, field SortField) int {
	switch field {
	case SortUpdated:
		return a.EndedAt.Compare(b.EndedAt)
	case SortProject:
		if c := cmp.Compare(a.Project, b.Project); c != 0 {
			return c
		}
		return a.StartedAt.Compare(b.StartedAt)
	case SortTool:
		if c := cmp.Compare(a.Tool, b.Tool); c != 0 {
			return c
		}
		return a.StartedAt.Compare(b.StartedAt)
	case SortStatus:
		if c := cmp.Compare(StatusLabel(a), StatusLabel(b)); c != 0 {
			return c
		}
		return a.StartedAt.Compare(b.StartedAt)
	case SortSummary:
		if c := cmp.Compare(a.Summary, b.Summary); c != 0 {
			return c
		}
		return a.StartedAt.Compare(b.StartedAt)
	default:
		return a.StartedAt.Compare(b.StartedAt)
	}
}

func EndedLabel(end time.Time, status string) string {
	if status == string(StatusActive) || end.IsZero() {
		return "still running"
	}
	return end.Format(time.RFC1123)
}

func StatusLabel(s Session) string {
	if currentState := strings.TrimSpace(s.CurrentState); currentState != "" {
		return currentState
	}

	switch strings.TrimSpace(s.Status) {
	case string(StatusActive):
		return string(StateRunning)
	case string(StatusCompleted):
		return string(StateDone)
	case string(StatusAborted):
		return string(StateAborted)
	case "":
		return string(StateUnknown)
	default:
		return strings.TrimSpace(s.Status)
	}
}
