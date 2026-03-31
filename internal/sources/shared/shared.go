package shared

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/adin/ai-dash/internal/session"
)

type Source struct {
	Tool   string
	Kind   string
	Path   string
	Exists bool
	Note   string
}

type TranscriptFile struct {
	Tool    string
	Path    string
	Project string
	ModTime time.Time
}

type Result struct {
	Sources     []Source
	Transcripts []TranscriptFile
	Sessions    []session.Session
}

type Discovery struct {
	Sources     []Source
	Transcripts []TranscriptFile
	Sessions    []session.Session
}

type DiscoveryProvider interface {
	Name() string
	Discover() (Result, error)
}

type SessionProvider interface {
	DiscoveryProvider
	ImportSessions(Result) ([]session.Session, error)
	ResumeArgs(sessionID string) []string
	NewSessionArgs(projectDir string) []string
}

// SubagentClassifier is optionally implemented by sources that can identify
// parent-child relationships between sessions. Each tool has different
// conventions for subagent/child sessions (e.g. Claude uses a subagents/
// directory, OpenCode stores parent_id in its database).
type SubagentClassifier interface {
	ParentSessionID(s session.Session) string
}

func (d Discovery) ExistingSources() int {
	count := 0
	for _, source := range d.Sources {
		if source.Exists {
			count++
		}
	}
	return count
}

func (d Discovery) SummaryLines() []string {
	lines := make([]string, 0, len(d.Sources)+1)
	lines = append(
		lines,
		fmt.Sprintf(
			"%d source(s) present, %d transcript(s) discovered",
			d.ExistingSources(),
			len(d.Transcripts),
		),
	)
	for _, source := range d.Sources {
		status := "missing"
		if source.Exists {
			status = "present"
		}
		lines = append(
			lines,
			fmt.Sprintf("%s %s: %s (%s)", source.Tool, source.Kind, source.Path, status),
		)
	}
	return lines
}

func NewSource(tool, kind, path, note string) Source {
	_, err := os.Stat(path)
	return Source{Tool: tool, Kind: kind, Path: path, Exists: err == nil, Note: note}
}

func SortTranscripts(transcripts []TranscriptFile) {
	sort.Slice(transcripts, func(i, j int) bool {
		return transcripts[i].ModTime.After(transcripts[j].ModTime)
	})
}
