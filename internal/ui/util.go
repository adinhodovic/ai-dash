package ui

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/dustin/go-humanize"
	"github.com/samber/lo"

	"github.com/adin/ai-dash/internal/config"
	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources"
)

func humanizeKey(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	return capitalize(s)
}

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

// shortenPath replaces the user's home directory with ~ in any path string.
func shortenPath(value string) string {
	if homeDir != "" {
		value = strings.ReplaceAll(value, homeDir, "~")
	}
	return value
}

func cleanProjectName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	// Raw absolute paths or ~ paths — just shorten
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") {
		return shortenPath(value)
	}
	return value
}

func cleanSummary(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	if looksLikeRequestID(value) || looksLikeUUID(value) {
		return "Imported session"
	}
	// Collapse to single line — multi-line summaries break table row height.
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
	return append(
		[]string{""},
		uniqueSortedValues(sessions, func(s session.Session) string { return s.Tool })...)
}

func projectOptions(sessions []session.Session) []string {
	return append(
		[]string{""},
		uniqueSortedValues(sessions, func(s session.Session) string { return s.Project })...)
}

func uniqueSortedValues(sessions []session.Session, pick func(session.Session) string) []string {
	values := lo.Uniq(lo.FilterMap(sessions, func(s session.Session, _ int) (string, bool) {
		v := pick(s)
		return v, v != ""
	}))
	slices.Sort(values)
	return values
}

// renderPane renders a bordered panel with a title and body, constrained to the given dimensions.
// height is the total pane height including borders. Width is total including borders.
func renderPane(border, header lipgloss.Style, title, body string, width, height int) string {
	titleLine := header.PaddingRight(1).PaddingLeft(1).MarginBottom(1).Render(title)
	content := lipgloss.JoinVertical(lipgloss.Left, titleLine, body)
	return border.
		Width(width).
		Height(height).
		MaxHeight(height).
		Render(content)
}

// panelStyle returns the active or inactive border style for a pane.
func panelStyle(s styles, active bool) lipgloss.Style {
	if active {
		return s.active
	}
	return s.panel
}

func nextSortField(current session.SortField) session.SortField {
	fields := []session.SortField{
		session.SortUpdated,
		session.SortTool,
		session.SortProject,
		session.SortSummary,
	}
	for i, field := range fields {
		if field == current {
			return fields[(i+1)%len(fields)]
		}
	}
	return session.SortUpdated
}

func prevSortField(current session.SortField) session.SortField {
	fields := []session.SortField{
		session.SortUpdated,
		session.SortTool,
		session.SortProject,
		session.SortSummary,
	}
	for i, field := range fields {
		if field == current {
			return fields[(i-1+len(fields))%len(fields)]
		}
	}
	return session.SortUpdated
}

func sessionDir(s session.Session) string {
	if s.Repo != "" && s.Repo != "/" {
		return s.Repo
	}
	if s.Project != "" && (strings.HasPrefix(s.Project, "/") || strings.HasPrefix(s.Project, "~")) {
		return s.Project
	}
	return ""
}

func sessionCommand(s session.Session, cfg config.Config) *exec.Cmd {
	args := sources.ResumeArgs(cfg, s.Tool, s.ID, sessionDir(s))
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
