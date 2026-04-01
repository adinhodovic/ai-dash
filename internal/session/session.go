package session

import (
	"sort"
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
	SortSummary SortField = "summary"
)

func Sort(sessions []Session) {
	SortBy(sessions, SortStarted, true)
}

func SortBy(sessions []Session, field SortField, descending bool) {
	sort.Slice(sessions, func(i, j int) bool {
		left, right := sessions[i], sessions[j]
		if descending {
			return compareSessions(right, left, field)
		}
		return compareSessions(left, right, field)
	})
}

func compareSessions(left, right Session, field SortField) bool {
	switch field {
	case SortUpdated:
		return left.EndedAt.Before(right.EndedAt)
	case SortProject:
		if left.Project == right.Project {
			return left.StartedAt.Before(right.StartedAt)
		}
		return left.Project < right.Project
	case SortTool:
		if left.Tool == right.Tool {
			return left.StartedAt.Before(right.StartedAt)
		}
		return left.Tool < right.Tool
	case SortSummary:
		if left.Summary == right.Summary {
			return left.StartedAt.Before(right.StartedAt)
		}
		return left.Summary < right.Summary
	default:
		return left.StartedAt.Before(right.StartedAt)
	}
}

func EndedLabel(end time.Time, status string) string {
	if status == "active" || end.IsZero() {
		return "still running"
	}
	return end.Format(time.RFC1123)
}
