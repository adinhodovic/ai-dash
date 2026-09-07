package pi

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adinhodovic/ai-dash/internal/config"
	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources/shared"
)

type Source struct {
	pathOverride string
}

var _ shared.SessionProvider = Source{}

func New(cfg config.Config) Source {
	return Source{pathOverride: cfg.SourcePath("pi", "")}
}

func (Source) Name() string { return "pi" }

func (s Source) Discover() (shared.Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return shared.Result{}, err
	}

	sessionsPath := filepath.Join(home, ".pi/agent/sessions")
	if s.pathOverride != "" {
		sessionsPath = s.pathOverride
	}

	paths, err := discoverSessionFiles(sessionsPath)
	if err != nil {
		return shared.Result{}, err
	}

	return shared.Result{
		Sources: []shared.Source{
			shared.NewSource("pi", "jsonl", sessionsPath, "pi agent sessions"),
		},
		Sessions: importPiSessions(paths),
	}, nil
}

func (Source) ResumeArgs(sessionID, projectDir string) []string {
	q := shared.ShellQuote
	if projectDir != "" {
		return []string{"cd", q(projectDir), "&&", "pi", "--session", q(sessionID)}
	}
	return []string{"pi", "--session", q(sessionID)}
}

func (Source) NewSessionArgs(projectDir string) []string {
	q := shared.ShellQuote
	return []string{"cd", q(projectDir), "&&", "pi"}
}

func discoverSessionFiles(root string) ([]string, error) {
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

	var paths []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

// rawEntry is a lenient decode target covering every line shape in a pi
// session file: the header (type=="session") and every tree entry
// (type=="message"|"session_info"|"model_change"|"compaction"|...).
type rawEntry struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	ParentID  *string         `json:"parentId"`
	Timestamp string          `json:"timestamp"`
	Cwd       string          `json:"cwd"` // header only
	Name      string          `json:"name"`
	Message   *messagePayload `json:"message"`
}

type usageCost struct {
	Total float64 `json:"total"`
}

type usagePayload struct {
	Input  int       `json:"input"`
	Output int       `json:"output"`
	Cost   usageCost `json:"cost"`
}

// messagePayload loosely covers every AgentMessage role. Fields that don't
// apply to a given role are simply left zero, mirroring how codex/claude
// use one lenient struct across a native message-type union.
type messagePayload struct {
	Role         string          `json:"role"`
	Content      json.RawMessage `json:"content"`
	Model        string          `json:"model"`
	Usage        *usagePayload   `json:"usage"`
	StopReason   string          `json:"stopReason"`
	ErrorMessage string          `json:"errorMessage"`
}

// treeEntry is the tree-walk representation of one non-header line.
type treeEntry struct {
	id        string
	parentID  string
	hasParent bool
	entryType string
	timestamp time.Time
	message   *messagePayload
	name      string // session_info title, if this entry is a session_info
}

func importPiSessions(paths []string) []session.Session {
	out := make([]session.Session, 0, len(paths))
	gitRootCache := map[string]string{}
	for _, path := range paths {
		parsed, ok := parsePiSession(path)
		if !ok {
			continue
		}
		if parsed.Repo != "" {
			parsed.Project = resolveProjectRoot(parsed.Repo, gitRootCache)
		}
		out = append(out, parsed)
	}
	session.Sort(out)
	return out
}

