package shared

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DiscoverCandidateFiles(root string) ([]string, error) {
	return DiscoverCandidateFilesWithPatterns(root, nil)
}

func DiscoverCandidateFilesWithPatterns(root string, extraPatterns []string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	var matches []string
	patterns := append(
		[]string{"session", "history", "transcript", "chat", "conversation", "messages"},
		extraPatterns...)
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		base := strings.ToLower(filepath.Base(path))
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".json" && ext != ".jsonl" {
			return nil
		}
		if strings.Contains(base, "config") || strings.Contains(base, "settings") ||
			strings.Contains(base, "theme") ||
			strings.Contains(base, "keybind") {
			return nil
		}
		if ext == ".jsonl" || containsAny(base, patterns) {
			matches = append(matches, path)
		}
		return nil
	})
	sort.Strings(matches)
	return matches, err
}

func containsAny(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(value, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
