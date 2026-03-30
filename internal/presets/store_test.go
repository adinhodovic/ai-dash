package presets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsEmptyStore(t *testing.T) {
	t.Setenv("AIDASH_PRESETS_PATH", filepath.Join(t.TempDir(), "presets.json"))
	store, err := Load()
	if err != nil {
		t.Fatalf("load presets: %v", err)
	}
	if len(store.Projects) != 0 {
		t.Fatalf("expected empty store, got %#v", store)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "presets.json")
	t.Setenv("AIDASH_PRESETS_PATH", path)
	want := Store{Projects: map[string]Preset{"demo": {Tool: "claude", Search: "auth"}}}
	if err := Save(want); err != nil {
		t.Fatalf("save presets: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat saved presets: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("load presets: %v", err)
	}
	if got.Projects["demo"].Tool != "claude" || got.Projects["demo"].Search != "auth" {
		t.Fatalf("unexpected loaded preset: %#v", got)
	}
}
