package session

import "testing"

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		name string
		s    Session
		want string
	}{
		{"normalized current state", Session{CurrentState: "tool call"}, "tool call"},
		{"active waiting", Session{Status: "active", Meta: map[string]string{"stop_reason": "end_turn"}}, "waiting"},
		{"active tool call", Session{Status: "active", Meta: map[string]string{"stop_reason": "tool_use"}}, "tool call"},
		{"completed", Session{Status: "completed"}, "done"},
		{"aborted", Session{Status: "aborted"}, "aborted"},
		{"unknown", Session{}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StatusLabel(tt.s); got != tt.want {
				t.Fatalf("StatusLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
