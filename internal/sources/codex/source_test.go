package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adin/ai-dash/internal/config"
)

func TestDiscoverUsesEnvOverrides(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	sessionsDir := filepath.Join(root, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("model = \"gpt-5.4\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	result, err := New(config.Config{CodexPath: configPath}).Discover()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}

	found := false
	for _, source := range result.Sources {
		if source.Path == sessionsDir && source.Exists {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected sessions directory source, got %#v", result.Sources)
	}
}

func TestParseCodexSessionFromFixture(t *testing.T) {
	parsed, ok := parseCodexSession("testdata/session.jsonl")
	if !ok {
		t.Fatalf("expected fixture to parse")
	}
	if parsed.ID != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Fatalf("unexpected id: %#v", parsed)
	}
	if parsed.Project != "api" {
		t.Fatalf("unexpected project: %#v", parsed)
	}
	if parsed.Model != "gpt-5.2-codex" {
		t.Fatalf("unexpected model: %#v", parsed)
	}
	if parsed.Status != "aborted" {
		t.Fatalf("unexpected status: %#v", parsed)
	}
	if parsed.Summary != "Ship the rollout cleanup change and exit." {
		t.Fatalf("unexpected summary: %#v", parsed)
	}
}

func TestDiscoverReadsSessionsDirectory(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	if err := os.WriteFile(configPath, []byte("model = \"gpt-5.4\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	sessionsDir := filepath.Join(root, "sessions", "2026", "03", "17")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions dir: %v", err)
	}
	fixture, err := os.ReadFile("testdata/session.jsonl")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	rollout := filepath.Join(sessionsDir, "rollout.jsonl")
	if err := os.WriteFile(rollout, fixture, 0o644); err != nil {
		t.Fatalf("write session fixture: %v", err)
	}
	result, err := New(config.Config{CodexPath: configPath}).Discover()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(result.Sessions) == 0 {
		t.Fatalf("expected discovered codex sessions")
	}
}
