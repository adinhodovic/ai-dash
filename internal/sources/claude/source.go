package claude

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adinhodovic/ai-dash/internal/config"
	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources/shared"
)

type Source struct {
	pathOverride string
	cache        *shared.Cache
}

var (
	_ shared.SessionProvider    = Source{}
	_ shared.SubagentClassifier = Source{}
	_ shared.PromptReader       = Source{}
	_ shared.HeadlessRunner     = Source{}
)

func New(cfg config.Config, cache *shared.Cache) Source {
	return Source{pathOverride: cfg.SourcePath("claude", ""), cache: cache}
}

func (Source) Name() string { return "claude" }

func (s Source) Discover() (shared.Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return shared.Result{}, err
	}

	projectsPath := filepath.Join(home, ".claude/projects")
	if s.pathOverride != "" {
		projectsPath = s.pathOverride
	}

	result := shared.Result{
		Sources: []shared.Source{
			shared.NewSource("claude", "jsonl", projectsPath, "Claude Code transcripts"),
		},
	}

	transcripts, err := discoverTranscripts(projectsPath)
	result.Transcripts = transcripts
	result.Sessions = importTranscriptSessions(s.cache, transcripts)
	return result, err
}

func (Source) ResumeArgs(sessionID, projectDir string) []string {
	q := shared.ShellQuote
	if projectDir != "" {
		return []string{"cd", q(projectDir), "&&", "claude", "--resume", q(sessionID)}
	}
	return []string{"claude", "--resume", q(sessionID)}
}

func (Source) NewSessionArgs(projectDir string) []string {
	q := shared.ShellQuote
	return []string{"cd", q(projectDir), "&&", "claude"}
}

func (Source) PromptArgs(prompt string) []string {
	return []string{shared.ShellQuote(prompt)}
}

func (Source) ParentSessionID(s session.Session) string {
	return subagentParentID(s.TranscriptPath)
}

// HeadlessArgs uses haiku with no tools: titling needs neither, and it's fast.
func (Source) HeadlessArgs(prompt string) []string {
	return []string{
		"claude", "-p", "--no-session-persistence", "--model", "haiku", "--tools=", prompt,
	}
}

// UserPrompts returns the user's typed messages, skipping tool results and
// harness-injected messages (slash command output, reminders, notifications).
func (Source) UserPrompts(s session.Session) ([]string, error) {
	file, err := os.Open(s.TranscriptPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)
	var prompts []string
	for scanner.Scan() {
		var line claudeLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type != "user" || line.IsMeta || line.Message == nil ||
			line.Message.Role != "user" {
			continue
		}
		text := extractTextContent(line.Message.Content)
		if text == "" || strings.HasPrefix(text, "<") || strings.HasPrefix(text, "Caveat:") {
			continue
		}
		prompts = append(prompts, text)
	}
	return prompts, scanner.Err()
}

func discoverTranscripts(root string) ([]shared.TranscriptFile, error) {
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

	var transcripts []shared.TranscriptFile
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		transcripts = append(transcripts, shared.TranscriptFile{
			Tool:    "claude",
			Path:    path,
			Project: projectNameFromPath(root, path),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
		return nil
	})
	return transcripts, err
}

func projectNameFromPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) == 0 {
		return ""
	}
	return sanitizeProjectName(parts[0])
}

func sanitizeProjectName(value string) string {
	original := strings.Trim(value, "-")
	if original == "" {
		return "unknown"
	}
	// Claude encodes project paths by replacing / with -.
	// e.g. /home/user/projects/myapp -> -home-user-projects-myapp
	// We can't perfectly reverse this (- is ambiguous), but we can match
	// against the home directory slug to strip the known prefix.
	if !strings.HasPrefix(value, "-") {
		return original
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return original
	}
	homeSlug := strings.ReplaceAll(strings.TrimPrefix(home, "/"), "/", "-")
	if rest, ok := strings.CutPrefix(original, homeSlug+"-"); ok {
		return rest
	}
	return original
}

