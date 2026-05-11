package claude

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/adinhodovic/ai-dash/internal/config"
	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources/shared"
)

func TestDiscoverFindsTranscriptSessions(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "repo-a")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	transcriptPath := filepath.Join(projectDir, "transcript.jsonl")
	if err := os.WriteFile(transcriptPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	result, err := New(config.Config{ClaudePath: root}).Discover()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(result.Transcripts) != 1 {
		t.Fatalf("expected 1 transcript, got %d", len(result.Transcripts))
	}
	if len(result.Sessions) != 1 || result.Sessions[0].Project != "repo-a" {
		t.Fatalf("expected imported repo-a session, got %#v", result.Sessions)
	}
	if result.Sessions[0].ID != "transcript" {
		t.Fatalf("expected transcript id from filename, got %#v", result.Sessions[0])
	}
}

func TestImportTranscriptSessionsBuildsDiscoveredSession(t *testing.T) {
	modTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	transcripts := []shared.TranscriptFile{{
		Tool:    "claude",
		Path:    "/tmp/transcript.jsonl",
		Project: "demo",
		ModTime: modTime,
	}}

	sessions := importTranscriptSessions(transcripts)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].TranscriptPath != "/tmp/transcript.jsonl" || sessions[0].StartedAt != modTime {
		t.Fatalf("unexpected session: %#v", sessions[0])
	}
	if sessions[0].ID != "transcript" {
		t.Fatalf("expected transcript id from filename, got %#v", sessions[0])
	}
}

func TestParseClaudeTranscriptFromFixture(t *testing.T) {
	path := filepath.Join("testdata", "session.jsonl")
	transcript := shared.TranscriptFile{
		Tool:    "claude",
		Path:    path,
		Project: "webapp",
		ModTime: time.Now(),
	}
	s := parseClaudeTranscript(transcript)

	if s.Summary != "add rate limiting to the login endpoint" {
		t.Errorf("summary = %q, want first user message", s.Summary)
	}
	if s.ID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("id = %q, want sessionId from transcript", s.ID)
	}
	if s.Slug != "bright-morning-star" {
		t.Errorf("slug = %q, want slug from transcript", s.Slug)
	}
	if s.Repo != "/home/user/projects/webapp" {
		t.Errorf("repo = %q, want cwd", s.Repo)
	}
	if s.Branch != "feature/auth" {
		t.Errorf("branch = %q", s.Branch)
	}
	if s.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want model from assistant message", s.Model)
	}
	if s.Status != "completed" {
		t.Errorf("status = %q, want completed (last stop_reason=end_turn)", s.Status)
	}
	if s.CurrentState != "done" {
		t.Errorf("current state = %q, want done", s.CurrentState)
	}
	if s.Meta["current_state_source"] != "stop_reason=end_turn" {
		t.Errorf(
			"current_state_source = %q, want stop_reason=end_turn",
			s.Meta["current_state_source"],
		)
	}
	if s.TokensIn == 0 {
		t.Error("expected non-zero TokensIn")
	}
	if s.TokensOut == 0 {
		t.Error("expected non-zero TokensOut")
	}
	expectedIn := 1200 + 2400
	expectedOut := 150 + 280
	if s.TokensIn != expectedIn {
		t.Errorf("TokensIn = %d, want %d", s.TokensIn, expectedIn)
	}
	if s.TokensOut != expectedOut {
		t.Errorf("TokensOut = %d, want %d", s.TokensOut, expectedOut)
	}
}

func TestParseClaudeTranscriptMarksToolUseAsToolCall(t *testing.T) {
	s := parseClaudeTranscript(shared.TranscriptFile{
		Tool:    "claude",
		Path:    filepath.Join("testdata", "tool_use_session.jsonl"),
		ModTime: time.Now(),
	})
	if s.Status != "active" {
		t.Fatalf("status = %q, want active", s.Status)
	}
	if s.CurrentState != "tool call" {
		t.Fatalf("current state = %q, want tool call", s.CurrentState)
	}
	if s.Meta["current_state_source"] != "stop_reason=tool_use" {
		t.Fatalf(
			"current_state_source = %q, want stop_reason=tool_use",
			s.Meta["current_state_source"],
		)
	}
}

func TestParseClaudeTranscriptUsesModtimeHeuristicOnlyWithoutStopReason(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "session.jsonl")
	content := []byte("{" +
		"\"type\":\"user\",\"timestamp\":\"2026-04-01T00:00:00Z\",\"message\":{\"role\":\"user\",\"content\":\"check status\"},\"cwd\":\"/tmp/demo\",\"sessionId\":\"sess-1\"}\n" +
		"{\"type\":\"assistant\",\"timestamp\":\"2026-04-01T00:00:01Z\",\"message\":{\"role\":\"assistant\",\"model\":\"claude-sonnet-4-6\"},\"cwd\":\"/tmp/demo\",\"sessionId\":\"sess-1\"}\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	s := parseClaudeTranscript(shared.TranscriptFile{
		Tool:    "claude",
		Path:    path,
		ModTime: time.Now(),
	})
	if s.Status != "active" || s.CurrentState != "running" {
		t.Fatalf("got (%q, %q), want (active, running)", s.Status, s.CurrentState)
	}
	if s.Meta["current_state_source"] != "transcript.modtime heuristic" {
		t.Fatalf("current_state_source = %q", s.Meta["current_state_source"])
	}
}

