package pi

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/adinhodovic/ai-dash/internal/config"
)

func TestParsePiSessionStop(t *testing.T) {
	parsed, ok := parsePiSession("testdata/session.jsonl")
	if !ok {
		t.Fatalf("expected fixture to parse")
	}
	if parsed.ID != "pi-session-uuid-1" {
		t.Fatalf("unexpected id: %#v", parsed)
	}
	if parsed.Tool != "pi" {
		t.Fatalf("unexpected tool: %#v", parsed)
	}
	if parsed.Repo != "/home/user/projects/myapp" {
		t.Fatalf("unexpected repo: %#v", parsed)
	}
	if parsed.Model != "claude-sonnet-4-5" {
		t.Fatalf("unexpected model: %#v", parsed)
	}
	if parsed.Status != "active" || parsed.CurrentState != "waiting" {
		t.Fatalf("unexpected status/state: %#v", parsed)
	}
	if parsed.Meta["current_state_source"] != "stopReason=stop" {
		t.Fatalf("unexpected current_state_source: %#v", parsed)
	}
	if parsed.Summary != "Fix the flaky test in the auth suite" {
		t.Fatalf("unexpected summary: %#v", parsed)
	}
	if parsed.TokensIn != 1200 || parsed.TokensOut != 430 {
		t.Fatalf("unexpected token counts: %#v", parsed)
	}
	if parsed.CostUSD != 0.009 {
		t.Fatalf("unexpected cost: %#v", parsed)
	}
}

func TestParsePiSessionToolUseLeaf(t *testing.T) {
	parsed, ok := parsePiSession("testdata/tool_use_session.jsonl")
	if !ok {
		t.Fatalf("expected fixture to parse")
	}
	if parsed.Status != "active" || parsed.CurrentState != "tool call" {
		t.Fatalf("unexpected status/state: %#v", parsed)
	}
	if parsed.Meta["current_state_source"] != "stopReason=toolUse" {
		t.Fatalf("unexpected current_state_source: %#v", parsed)
	}
}

func TestParsePiSessionToolResultLeafBeatsLastAssistantMessage(t *testing.T) {
	parsed, ok := parsePiSession("testdata/active_session.jsonl")
	if !ok {
		t.Fatalf("expected fixture to parse")
	}
	// The last assistant message has stopReason=toolUse, but a toolResult
	// entry follows it as the true leaf -- classification must key off the
	// leaf, not "the last assistant message".
	if parsed.Status != "active" || parsed.CurrentState != "running" {
		t.Fatalf("unexpected status/state: %#v", parsed)
	}
	if parsed.Meta["current_state_source"] != "leaf=toolResult" {
		t.Fatalf("unexpected current_state_source: %#v", parsed)
	}
}

func TestParsePiSessionAborted(t *testing.T) {
	parsed, ok := parsePiSession("testdata/aborted_session.jsonl")
	if !ok {
		t.Fatalf("expected fixture to parse")
	}
	if parsed.Status != "aborted" || parsed.CurrentState != "aborted" {
		t.Fatalf("unexpected status/state: %#v", parsed)
	}
	if parsed.Meta["current_state_source"] != "stopReason=error" {
		t.Fatalf("unexpected current_state_source: %#v", parsed)
	}
	if parsed.Meta["error"] != "upstream API returned 529" {
		t.Fatalf("unexpected error meta: %#v", parsed)
	}
}

func TestParsePiSessionInterspersedAdminEntries(t *testing.T) {
	parsed, ok := parsePiSession("testdata/interspersed_admin_session.jsonl")
	if !ok {
		t.Fatalf("expected fixture to parse")
	}
	if parsed.Status != "active" || parsed.CurrentState != "waiting" {
		t.Fatalf("unexpected status/state: %#v", parsed)
	}
	if parsed.Summary != "Add cohort retention chart" {
		t.Fatalf("unexpected summary: %#v", parsed)
	}
	if parsed.Model != "gpt-4o" {
		t.Fatalf("unexpected model: %#v", parsed)
	}
	// Two assistant messages contribute usage; model_change/compaction/
	// branch_summary entries in between must not break the walk or get
	// double-counted.
	if parsed.TokensIn != 1300 || parsed.TokensOut != 230 {
		t.Fatalf("unexpected token counts: %#v", parsed)
	}
	if parsed.CostUSD != 0.005 {
		t.Fatalf("unexpected cost: %#v", parsed)
	}
}

