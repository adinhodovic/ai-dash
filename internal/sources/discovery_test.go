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

func TestDiscoverPiSessions(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "--home-user-projects-myapp--")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `{"type":"session","version":3,"id":"pi-uuid","timestamp":"2024-12-03T14:00:00.000Z","cwd":"/home/user/projects/myapp"}
{"type":"message","id":"a1","parentId":null,"timestamp":"2024-12-03T14:00:01.000Z","message":{"role":"user","content":"hello"}}
{"type":"message","id":"a2","parentId":"a1","timestamp":"2024-12-03T14:00:02.000Z","message":{"role":"assistant","content":[{"type":"text","text":"hi"}],"model":"claude-sonnet-4-5","stopReason":"stop"}}
`
	sessionPath := filepath.Join(projectDir, "20241203_pi-uuid.jsonl")
	if err := os.WriteFile(sessionPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write session: %v", err)
	}
	discovery, err := Discover(config.Config{PiPath: root})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}

	found := false
	for _, source := range discovery.Sources {
		if source.Tool == "pi" && source.Path == root && source.Exists {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected pi sessions directory source to be present, got %#v", discovery.Sources)
	}

	piSessionFound := false
	for _, s := range discovery.Sessions {
		if s.Tool == "pi" && s.Project == "/home/user/projects/myapp" {
			piSessionFound = true
		}
	}
	if !piSessionFound {
		t.Fatalf("expected a discovered pi session, got %#v", discovery.Sessions)
	}
}
