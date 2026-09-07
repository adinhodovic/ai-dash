package layout

import "testing"

func TestContentHeight(t *testing.T) {
	tests := []struct {
		termH int
		want  int
	}{
		{24, 18},
		{10, 4},
		{3, 4},
		{0, 4},
	}
	for _, tt := range tests {
		got := ContentHeight(tt.termH)
		if got != tt.want {
			t.Errorf("ContentHeight(%d) = %d, want %d", tt.termH, got, tt.want)
		}
	}
}
