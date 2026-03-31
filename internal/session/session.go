package session

import (
	"fmt"
	"sort"
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

type File struct {
	Sessions []Session `json:"sessions"`
}

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
	case SortStatus:
		if left.Status == right.Status {
			return left.StartedAt.Before(right.StartedAt)
		}
		return left.Status < right.Status
	case SortSummary:
		if left.Summary == right.Summary {
			return left.StartedAt.Before(right.StartedAt)
		}
		return left.Summary < right.Summary
	default:
		return left.StartedAt.Before(right.StartedAt)
	}
}

func SummaryLine(sessions []Session) string {
	active := 0
	toolCounts := map[string]int{}
	for _, s := range sessions {
		if s.Status == "active" {
			active++
		}
		toolCounts[s.Tool]++
	}

	parts := []string{fmt.Sprintf("%d total", len(sessions)), fmt.Sprintf("%d active", active)}
	tools := []string{"opencode", "codex", "claude"}
	for _, tool := range tools {
		if count := toolCounts[tool]; count > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", tool, count))
		}
	}

	return strings.Join(parts, "  |  ")
}

func ProjectOverview(sessions []Session) string {
	projects := map[string]int{}
	tokens := 0
	for _, s := range sessions {
		projects[s.Project]++
		tokens += s.TokensIn + s.TokensOut
	}

	keys := make([]string, 0, len(projects))
	for key := range projects {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := []string{
		fmt.Sprintf("Projects: %d", len(projects)),
		fmt.Sprintf("Total tokens: %d", tokens),
	}

	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s: %d session(s)", key, projects[key]))
	}

	return strings.Join(parts, "\n")
}

func ProjectOverviewRows(sessions []Session) [][2]string {
	projects := map[string]int{}
	tokens := 0
	for _, s := range sessions {
		projects[s.Project]++
		tokens += s.TokensIn + s.TokensOut
	}

	keys := make([]string, 0, len(projects))
	for key := range projects {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	rows := [][2]string{
		{"Projects", fmt.Sprintf("%d", len(projects))},
		{"Total tokens", fmt.Sprintf("%d", tokens)},
	}
	for _, key := range keys {
		rows = append(rows, [2]string{key, fmt.Sprintf("%d session(s)", projects[key])})
	}
	return rows
}

func TimeLabel(start time.Time, status string) string {
	if status == "active" {
		return fmt.Sprintf("started %s", start.Format("2006-01-02 15:04"))
	}
	return fmt.Sprintf("ran %s", start.Format("2006-01-02 15:04"))
}

func EndedLabel(end time.Time, status string) string {
	if status == "active" || end.IsZero() {
		return "still running"
	}
	return end.Format(time.RFC1123)
}
