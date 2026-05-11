package ui

import (
	"testing"

	"github.com/adinhodovic/ai-dash/internal/config"
)

func TestSpawnSessionPrefersMultiplexer(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")
	t.Setenv("ZELLIJ", "")
	cfg := config.Config{Terminal: "ghostty"}
	cmd := spawnSession(cfg, []string{"opencode", "-s", "abc"})
	if cmd == nil {
		t.Fatal("spawnSession returned nil")
	}
	if cmd.Path == "" || (cmd.Args[0] != "tmux" && !endsWith(cmd.Path, "tmux")) {
		t.Fatalf("expected tmux invocation, got path=%q args=%v", cmd.Path, cmd.Args)
	}
}

func TestSpawnSessionFallsBackToTerminal(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("ZELLIJ", "")
	cfg := config.Config{Terminal: "ghostty"}
	cmd := spawnSession(cfg, []string{"opencode", "-s", "abc"})
	if cmd == nil {
		t.Fatal("spawnSession returned nil")
	}
	if cmd.Args[0] != "ghostty" && !endsWith(cmd.Path, "ghostty") {
		t.Fatalf("expected ghostty invocation, got args=%v", cmd.Args)
	}
}

func TestSpawnSessionMultiplexerOff(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")
	cfg := config.Config{Terminal: "ghostty", Multiplexer: "off"}
	cmd := spawnSession(cfg, []string{"opencode"})
	if cmd == nil {
		t.Fatal("spawnSession returned nil")
	}
	if cmd.Args[0] != "ghostty" && !endsWith(cmd.Path, "ghostty") {
		t.Fatalf("expected ghostty (multiplexer off), got args=%v", cmd.Args)
	}
}

func TestSpawnSessionNoTerminalNoMultiplexer(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("ZELLIJ", "")
	cfg := config.Config{}
	if cmd := spawnSession(cfg, []string{"opencode"}); cmd != nil {
		t.Fatalf("expected nil, got %v", cmd)
	}
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func TestTerminalCommand(t *testing.T) {
	name, args := terminalCommand("ghostty", []string{"opencode", "-s", "abc"})
	if name != "ghostty" {
		t.Fatalf("name = %q, want ghostty", name)
	}
	want := []string{"-e", "sh", "-c", "opencode -s abc"}
	assertStringSlice(t, args, want)
}

func TestTerminalCommandEmpty(t *testing.T) {
	name, args := terminalCommand("", []string{"opencode"})
	if name != "" {
		t.Fatalf("name = %q, want empty", name)
	}
	if args != nil {
		t.Fatalf("args = %v, want nil", args)
	}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%q)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
