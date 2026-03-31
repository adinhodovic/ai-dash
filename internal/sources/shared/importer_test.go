package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverCandidateFilesWithPatterns(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		filepath.Join(root, "session-1.json"),
		filepath.Join(root, "history.jsonl"),
		filepath.Join(root, "config.json"),
	}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	matches, err := DiscoverCandidateFiles(root)
	if err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 candidate files, got %#v", matches)
	}
}
