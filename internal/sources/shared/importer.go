package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adin/ai-dash/internal/session"
)

func DiscoverCandidateFiles(root string) ([]string, error) {
	return DiscoverCandidateFilesWithPatterns(root, nil)
}

func DiscoverCandidateFilesWithPatterns(root string, extraPatterns []string) ([]string, error) {
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

	var matches []string
	patterns := append(
		[]string{"session", "history", "transcript", "chat", "conversation", "messages"},
		extraPatterns...)
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		base := strings.ToLower(filepath.Base(path))
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".json" && ext != ".jsonl" {
			return nil
		}
		if strings.Contains(base, "config") || strings.Contains(base, "settings") ||
			strings.Contains(base, "theme") ||
			strings.Contains(base, "keybind") {
			return nil
		}
		if ext == ".jsonl" || containsAny(base, patterns) {
			matches = append(matches, path)
		}
		return nil
	})
	sort.Strings(matches)
	return matches, err
}

func containsAny(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(value, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func ImportGenericSessions(tool string, paths []string) []session.Session {
	imported := make([]session.Session, 0, len(paths))
	for _, path := range paths {
		if s, ok := parseSessionFile(tool, path); ok {
			imported = append(imported, s)
		}
	}
	session.Sort(imported)
	return imported
}

func parseSessionFile(tool, path string) (session.Session, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return session.Session{}, false
	}

	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		payload = parseJSONLLines(data)
	}

	stringsFound := collectStrings(payload)
	if !looksLikeSession(stringsFound) {
		return session.Session{}, false
	}

	modTime := fileModTime(path)
	model := detectModel(stringsFound)
	summary := detectSummary(stringsFound)
	project := inferProject(path)
	status := detectStatus(stringsFound)
	start, end := detectTimes(stringsFound, modTime)

	return session.Session{
		ID:             fmt.Sprintf("%s-%s", tool, slug(filepath.Base(path))),
		Tool:           tool,
		Project:        project,
		Status:         status,
		StartedAt:      start,
		EndedAt:        end,
		Model:          model,
		Summary:        summary,
		TranscriptPath: path,
		Tags:           []string{"auto-imported"},
	}, true
}

func parseJSONLLines(data []byte) []any {
	lines := strings.Split(string(data), "\n")
	out := make([]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var item any
		if err := json.Unmarshal([]byte(line), &item); err == nil {
			out = append(out, item)
		}
	}
	return out
}

func collectStrings(v any) []string {
	var out []string
	var walk func(any)
	walk = func(node any) {
		switch n := node.(type) {
		case map[string]any:
			for k, val := range n {
				out = append(out, k)
				walk(val)
			}
		case []any:
			for _, val := range n {
				walk(val)
			}
		case string:
			if trimmed := strings.TrimSpace(n); trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	walk(v)
	return out
}

func looksLikeSession(values []string) bool {
	joined := strings.ToLower(strings.Join(values, " "))
	markers := []string{
		"assistant",
		"user",
		"message",
		"prompt",
		"transcript",
		"conversation",
		"session",
		"model",
	}
	count := 0
	for _, marker := range markers {
		if strings.Contains(joined, marker) {
			count++
		}
	}
	return count >= 2
}

func detectModel(values []string) string {
	for _, value := range values {
		lower := strings.ToLower(value)
		if strings.Contains(lower, "gpt-") || strings.Contains(lower, "claude") ||
			strings.Contains(lower, "opus") ||
			strings.Contains(lower, "sonnet") {
			return value
		}
	}
	return ""
}

func detectSummary(values []string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if len(trimmed) >= 24 && len(trimmed) <= 180 && !looksLikeTimestamp(trimmed) &&
			!strings.Contains(trimmed, "/") &&
			!looksLikeIdentifier(trimmed) {
			return trimmed
		}
	}
	return "Imported local session metadata from provider-specific storage."
}

func looksLikeIdentifier(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if len(value) < 16 {
		return false
	}
	if strings.Count(value, "-") >= 3 {
		return true
	}
	for _, prefix := range []string{"ses_", "msg_", "turn_", "call_"} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func detectStatus(values []string) string {
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "active", "running", "completed", "failed", "discovered":
			return strings.ToLower(strings.TrimSpace(value))
		}
	}
	return "imported"
}

func detectTimes(values []string, fallback time.Time) (time.Time, time.Time) {
	var parsed []time.Time
	for _, value := range values {
		if ts, ok := parseTime(value); ok {
			parsed = append(parsed, ts)
		}
	}
	if len(parsed) == 0 {
		return fallback, fallback
	}
	sort.Slice(parsed, func(i, j int) bool { return parsed[i].Before(parsed[j]) })
	return parsed[0], parsed[len(parsed)-1]
}

func parseTime(value string) (time.Time, bool) {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

func looksLikeTimestamp(value string) bool {
	_, ok := parseTime(value)
	return ok
}

func inferProject(path string) string {
	base := filepath.Base(filepath.Dir(path))
	if base == ".codex" || base == "opencode" || base == ".claude" || base == "projects" ||
		base == ".config" {
		return "unknown"
	}
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "unknown"
	}
	return base
}

func fileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func slug(value string) string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer(" ", "-", ".", "-", "_", "-", "/", "-")
	return replacer.Replace(value)
}