// claudeLine represents a single JSONL entry in a Claude Code transcript.
type claudeLine struct {
	Type      string `json:"type"`
	IsMeta    bool   `json:"isMeta"`
	SessionID string `json:"sessionId"`
	Slug      string `json:"slug"`
	Version   string `json:"version"`
	Cwd       string `json:"cwd"`
	GitBranch string `json:"gitBranch"`
	Timestamp string `json:"timestamp"`
	Message   *struct {
		ID         string       `json:"id"`
		Role       string       `json:"role"`
		Model      string       `json:"model"`
		Content    any          `json:"content"`
		StopReason string       `json:"stop_reason"`
		Usage      *claudeUsage `json:"usage"`
	} `json:"message"`
}

type claudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

// totalInput counts cached prompt tokens too; with prompt caching,
// input_tokens alone is only the uncached tail of each request.
func (u claudeUsage) totalInput() int {
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}

func importTranscriptSessions(
	cache *shared.Cache,
	transcripts []shared.TranscriptFile,
) []session.Session {
	gitRootCache := map[string]string{}
	return cache.Sessions(
		"claude",
		transcripts,
		func(f shared.TranscriptFile) (session.Session, bool) {
			s := parseClaudeTranscript(f)
			if s.Repo != "" {
				s.Project = resolveProjectRoot(s.Repo, gitRootCache)
			}
			return s, true
		}, refreshClaudeStatus)
}

// refreshClaudeStatus needs no nil check: parseClaudeTranscript always sets Meta["current_state_source"], so Meta survives the cache's JSON round-trip.
func refreshClaudeStatus(s *session.Session, modTime time.Time) {
	applyClaudeStatus(s, s.Meta["stop_reason"], modTime)
}

func applyClaudeStatus(s *session.Session, stop string, modTime time.Time) {
	s.Status, s.CurrentState = claudeCurrentState(stop, modTime)
	s.Meta["current_state_source"] = claudeCurrentStateSource(stop, modTime)
	s.CurrentState = session.StatusLabel(*s)
}

// resolveProjectRoot returns the git repo root containing dir, or dir itself
// if no git root is found. The cache memoizes lookups within a single import
// pass so subdirectory sessions in the same repo collapse into one project.
func resolveProjectRoot(dir string, cache map[string]string) string {
	if dir == "" {
		return dir
	}
	if cached, ok := cache[dir]; ok {
		return cached
	}
	root := findGitRoot(dir)
	if root == "" {
		root = dir
	}
	cache[dir] = root
	return root
}

