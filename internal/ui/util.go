package ui

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/samber/lo"

	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources"
)

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "~"
}

func truncateForCell(value string, width int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return truncate(value, width)
}

var homeDir, _ = os.UserHomeDir()
var homeSlug = func() string {
	if homeDir == "" {
		return ""
	}
	return strings.ReplaceAll(homeDir, "/", "-") + "-"
}()

// shortenPath replaces the user's home directory with ~ in any path string.
func shortenPath(value string) string {
	if homeDir != "" {
		value = strings.ReplaceAll(value, homeDir, "~")
	}
	return value
}

func cleanProjectName(value string) string {
	value = strings.TrimSpace(value)
	// Handle slug-encoded paths (e.g. -home-adin-oss-foo)
	if homeSlug != "" {
		value = strings.ReplaceAll(value, homeSlug, "~/")
	}
	value = strings.ReplaceAll(value, "-workspace-", "~/")
	value = strings.Trim(value, "-")
	if value == "" {
		return "unknown"
	}
	// Convert slug separators back to path separators
	value = strings.ReplaceAll(value, "-", "/")
	// Also handle raw absolute paths
	return shortenPath(value)
}

func cleanSummary(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	if looksLikeRequestID(value) || looksLikeUUID(value) {
		return "Imported session"
	}
	return value
}

func looksLikeRequestID(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "req_") || strings.HasPrefix(value, "ephemeral_") || strings.HasPrefix(value, "cache_creation_")
}

func looksLikeUUID(value string) bool {
	parts := strings.Split(value, "-")
	return len(parts) == 5 && !lo.Contains(parts, "")
}

func lastActive(s session.Session) time.Time {
	if !s.EndedAt.IsZero() {
		return s.EndedAt
	}
	return s.StartedAt
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	if time.Since(t) < time.Minute {
		return "< 1m ago"
	}
	return humanize.Time(t)
}

func durationLabel(s session.Session) string {
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

func formatTokens(in, out int) string {
	if in == 0 && out == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%s in / %s out", formatCount(in), formatCount(out))
}

func formatCost(cost float64) string {
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

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func toolOptions(sessions []session.Session) []string {
	return append([]string{""}, uniqueSortedValues(sessions, func(s session.Session) string { return s.Tool })...)
}

func statusOptions(sessions []session.Session) []string {
	return append([]string{""}, uniqueSortedValues(sessions, func(s session.Session) string { return s.Status })...)
}

func projectOptions(sessions []session.Session) []string {
	return append([]string{""}, uniqueSortedValues(sessions, func(s session.Session) string { return s.Project })...)
}

func uniqueSortedValues(sessions []session.Session, pick func(session.Session) string) []string {
	values := lo.Uniq(lo.FilterMap(sessions, func(s session.Session, _ int) (string, bool) {
		v := pick(s)
		return v, v != ""
	}))
	slices.Sort(values)
	return values
}

// contentHeight returns the available height for panes (total minus top bar and footer).
func contentHeight(termHeight int) int {
	// JoinVertical layout: top + content + footer.
	// Each join adds one separator, so the fixed overhead is top(1) + footer(1) + separators(2) = 4.
	return max(4, termHeight-4)
}

func topPaneHeight(termHeight int) int {
	return max(4, contentHeight(termHeight)*30/100)
}

func bottomPaneHeight(termHeight int) int {
	return max(4, contentHeight(termHeight)-topPaneHeight(termHeight))
}

func paneBodyHeight(paneHeight int) int {
	return max(1, paneHeight-4)
}

func nextSortField(current session.SortField) session.SortField {
	fields := []session.SortField{session.SortStarted, session.SortUpdated, session.SortProject, session.SortTool, session.SortStatus}
	for i, field := range fields {
		if field == current {
			return fields[(i+1)%len(fields)]
		}
	}
	return session.SortStarted
}

func prevSortField(current session.SortField) session.SortField {
	fields := []session.SortField{session.SortStarted, session.SortUpdated, session.SortProject, session.SortTool, session.SortStatus}
	for i, field := range fields {
		if field == current {
			return fields[(i-1+len(fields))%len(fields)]
		}
	}
	return session.SortStarted
}

func sessionCommand(s session.Session) *exec.Cmd {
	args := sources.ResumeArgs(s.Tool, s.ID)
	if len(args) == 0 {
		return nil
	}
	return spawnTerminal(args)
}

var terminalCmd string

func spawnTerminal(args []string) *exec.Cmd {
	if terminalCmd == "" {
		return nil
	}
	shell := strings.Join(args, " ")
	return exec.Command(terminalCmd, "-e", "sh", "-c", shell)
}

func relationLabel(selected, candidate session.Session) string {
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
