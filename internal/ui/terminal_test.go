package ui

import (
	"os/exec"
	"slices"
	"strings"
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
	t.Setenv("PATH", t.TempDir())
	t.Setenv("TMUX", "")
	t.Setenv("ZELLIJ", "")
	cfg := config.Config{}
	if cmd := spawnSession(cfg, []string{"opencode"}); cmd != nil {
		t.Fatalf("expected nil, got %v", cmd)
	}
}

func TestSpawnSessionTmuxTerminalMode(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")
	cfg := config.Config{Terminal: "ghostty", TmuxMode: tmuxModeTerminal}
	cmd := spawnSession(cfg, []string{"claude"})
	if cmd == nil {
		t.Fatal("spawnSession returned nil")
	}
	want := []string{"ghostty", "-e", "sh", "-c", "tmux new-session 'claude'"}
	assertStringSlice(t, cmd.Args, want)
	for _, kv := range cmd.Env {
		if strings.HasPrefix(kv, "TMUX=") {
			t.Fatalf("TMUX leaked into env: %q", kv)
		}
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

func TestStartSessionReportsTarget(t *testing.T) {
	target, err := startSession(exec.Command("echo", "-P", "dash:3"))
	if err != nil {
		t.Fatal(err)
	}
	if target != "-P dash:3" {
		t.Fatalf("target = %q", target)
	}
	want := "Opened claude session in tmux window dash:3"
	if got := openedMessage("claude", "dash:3"); got != want {
		t.Fatalf("openedMessage = %q, want %q", got, want)
	}
}

func TestResolveTerminal(t *testing.T) {
	found := func(names ...string) func(string) (string, error) {
		return func(name string) (string, error) {
			if slices.Contains(names, name) {
				return "/usr/bin/" + name, nil
			}
			return "", exec.ErrNotFound
		}
	}
	tests := []struct {
		name       string
		configured string
		lookPath   func(string) (string, error)
		want       string
	}{
		{"configured wins", "wezterm", found("ghostty"), "wezterm"},
		{"detects first candidate", "", found("xterm", "alacritty"), "alacritty"},
		{"nothing found", "", found(), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveTerminal(tt.configured, tt.lookPath); got != tt.want {
				t.Fatalf("resolveTerminal = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTerminalExecArgs(t *testing.T) {
	argv := []string{"sh", "-c", "claude"}
	tests := []struct {
		terminal string
		want     []string
	}{
		{"alacritty", []string{"-e", "sh", "-c", "claude"}},
		{"/usr/bin/ghostty", []string{"-e", "sh", "-c", "claude"}},
		{"kitty", []string{"sh", "-c", "claude"}},
		{"foot", []string{"sh", "-c", "claude"}},
		{"wezterm", []string{"start", "--", "sh", "-c", "claude"}},
		{"gnome-terminal", []string{"--", "sh", "-c", "claude"}},
		{"xfce4-terminal", []string{"-x", "sh", "-c", "claude"}},
	}
	for _, tt := range tests {
		t.Run(tt.terminal, func(t *testing.T) {
			assertStringSlice(t, terminalExecArgs(tt.terminal, argv), tt.want)
		})
	}
}
