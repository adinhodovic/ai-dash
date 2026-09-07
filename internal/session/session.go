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

type AttentionReason string

const (
	AttentionNone      AttentionReason = ""
	AttentionWaiting   AttentionReason = "waiting"
	AttentionMaxTokens AttentionReason = "max tokens"
	AttentionStalled   AttentionReason = "stalled"
)

// staleAfter is how long an active session can go without any observed
// activity before it's considered stalled and flagged for attention. It's
// intentionally longer than the per-source mtime heuristics used to decide
// Active vs Completed status, to avoid false positives on long tool calls.
const staleAfter = 30 * time.Minute

// staleCeiling caps how long any attention reason keeps getting flagged,
// including waiting/max-tokens. Past this, an untouched session — even one
// whose last recorded state was literally "waiting for input" — is presumed
// abandoned (crashed, forgotten terminal, moved on) rather than something to
// act on right now.
const staleCeiling = 2 * time.Hour

// Attention reports whether a session needs a human to look at it: it's
// waiting on input, has hit a token limit, or has gone quiet while still
// marked active. It's a pure function of existing fields, so it works
// uniformly across all sources — a source with no waiting/max-tokens signal
// (e.g. Codex) simply falls through to the staleness check. Every reason is
// bounded by staleCeiling: an old session doesn't stay flagged forever just
// because its last observed state happened to be "waiting".
func Attention(s Session) AttentionReason {
	if time.Since(s.EndedAt) > staleCeiling {
		return AttentionNone
	}
	switch CurrentState(strings.TrimSpace(s.CurrentState)) {
	case StateWaiting:
		return AttentionWaiting
	case StateMaxTokens:
		return AttentionMaxTokens
	}
	if s.Status == string(StatusActive) && time.Since(s.EndedAt) > staleAfter {
		return AttentionStalled
	}
	return AttentionNone
}

func NeedsAttention(s Session) bool {
	return Attention(s) != AttentionNone
}

// recentActivityWindow is how recently a session must have been touched —
// regardless of its terminal status — to still count as "active" in the
// broad, everyday sense: something you were just talking to, whether or not
// it happened to wrap up cleanly in the meantime.
const recentActivityWindow = 15 * time.Minute

// IsLive reports whether a session counts as active right now: either it's
// still genuinely running (as opposed to merely last observed mid-turn/
// mid-tool-call with no later terminating event recorded — the per-source
// Active/Completed heuristics don't apply a time check on that path, so a
// session can sit at Status==active indefinitely after being abandoned; the
// staleAfter threshold guards against that), or it simply had any activity
// at all within recentActivityWindow, whatever its final status.
func IsLive(s Session) bool {
	lastActive := s.EndedAt
	if lastActive.IsZero() {
		lastActive = s.StartedAt
	}
	if time.Since(lastActive) <= recentActivityWindow {
		return true
	}
	return s.Status == string(StatusActive) && time.Since(s.EndedAt) <= staleAfter
}

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
