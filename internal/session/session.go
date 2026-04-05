package session

import (
	"cmp"
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

type SortField string

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
	if status == "active" || end.IsZero() {
		return "still running"
	}
	return end.Format(time.RFC1123)
}

func StatusLabel(s Session) string {
	if currentState := strings.TrimSpace(s.CurrentState); currentState != "" {
		return currentState
	}

	if s.Status == "active" {
		switch strings.TrimSpace(s.Meta["stop_reason"]) {
		case "end_turn", "pause_turn":
			return "waiting"
		case "tool_use":
			return "tool call"
		case "max_tokens":
			return "max tokens"
		default:
			return "running"
		}
	}

	switch s.Status {
	case "completed":
		return "done"
	case "aborted":
		return "aborted"
	default:
		if status := strings.TrimSpace(s.Status); status != "" {
			return status
		}
		return "unknown"
	}
}
