package ui

import (
	"os"
	"strings"
)

type multiplexerKind int

const (
	multiplexerNone multiplexerKind = iota
	multiplexerTmux
	multiplexerZellij
)

func (k multiplexerKind) String() string {
	switch k {
	case multiplexerTmux:
		return "tmux"
	case multiplexerZellij:
		return "zellij"
	}
	return ""
}

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
func multiplexerCommand(kind multiplexerKind, args []string) (string, []string) {
	if kind == multiplexerNone || len(args) == 0 {
		return "", nil
	}
	shell := strings.Join(args, " ")
	switch kind {
	case multiplexerTmux:
		// tmux runs the argument through /bin/sh -c.
		return "tmux", []string{"new-window", shell}
	case multiplexerZellij:
		// zellij run executes the argv directly, so wrap in sh -c to honor
		// the cd '...' && cmd shape produced by source ResumeArgs.
		return "zellij", []string{"run", "--close-on-exit", "--", "sh", "-c", shell}
	}
	return "", nil
}
