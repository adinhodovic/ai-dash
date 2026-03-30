package presets

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Preset struct {
	Tool    string `json:"tool"`
	Status  string `json:"status"`
	Project string `json:"project"`
	Search  string `json:"search"`
}

type Store struct {
	Projects map[string]Preset `json:"projects"`
}

func DefaultPath() string {
	if override := os.Getenv("AIDASH_PRESETS_PATH"); override != "" {
		return override
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ".ai-dash-presets.json"
	}
	return filepath.Join(configDir, "ai-dash", "presets.json")
}

func Load() (Store, error) {
	path := DefaultPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Store{Projects: map[string]Preset{}}, nil
		}
		return Store{}, err
	}
	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return Store{}, err
	}
	if store.Projects == nil {
		store.Projects = map[string]Preset{}
	}
	return store, nil
}

func Save(store Store) error {
	path := DefaultPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
