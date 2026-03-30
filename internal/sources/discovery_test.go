package sources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adin/ai-dash/internal/session"
)

func TestDiscoverUsesEnvOverrides(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "custom-config.toml")
	if err := os.WriteFile(configPath, []byte("model = \"gpt-5.4\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("AIDASH_CODEX_CONFIG", configPath)

	discovery, err := Discover()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}

	found := false
	for _, source := range discovery.Sources {
		if source.Tool == "codex" && source.Path == configPath && source.Exists {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected codex env override source to be present")
	}
}

func TestDiscoverClaudeTranscripts(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "repo-a")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	transcriptPath := filepath.Join(projectDir, "transcript.jsonl")
	if err := os.WriteFile(transcriptPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	t.Setenv("AIDASH_CLAUDE_PROJECTS_DIR", root)

	discovery, err := Discover()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(discovery.Transcripts) != 1 {
		t.Fatalf("expected 1 transcript, got %d", len(discovery.Transcripts))
	}
	if discovery.Transcripts[0].Project != "repo-a" {
		t.Fatalf("expected project repo-a, got %q", discovery.Transcripts[0].Project)
	}
}

func TestLoadDefaultSessionsPrefersDiscoveredSessionsOverSample(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "sessions.sample.json"), []byte(`{"sessions":[{"id":"sample","tool":"opencode","project":"demo","status":"completed","started_at":"2026-03-29T12:00:00Z"}]}`), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	discovery := Discovery{Sessions: []session.Session{{ID: "real", Tool: "codex", Project: "repo", Status: "completed"}}}
	sessions, err := LoadDefaultSessions(discovery)
	if err != nil {
		t.Fatalf("load default sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "real" {
		t.Fatalf("expected discovered session to win, got %#v", sessions)
	}
}
