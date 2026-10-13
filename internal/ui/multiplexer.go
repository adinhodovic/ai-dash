package ui

import (
	"os"
	"strings"

	"github.com/samber/lo"
)

type multiplexerKind int

const (
	multiplexerNone multiplexerKind = iota
	multiplexerTmux
	multiplexerZellij
)

const (
	tmuxModeWindow     = "window"
	tmuxModeBackground = "background"
	tmuxModeTerminal   = "terminal"
)

type envLookup func(string) string

func osEnv(key string) string { return os.Getenv(key) }

// detectMultiplexer resolves the configured preference into a concrete
// multiplexer. "auto" / "" inspects $TMUX and $ZELLIJ; explicit names force
// that multiplexer; "off" disables the feature.
func detectMultiplexer(pref string, env envLookup) multiplexerKind {
	if env == nil {
		env = osEnv
	}
	switch strings.ToLower(strings.TrimSpace(pref)) {
	case "off", "none", "false", "disabled":
		return multiplexerNone
	case "tmux":
		return multiplexerTmux
	case "zellij":
		return multiplexerZellij
	case "", "auto":
		// fall through
	default:
		return multiplexerNone
	}
	if env("TMUX") != "" {
		return multiplexerTmux
	}
	if env("ZELLIJ") != "" {
		return multiplexerZellij
	}
	return multiplexerNone
}

// multiplexerCommand builds the (name, argv) pair to spawn a session inside
// the given multiplexer. Returns ("", nil) when no multiplexer is selected.
// tmuxMode only affects tmux; the terminal mode is handled by spawnSession.
func multiplexerCommand(
	kind multiplexerKind,
	args []string,
	tmuxMode string,
) (string, []string) {
	if kind == multiplexerNone || len(args) == 0 {
		return "", nil
	}
	shell := strings.Join(args, " ")
	switch kind {
	case multiplexerTmux:
		// tmux runs the argument through /bin/sh -c.
		if tmuxMode == tmuxModeBackground {
			// -P prints the new window's target so the status bar can name it.
			return "tmux", []string{
				"new-window", "-d", "-P", "-F", "#{session_name}:#{window_index}", shell,
			}
		}
		return "tmux", []string{"new-window", shell}
	case multiplexerZellij:
		// zellij run executes the argv directly, so wrap in sh -c to honor
		// the cd '...' && cmd shape produced by source ResumeArgs.
		return "zellij", []string{"run", "--close-on-exit", "--", "sh", "-c", shell}
	}
	return "", nil
}

// tmuxTerminalCommand opens a terminal emulator running the session in its own
// tmux session, so it gets a separate view from the one showing ai-dash.
func tmuxTerminalCommand(terminal string, args []string) (string, []string) {
	return terminalCommand(
		terminal,
		[]string{"tmux", "new-session", shellQuote(strings.Join(args, " "))},
	)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// withoutTmuxEnv drops the variables that make tmux refuse to start a session
// from inside another one.
func withoutTmuxEnv(env []string) []string {
	return lo.Reject(env, func(kv string, _ int) bool {
		return strings.HasPrefix(kv, "TMUX=") || strings.HasPrefix(kv, "TMUX_PANE=")
	})
}
