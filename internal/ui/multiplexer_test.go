package ui

import "testing"

func envFrom(values map[string]string) envLookup {
	return func(key string) string { return values[key] }
}

func TestDetectMultiplexerAuto(t *testing.T) {
	tests := []struct {
		name string
		pref string
		env  map[string]string
		want multiplexerKind
	}{
		{"auto with tmux", "auto", map[string]string{"TMUX": "/tmp/tmux"}, multiplexerTmux},
		{"empty with tmux", "", map[string]string{"TMUX": "/tmp/tmux"}, multiplexerTmux},
		{"auto with zellij", "auto", map[string]string{"ZELLIJ": "0"}, multiplexerZellij},
		{"auto with neither", "auto", map[string]string{}, multiplexerNone},
		{
			"auto prefers tmux when both",
			"auto",
			map[string]string{"TMUX": "x", "ZELLIJ": "0"},
			multiplexerTmux,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectMultiplexer(tt.pref, envFrom(tt.env))
			if got != tt.want {
				t.Fatalf("detectMultiplexer = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectMultiplexerExplicit(t *testing.T) {
	tests := []struct {
		pref string
		env  map[string]string
		want multiplexerKind
	}{
		{"tmux", map[string]string{}, multiplexerTmux},
		{"zellij", map[string]string{}, multiplexerZellij},
		{"TMUX", map[string]string{}, multiplexerTmux},
		{"off", map[string]string{"TMUX": "x"}, multiplexerNone},
		{"none", map[string]string{"ZELLIJ": "0"}, multiplexerNone},
		{"unknown-value", map[string]string{"TMUX": "x"}, multiplexerNone},
	}
	for _, tt := range tests {
		t.Run(tt.pref, func(t *testing.T) {
			got := detectMultiplexer(tt.pref, envFrom(tt.env))
			if got != tt.want {
				t.Fatalf("detectMultiplexer(%q) = %v, want %v", tt.pref, got, tt.want)
			}
		})
	}
}

func TestMultiplexerCommandTmux(t *testing.T) {
	name, args := multiplexerCommand(
		multiplexerTmux,
		[]string{"cd", "'/p'", "&&", "claude", "-c", "'abc'"},
	)
	if name != "tmux" {
		t.Fatalf("name = %q, want tmux", name)
	}
	want := []string{"new-window", "cd '/p' && claude -c 'abc'"}
	assertStringSlice(t, args, want)
}

func TestMultiplexerCommandZellij(t *testing.T) {
	name, args := multiplexerCommand(
		multiplexerZellij,
		[]string{"cd", "'/p'", "&&", "opencode", "-s", "'abc'"},
	)
	if name != "zellij" {
		t.Fatalf("name = %q, want zellij", name)
	}
	want := []string{"run", "--close-on-exit", "--", "sh", "-c", "cd '/p' && opencode -s 'abc'"}
	assertStringSlice(t, args, want)
}

func TestMultiplexerCommandNone(t *testing.T) {
	name, args := multiplexerCommand(multiplexerNone, []string{"opencode"})
	if name != "" || args != nil {
		t.Fatalf("got (%q, %v), want empty", name, args)
	}
}

func TestMultiplexerCommandEmptyArgs(t *testing.T) {
	name, args := multiplexerCommand(multiplexerTmux, nil)
	if name != "" || args != nil {
		t.Fatalf("got (%q, %v), want empty", name, args)
	}
}