func findGitRoot(dir string) string {
	if _, err := os.Stat(dir); err != nil {
		return ""
	}
	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func parseClaudeTranscript(transcript shared.TranscriptFile) session.Session {
	sessionID := sessionIDFromPath(transcript.Path)
	project := transcript.Project
	if project == "" {
		project = "unknown"
	}

	s := session.Session{
		ID:             sessionID,
		Slug:           sessionID,
		Tool:           "claude",
		Project:        project,
		Status:         string(session.StatusCompleted),
		StartedAt:      transcript.ModTime,
		EndedAt:        transcript.ModTime,
		TranscriptPath: transcript.Path,
		Tags:           []string{"auto-imported"},
	}

	file, err := os.Open(transcript.Path)
	if err != nil {
		return s
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	var (
		firstUserMsg   string
		startedAt      time.Time
		endedAt        time.Time
		lastStopReason string
		// One API response is split into a line per content block, each
		// repeating the same usage, so usage is keyed by message ID.
		usageByMsg = map[string]claudeUsage{}
		usageOrder []string
	)

	for scanner.Scan() {
		var line claudeLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}

		if ts := parseTimestamp(line.Timestamp); !ts.IsZero() {
			if startedAt.IsZero() || ts.Before(startedAt) {
				startedAt = ts
			}
			if endedAt.IsZero() || ts.After(endedAt) {
				endedAt = ts
			}
		}

		if line.SessionID != "" && s.ID == sessionID {
			s.ID = line.SessionID
		}
		if line.Slug != "" {
			s.Slug = line.Slug
		}
		if line.Version != "" {
			s.Model = line.Version
		}
		if line.Cwd != "" {
			s.Repo = line.Cwd
		}
		if line.GitBranch != "" {
			s.Branch = line.GitBranch
		}

		if line.Type == "user" && line.Message != nil && line.Message.Role == "user" &&
			firstUserMsg == "" {
			firstUserMsg = extractTextContent(line.Message.Content)
		}

		if line.Type == "assistant" && line.Message != nil {
			if line.Message.Model != "" {
				s.Model = line.Message.Model
			}
			if line.Message.StopReason != "" {
				lastStopReason = line.Message.StopReason
			}
			if line.Message.Usage != nil {
				id := line.Message.ID
				if id == "" {
					id = strconv.Itoa(len(usageOrder))
				}
				if _, seen := usageByMsg[id]; !seen {
					usageOrder = append(usageOrder, id)
				}
				usageByMsg[id] = *line.Message.Usage
			}
		}
	}

	if !startedAt.IsZero() {
		s.StartedAt = startedAt
	}
	if !endedAt.IsZero() {
		s.EndedAt = endedAt
	}
	for _, id := range usageOrder {
		u := usageByMsg[id]
		s.TokensIn += u.totalInput()
		s.TokensOut += u.OutputTokens
	}

	// Use cwd for project name — it's the actual path, not slug-encoded.
	if s.Repo != "" {
		s.Project = s.Repo
	}

	if firstUserMsg != "" {
		s.Summary = summarizeUserMessage(firstUserMsg)
	} else {
		s.Summary = "Imported session"
	}

	s.Meta = map[string]string{}
	if s.Model != "" {
		s.Meta["model"] = s.Model
	}
	if lastStopReason != "" {
		s.Meta["stop_reason"] = lastStopReason
	}
	if s.Branch != "" {
		s.Meta["branch"] = s.Branch
	}

	applyClaudeStatus(&s, lastStopReason, transcript.ModTime)

	return s
}

func claudeCurrentState(lastStopReason string, modTime time.Time) (string, string) {
	if lastStopReason != "" {
		if lastStopReason == "end_turn" {
			return string(session.StatusCompleted), string(session.StateDone)
		}
		switch lastStopReason {
		case "pause_turn":
			return string(session.StatusActive), string(session.StateWaiting)
		case "tool_use":
			return string(session.StatusActive), string(session.StateToolCall)
		case "max_tokens":
			return string(session.StatusActive), string(session.StateMaxTokens)
		default:
			return string(session.StatusActive), string(session.StateRunning)
		}
	}
	if !modTime.IsZero() && time.Since(modTime) < 5*time.Minute {
		return string(session.StatusActive), string(session.StateRunning)
	}
	return string(session.StatusCompleted), string(session.StateDone)
}

func claudeCurrentStateSource(lastStopReason string, modTime time.Time) string {
	if lastStopReason != "" {
		return "stop_reason=" + lastStopReason
	}
	if !modTime.IsZero() && time.Since(modTime) < 5*time.Minute {
		return "transcript.modtime heuristic"
	}
	return "transcript complete"
}

func extractTextContent(content any) string {
	switch c := content.(type) {
	case string:
		return strings.TrimSpace(c)
	case []any:
		for _, item := range c {
			if m, ok := item.(map[string]any); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						return strings.TrimSpace(text)
					}
				}
			}
		}
	}
	return ""
}

func summarizeUserMessage(msg string) string {
	// Collapse whitespace
	msg = strings.Join(strings.Fields(msg), " ")
	// Skip system-like messages
	if strings.HasPrefix(msg, "<") || strings.Contains(msg, "AGENTS.md") {
		return "Imported session"
	}
	// The UI wraps and truncates summaries for display itself (up to two
	// lines), so this only needs to guard against pathologically long
	// messages, not pre-truncate to whatever a list row happens to fit.
	runes := []rune(msg)
	if len(runes) > 500 {
		return string(runes[:499]) + "~"
	}
	return msg
}

func parseTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		if ts, err := time.Parse(layout, value); err == nil {
			return ts
		}
	}
	return time.Time{}
}

func subagentParentID(path string) string {
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	for i, part := range parts {
		if part == "subagents" && i > 0 {
			return parts[i-1]
		}
	}
	return ""
}

func sessionIDFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
