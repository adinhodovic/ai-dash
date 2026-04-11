package sources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adinhodovic/ai-dash/internal/config"
)

func TestDiscoverUsesEnvOverrides(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "custom-config.toml")
	sessionsDir := filepath.Join(root, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("model = \"gpt-5.4\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	discovery, err := Discover(config.Config{CodexPath: configPath})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}

	found := false
	for _, source := range discovery.Sources {
		if source.Tool == "codex" && source.Path == sessionsDir && source.Exists {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected codex sessions directory source to be present")
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
	discovery, err := Discover(config.Config{ClaudePath: root})
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
