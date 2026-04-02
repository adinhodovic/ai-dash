package ui

import "testing"

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