func TestParsePiSessionForkIgnoresAbandonedFirstMessage(t *testing.T) {
	parsed, ok := parsePiSession("testdata/forked_session.jsonl")
	if !ok {
		t.Fatalf("expected fixture to parse")
	}
	if parsed.Summary != "Please write a binary search tree implementation instead" {
		t.Fatalf("expected summary from active branch, got %#v", parsed.Summary)
	}
}

func TestParsePiSessionSessionInfoOverridesSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	content := `{"type":"session","version":3,"id":"named-session","timestamp":"2024-12-03T20:00:00.000Z","cwd":"/home/user/projects/myapp"}
{"type":"message","id":"a1","parentId":null,"timestamp":"2024-12-03T20:00:01.000Z","message":{"role":"user","content":"Investigate the auth bug"}}
{"type":"session_info","id":"a2","parentId":"a1","timestamp":"2024-12-03T20:00:02.000Z","name":"Auth bug investigation"}
{"type":"message","id":"a3","parentId":"a2","timestamp":"2024-12-03T20:05:00.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Found the root cause."}],"model":"claude-sonnet-4-5","stopReason":"stop"}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	parsed, ok := parsePiSession(path)
	if !ok {
		t.Fatalf("expected fixture to parse")
	}
	if parsed.Summary != "Auth bug investigation" {
		t.Fatalf("expected session_info name to override summary, got %#v", parsed.Summary)
	}
}

func TestParsePiSessionUserLeafRecency(t *testing.T) {
	writeUserLeafFixture := func(t *testing.T, ts time.Time) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "session.jsonl")
		content := `{"type":"session","version":3,"id":"leaf-session","timestamp":"` + ts.Format(
			time.RFC3339,
		) + `","cwd":"/home/user/projects/myapp"}
{"type":"message","id":"b1","parentId":null,"timestamp":"` + ts.Format(
			time.RFC3339,
		) + `","message":{"role":"user","content":"Are you still there?"}}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		return path
	}

	fresh := writeUserLeafFixture(t, time.Now().Add(-1*time.Minute))
	parsed, ok := parsePiSession(fresh)
	if !ok {
		t.Fatalf("expected fresh fixture to parse")
	}
	if parsed.Status != "active" || parsed.CurrentState != "running" {
		t.Fatalf("expected fresh user leaf to be active/running, got %#v", parsed)
	}

	stale := writeUserLeafFixture(t, time.Now().Add(-24*time.Hour))
	parsed, ok = parsePiSession(stale)
	if !ok {
		t.Fatalf("expected stale fixture to parse")
	}
	if parsed.Status != "completed" || parsed.CurrentState != "done" {
		t.Fatalf("expected stale user leaf to be completed/done, got %#v", parsed)
	}
}

func TestParsePiSessionMissingHeaderIsSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	content := `{"type":"message","id":"c1","parentId":null,"timestamp":"2024-12-03T21:00:00.000Z","message":{"role":"user","content":"hello"}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, ok := parsePiSession(path); ok {
		t.Fatalf("expected file without a session header to be skipped")
	}
}

func TestDiscoverReadsSessionsDirectory(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "--home-user-projects-myapp--")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	fixture, err := os.ReadFile("testdata/session.jsonl")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sessionPath := filepath.Join(projectDir, "20241203_pi-session-uuid-1.jsonl")
	if err := os.WriteFile(sessionPath, fixture, 0o644); err != nil {
		t.Fatalf("write session fixture: %v", err)
	}

	result, err := New(config.Config{PiPath: root}).Discover()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(result.Sessions) != 1 {
		t.Fatalf("expected 1 discovered pi session, got %d", len(result.Sessions))
	}
	if result.Sessions[0].Project != "/home/user/projects/myapp" {
		t.Fatalf("unexpected project: %#v", result.Sessions[0])
	}
}
