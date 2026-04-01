package shared

import (
	"fmt"
	"os"
	"slices"
	"strings"
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
	ResumeArgs(sessionID, projectDir string) []string
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

// ShellQuote wraps s in single quotes, escaping any embedded single quotes.
// This prevents shell metacharacter injection when the value is interpolated
// into a command string passed to sh -c.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func SortTranscripts(transcripts []TranscriptFile) {
	slices.SortFunc(transcripts, func(a, b TranscriptFile) int {
		return b.ModTime.Compare(a.ModTime)
	})
}