func parsePiSession(path string) (session.Session, bool) {
	file, err := os.Open(path)
	if err != nil {
		return session.Session{}, false
	}
	defer func() { _ = file.Close() }()

	var (
		headerCwd       string
		headerID        string
		headerTimestamp time.Time
		seenHeader      bool
		entries         = map[string]treeEntry{}
		order           []string
	)

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)
	for scanner.Scan() {
		var raw rawEntry
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}
		if raw.Type == "session" {
			if !seenHeader {
				headerCwd = raw.Cwd
				headerID = raw.ID
				headerTimestamp, _ = parseTimestamp(raw.Timestamp)
				seenHeader = true
			}
			continue
		}
		if raw.ID == "" {
			continue
		}
		ts, _ := parseTimestamp(raw.Timestamp)
		te := treeEntry{
			id:        raw.ID,
			entryType: raw.Type,
			timestamp: ts,
			message:   raw.Message,
			name:      raw.Name,
		}
		if raw.ParentID != nil {
			te.parentID = *raw.ParentID
			te.hasParent = true
		}
		entries[raw.ID] = te
		order = append(order, raw.ID)
	}
	if err := scanner.Err(); err != nil {
		return session.Session{}, false
	}
	if !seenHeader {
		return session.Session{}, false
	}

	activePath := walkActivePath(order, entries)

	var (
		summary         string
		sessionInfoName string
		lastAssistant   *messagePayload
		tokensIn        int
		tokensOut       int
		costTotal       float64
	)
	for _, e := range activePath {
		switch e.entryType {
		case "session_info":
			if e.name != "" {
				sessionInfoName = e.name
			}
		case "message":
			if e.message == nil {
				continue
			}
			switch e.message.Role {
			case "user":
				if summary == "" {
					if text := sanitizeText(extractText(e.message.Content)); text != "" {
						summary = text
					}
				}
			case "assistant":
				lastAssistant = e.message
				if e.message.Usage != nil {
					tokensIn += e.message.Usage.Input
					tokensOut += e.message.Usage.Output
					costTotal += e.message.Usage.Cost.Total
				}
			}
		}
	}
	if sessionInfoName != "" {
		summary = sessionInfoName
	}

	startedAt := headerTimestamp
	endedAt := headerTimestamp
	if len(activePath) > 0 {
		leaf := activePath[len(activePath)-1]
		if !leaf.timestamp.IsZero() {
			endedAt = leaf.timestamp
		}
	}

	var leaf *treeEntry
	if len(activePath) > 0 {
		leaf = &activePath[len(activePath)-1]
	}
	status, currentState, stateSource := classifyLeaf(leaf, endedAt)

	metaMap := map[string]string{
		"current_state_source": stateSource,
	}
	if lastAssistant != nil && lastAssistant.ErrorMessage != "" {
		metaMap["error"] = lastAssistant.ErrorMessage
	}

	model := ""
	if lastAssistant != nil {
		model = lastAssistant.Model
	}

	return session.Session{
		ID:             firstNonEmpty(headerID, sessionIDFromPath(path)),
		Slug:           sessionIDFromPath(path),
		Tool:           "pi",
		Repo:           headerCwd,
		Status:         status,
		CurrentState:   currentState,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		Model:          model,
		Summary:        summarizePrompt(summary),
		TranscriptPath: path,
		TokensIn:       tokensIn,
		TokensOut:      tokensOut,
		CostUSD:        costTotal,
		Tags:           []string{"pi"},
		Meta:           metaMap,
	}, true
}

// walkActivePath returns the entries on the path from the tree root to the
// current leaf, in root-to-leaf order. pi only ever appends new entries at
// the current leaf — branches abandoned by /fork or an edited message are
// written earlier in the file and never touched again — so the entry with
// an id that appears last in file order is exactly the active leaf.
func walkActivePath(order []string, entries map[string]treeEntry) []treeEntry {
	if len(order) == 0 {
		return nil
	}
	leafID := order[len(order)-1]

	var reversed []treeEntry
	seen := map[string]bool{}
	cur := leafID
	for cur != "" {
		if seen[cur] {
			break
		}
		seen[cur] = true
		e, ok := entries[cur]
		if !ok {
			break
		}
		reversed = append(reversed, e)
		if !e.hasParent {
			break
		}
		cur = e.parentID
	}

	activePath := make([]treeEntry, len(reversed))
	for i, e := range reversed {
		activePath[len(reversed)-1-i] = e
	}
	return activePath
}

func classifyLeaf(leaf *treeEntry, endedAt time.Time) (status, currentState, source string) {
	if leaf != nil && leaf.entryType == "message" && leaf.message != nil {
		switch leaf.message.Role {
		case "assistant":
			switch leaf.message.StopReason {
			case "stop":
				return string(session.StatusActive), string(session.StateWaiting), "stopReason=stop"
			case "length":
				return string(
						session.StatusActive,
					), string(
						session.StateMaxTokens,
					), "stopReason=length"
			case "toolUse":
				return string(
						session.StatusActive,
					), string(
						session.StateToolCall,
					), "stopReason=toolUse"
			case "error":
				return string(
						session.StatusAborted,
					), string(
						session.StateAborted,
					), "stopReason=error"
			case "aborted":
				return string(
						session.StatusAborted,
					), string(
						session.StateAborted,
					), "stopReason=aborted"
			}
		case "toolResult":
			return string(session.StatusActive), string(session.StateRunning), "leaf=toolResult"
		case "bashExecution":
			return string(session.StatusActive), string(session.StateRunning), "leaf=bashExecution"
		case "user":
			return recencyState(endedAt, "leaf=user message")
		}
	}
	label := "leaf=none"
	if leaf != nil {
		label = "leaf=" + leaf.entryType
	}
	return recencyState(endedAt, label)
}

func recencyState(endedAt time.Time, label string) (status, currentState, source string) {
	if time.Since(endedAt) < 5*time.Minute {
		return string(session.StatusActive), string(session.StateRunning), label + " (recent)"
	}
	return string(session.StatusCompleted), string(session.StateDone), label + " (stale)"
}

func extractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, " ")
}

func sanitizeText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(text, "<") {
		return ""
	}
	return strings.Join(strings.Fields(text), " ")
}

func summarizePrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "Imported pi session"
	}
	// The UI wraps and truncates summaries for display itself (up to two
	// lines), so this only needs to guard against pathologically long
	// messages, not pre-truncate to whatever a list row happens to fit.
	runes := []rune(prompt)
	if len(runes) > 500 {
		return string(runes[:499]) + "~"
	}
	return prompt
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

func sessionIDFromPath(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if idx := strings.LastIndex(name, "_"); idx != -1 && idx+1 < len(name) {
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
