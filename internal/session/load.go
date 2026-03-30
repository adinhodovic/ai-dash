package session

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadFile(path string) ([]Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sf File
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("decode session file %s: %w", path, err)
	}

	Sort(sf.Sessions)
	return sf.Sessions, nil
}

func LoadDefaultSessions() ([]Session, error) {
	return LoadFile("sessions.json")
}

func LoadSampleSessions() ([]Session, error) {
	return LoadFile("sessions.sample.json")
}
