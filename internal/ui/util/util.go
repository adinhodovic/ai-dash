package util

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/samber/lo"

	"github.com/adin/ai-dash/internal/session"
)

func HumanizeKey(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	return Capitalize(s)
}

func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func Truncate(value string, limit int) string {
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "~"
}

func TruncateForCell(value string, width int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return Truncate(value, width)
}

var homeDir, _ = os.UserHomeDir()

func ShortenPath(value string) string {
	if homeDir != "" {
		value = strings.ReplaceAll(value, homeDir, "~")
	}
	return value
}

func CleanProjectName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") {
		return ShortenPath(value)
	}
	return value
}

func CleanSummary(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	if looksLikeRequestID(value) || looksLikeUUID(value) {
		return "Imported session"
	}
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.Join(strings.Fields(value), " ")
}

func looksLikeRequestID(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "req_") || strings.HasPrefix(value, "ephemeral_") ||
		strings.HasPrefix(value, "cache_creation_")
}

func looksLikeUUID(value string) bool {
	parts := strings.Split(value, "-")
	return len(parts) == 5 && !lo.Contains(parts, "")
}

func LastActive(s session.Session) time.Time {
	if !s.EndedAt.IsZero() {
		return s.EndedAt
	}
	return s.StartedAt
}

func TimeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	if time.Since(t) < time.Minute {
		return "< 1m ago"
	}
	return humanize.Time(t)
}

func DurationLabel(s session.Session) string {
	if s.Status == "active" {
		return "running"
	}
	if s.EndedAt.IsZero() || s.StartedAt.IsZero() {
		return "unknown"
	}
	d := s.EndedAt.Sub(s.StartedAt)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func FormatTokens(in, out int) string {
	if in == 0 && out == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%s in / %s out", formatCount(in), formatCount(out))
}

func FormatCost(cost float64) string {
	if cost == 0 {
		return "n/a"
	}
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}

func formatCount(n int) string {
	return humanize.Comma(int64(n))
}

func ValueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func RelationLabel(selected, candidate session.Session) string {
	switch {
	case selected.ParentID != "" && candidate.ID == selected.ParentID:
		return "parent"
	case candidate.ParentID != "" && candidate.ParentID == selected.ID:
		return "child"
	case selected.Project != "" && candidate.Project == selected.Project:
		return "project"
	case selected.Repo != "" && candidate.Repo == selected.Repo:
		return "repo"
	default:
		return ""
	}
}