func TestParentSessionIDClassifiesSubagents(t *testing.T) {
	src := New(config.Config{})

	parent := session.Session{
		TranscriptPath: "/home/user/.claude/projects/repo-a/abc-123.jsonl",
	}
	if id := src.ParentSessionID(parent); id != "" {
		t.Fatalf("expected no parent for top-level session, got %q", id)
	}

	child := session.Session{
		TranscriptPath: "/home/user/.claude/projects/repo-a/abc-123/subagents/def-456.jsonl",
	}
	if id := src.ParentSessionID(child); id != "abc-123" {
		t.Fatalf("expected parent ID %q, got %q", "abc-123", id)
	}
}

func TestResolveProjectRootWalksToGitDir(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	nested := filepath.Join(repo, "internal", "ui")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	cache := map[string]string{}
	if got := resolveProjectRoot(nested, cache); got != repo {
		t.Fatalf("resolveProjectRoot(%q) = %q, want %q", nested, got, repo)
	}
	if got := resolveProjectRoot(repo, cache); got != repo {
		t.Fatalf("resolveProjectRoot(%q) = %q, want %q", repo, got, repo)
	}
}

func TestResolveProjectRootHandlesGitFileWorktree(t *testing.T) {
	repo := t.TempDir()
	// In git worktrees, .git is a regular file, not a directory.
	if err := os.WriteFile(
		filepath.Join(repo, ".git"),
		[]byte("gitdir: /elsewhere\n"),
		0o644,
	); err != nil {
		t.Fatalf("write .git: %v", err)
	}
	nested := filepath.Join(repo, "pkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	if got := resolveProjectRoot(nested, map[string]string{}); got != repo {
		t.Fatalf("resolveProjectRoot(%q) = %q, want %q", nested, got, repo)
	}
}

func TestResolveProjectRootFallsBackToInput(t *testing.T) {
	dir := t.TempDir() // no .git anywhere
	if got := resolveProjectRoot(dir, map[string]string{}); got != dir {
		t.Fatalf("resolveProjectRoot(%q) = %q, want fallback to input", dir, got)
	}
}

func TestResolveProjectRootMissingDirReturnsInput(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if got := resolveProjectRoot(missing, map[string]string{}); got != missing {
		t.Fatalf("resolveProjectRoot(%q) = %q, want input unchanged for missing dir", missing, got)
	}
}

func TestImportConsolidatesSubdirSessionsIntoGitRoot(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	nested := filepath.Join(repo, "internal", "ui")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	rootTranscript := filepath.Join(t.TempDir(), "root.jsonl")
	rootContent := []byte(
		"{\"type\":\"user\",\"timestamp\":\"2026-04-01T00:00:00Z\",\"message\":{\"role\":\"user\",\"content\":\"hi\"},\"cwd\":\"" + repo + "\",\"sessionId\":\"sess-root\"}\n",
	)
	if err := os.WriteFile(rootTranscript, rootContent, 0o644); err != nil {
		t.Fatalf("write root transcript: %v", err)
	}
	nestedTranscript := filepath.Join(t.TempDir(), "nested.jsonl")
	nestedContent := []byte(
		"{\"type\":\"user\",\"timestamp\":\"2026-04-01T00:00:00Z\",\"message\":{\"role\":\"user\",\"content\":\"hi\"},\"cwd\":\"" + nested + "\",\"sessionId\":\"sess-nested\"}\n",
	)
	if err := os.WriteFile(nestedTranscript, nestedContent, 0o644); err != nil {
		t.Fatalf("write nested transcript: %v", err)
	}

	sessions := importTranscriptSessions([]shared.TranscriptFile{
		{Tool: "claude", Path: rootTranscript, Project: "bucket-root", ModTime: time.Now()},
		{Tool: "claude", Path: nestedTranscript, Project: "bucket-nested", ModTime: time.Now()},
	})

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].Project != repo || sessions[1].Project != repo {
		t.Fatalf("expected both sessions to share project %q, got %q and %q",
			repo, sessions[0].Project, sessions[1].Project)
	}
}

func TestDiscoverIncludesSubagentTranscripts(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "repo-a")
	subagentDir := filepath.Join(projectDir, "parent-session", "subagents")
	if err := os.MkdirAll(subagentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	parentPath := filepath.Join(projectDir, "parent-session.jsonl")
	if err := os.WriteFile(parentPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write parent transcript: %v", err)
	}
	agentPath := filepath.Join(subagentDir, "agent-123.jsonl")
	if err := os.WriteFile(agentPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write subagent transcript: %v", err)
	}
	result, err := New(config.Config{ClaudePath: root}).Discover()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(result.Sessions) != 2 {
		t.Fatalf("expected 2 sessions (parent + subagent), got %d", len(result.Sessions))
	}
}
