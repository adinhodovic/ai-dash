package ui

import "testing"

func TestFollowCursor(t *testing.T) {
	tests := []struct {
		name           string
		cursor         int
		count          int
		viewportHeight int
		yOffset        int
		want           int
	}{
		{"already visible, no scroll", 2, 10, 12, 0, 0},
		{"cursor above window scrolls up to its top", 1, 10, 4, 8, 4},
		{"cursor below window scrolls down to its bottom", 5, 10, 4, 0, 19},
		{"first session, top of window", 0, 10, 4, 2, 0},
		{"last session, bottom of window", 9, 10, 4, 0, 35},
		{"empty list", 0, 0, 4, 3, 0},
		{"zero-height viewport", 3, 10, 0, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := followCursor(tt.cursor, tt.count, tt.viewportHeight, tt.yOffset)
			if got != tt.want {
				t.Fatalf(
					"followCursor(cursor=%d, count=%d, viewportHeight=%d, yOffset=%d) = %d, want %d",
					tt.cursor, tt.count, tt.viewportHeight, tt.yOffset, got, tt.want,
				)
			}
		})
	}
}
