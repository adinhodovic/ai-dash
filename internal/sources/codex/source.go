package codex

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adin/ai-dash/internal/config"
	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources/shared"
)

type Source struct {
	pathOverride string
}

type sessionMetaPayload struct {
	ID            string `json:"id"`
	Timestamp     string `json:"timestamp"`
	Cwd           string `json:"cwd"`
	CliVersion    string `json:"cli_version"`
	ModelProvider string `json:"model_provider"`
}

type turnContextPayload struct {
	Cwd    string `json:"cwd"`
	Model  string `json:"model"`
	Effort string `json:"effort"`
}

type responseItemPayload struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type eventPayload struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

type logLine struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

var _ shared.SessionProvider = Source{}

func New(cfg config.Config) Source {
	return Source{pathOverride: cfg.SourcePath("codex", "")}
}

func (Source) Name() string { return "codex" }

func (s Source) Discover() (shared.Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return shared.Result{}, err
	}

	configPath := filepath.Join(home, ".codex/config.toml")
	if s.pathOverride != "" {
		configPath = s.pathOverride
	}
	baseDir := filepath.Dir(configPath)
	logDir := filepath.Join(baseDir, "logs")
	sessionsDir := filepath.Join(baseDir, "sessions")
	roots := uniquePaths(baseDir, logDir, sessionsDir)
	candidates, err := discoverCandidates(roots)
	if err != nil {
		return shared.Result{}, err
	}
	parsed := importCodexSessions(candidates)

	return shared.Result{
		Sources: []shared.Source{
			shared.NewSource("codex", "jsonl", sessionsDir, "Codex sessions directory"),
		},
		Sessions: parsed,
	}, nil
}

func (Source) ImportSessions(result shared.Result) ([]session.Session, error) {
	return append([]session.Session(nil), result.Sessions...), nil
}

func (Source) ResumeArgs(sessionID string) []string {
	return []string{"codex", "resume", sessionID}
}

func (Source) NewSessionArgs(projectDir string) []string {
	return []string{"codex", "--cwd", projectDir}
}

func discoverCandidates(roots []string) ([]string, error) {
	var all []string
	for _, root := range roots {
		matches, err := shared.DiscoverCandidateFilesWithPatterns(
			root,
			[]string{"codex", "thread", "run", "rollout"},
		)
		if err != nil {
			return nil, err
		}
		all = append(all, matches...)
	}
	return uniquePaths(all...), nil
}

func importCodexSessions(paths []string) []session.Session {
	out := make([]session.Session, 0, len(paths))
	for _, path := range paths {
		parsed, ok := parseCodexSession(path)
		if ok {
			out = append(out, parsed)
		}
	}
	session.Sort(out)
	return out
}

func parseCodexSession(path string) (session.Session, bool) {
	file, err := os.Open(path)
	if err != nil {
		return session.Session{}, false
	}
	defer func() { _ = file.Close() }()

	var (
		meta        sessionMetaPayload
		ctx         turnContextPayload
		userPrompt  string
		startedAt   time.Time
		endedAt     time.Time
		status      = "completed"
		seenMeta    bool
		seenContext bool
	)

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)
	for scanner.Scan() {
		var line logLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if ts, ok := parseTimestamp(line.Timestamp); ok {
			if startedAt.IsZero() || ts.Before(startedAt) {
				startedAt = ts
			}
			if endedAt.IsZero() || ts.After(endedAt) {
				endedAt = ts
			}
		}

		switch line.Type {
		case "session_meta":
			if json.Unmarshal(line.Payload, &meta) == nil {
				seenMeta = true
				if ts, ok := parseTimestamp(meta.Timestamp); ok &&
					(startedAt.IsZero() || ts.Before(startedAt)) {
					startedAt = ts
				}
			}
		case "turn_context":
			if json.Unmarshal(line.Payload, &ctx) == nil {
				seenContext = true
			}
		case "response_item":
			var payload responseItemPayload
			if json.Unmarshal(line.Payload, &payload) == nil && payload.Type == "message" &&
				payload.Role == "user" {
				for _, item := range payload.Content {
					if item.Type == "input_text" {
						text := sanitizeUserText(item.Text)
						if text != "" && !strings.HasPrefix(text, "<") {
							userPrompt = text
							break
						}
					}
				}
			}
		case "event_msg":
			var payload eventPayload
			if json.Unmarshal(line.Payload, &payload) == nil {
				switch payload.Type {
				case "turn_aborted":
					status = "aborted"
				case "task_started":
					if status == "completed" {
						status = "active"
					}
				case "user_message":
					if userPrompt == "" {
						userPrompt = sanitizeUserText(payload.Message)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return session.Session{}, false
	}
	if !seenMeta && !seenContext {
		return session.Session{}, false
	}
	if status == "active" && !endedAt.IsZero() {
		status = "completed"
	}

	cwd := firstNonEmpty(ctx.Cwd, meta.Cwd)
	model := firstNonEmpty(ctx.Model, meta.ModelProvider)
	project := inferProject(cwd, path)
	summary := summarizePrompt(userPrompt)
	tags := []string{"native-codex"}
	if meta.CliVersion != "" {
		tags = append(tags, "cli-"+meta.CliVersion)
	}

	meta2 := map[string]string{}
	if meta.CliVersion != "" {
		meta2["cli_version"] = meta.CliVersion
	}
	if meta.ModelProvider != "" {
		meta2["model_provider"] = meta.ModelProvider
	}
	if ctx.Effort != "" {
		meta2["effort"] = ctx.Effort
	}

	return session.Session{
		ID:             firstNonEmpty(meta.ID, sessionIDFromPath(path)),
		Slug:           sessionIDFromPath(path),
		Tool:           "codex",
		Project:        project,
		Repo:           cwd,
		Status:         status,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		Model:          model,
		Summary:        summary,
		TranscriptPath: path,
		Tags:           tags,
		Meta:           meta2,
	}, true
}

func parseTimestamp(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		if ts, err := time.Parse(layout, value); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

func sanitizeUserText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if strings.HasPrefix(text, "<environment_context>") ||
		strings.HasPrefix(text, "<turn_aborted>") {
		return ""
	}
	if strings.Contains(text, "AGENTS.md instructions") {
		return ""
	}
	return strings.Join(strings.Fields(text), " ")
}

func summarizePrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "Imported Codex session"
	}
	runes := []rune(prompt)
	if len(runes) > 120 {
		return string(runes[:119]) + "~"
	}
	return prompt
}

func inferProject(cwd, path string) string {
	if cwd != "" {
		base := filepath.Base(cwd)
		if base != "" && base != string(filepath.Separator) {
			return base
		}
	}
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, part := range parts {
		if part == "sessions" && i > 0 {
			return parts[i-1]
		}
	}
	return "unknown"
}

func sessionIDFromPath(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if idx := strings.LastIndex(name, "-"); idx != -1 && idx+1 < len(name) {
		return name[idx+1:]
	}
	return name
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func uniquePaths(paths ...string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}
