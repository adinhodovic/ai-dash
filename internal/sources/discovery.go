package sources

import (
	"fmt"

	"github.com/adin/ai-dash/internal/session"
	"github.com/adin/ai-dash/internal/sources/claude"
	"github.com/adin/ai-dash/internal/sources/codex"
	"github.com/adin/ai-dash/internal/sources/opencode"
	"github.com/adin/ai-dash/internal/sources/shared"
)

type Discovery = shared.Discovery
type Source = shared.Source
type TranscriptFile = shared.TranscriptFile

func providers() []shared.SessionProvider {
	return []shared.SessionProvider{
		opencode.New(),
		codex.New(),
		claude.New(),
	}
}

func ResumeArgs(tool, sessionID string) []string {
	for _, p := range providers() {
		if p.Name() == tool {
			return p.ResumeArgs(sessionID)
		}
	}
	return nil
}

func Discover() (Discovery, error) {
	providers := providers()
	discovery := Discovery{}
	for _, provider := range providers {
		result, err := provider.Discover()
		discovery.Sources = append(discovery.Sources, result.Sources...)
		discovery.Transcripts = append(discovery.Transcripts, result.Transcripts...)
		discovery.Sessions = append(discovery.Sessions, result.Sessions...)
		if err != nil {
			return discovery, fmt.Errorf("discover %s sources: %w", provider.Name(), err)
		}
	}

	shared.SortTranscripts(discovery.Transcripts)
	session.Sort(discovery.Sessions)
	return discovery, nil
}

func LoadDefaultSessions(discovery Discovery) ([]session.Session, error) {
	imported, importErr := ImportSessions(discovery)
	if importErr == nil && len(imported) > 0 {
		return imported, nil
	}

	sessions, err := session.LoadDefaultSessions()
	if err == nil {
		return sessions, nil
	}

	sample, sampleErr := session.LoadSampleSessions()
	if sampleErr == nil {
		return sample, nil
	}

	if importErr != nil {
		return nil, importErr
	}
	return nil, err
}

func ImportSessions(discovery Discovery) ([]session.Session, error) {
	if len(discovery.Sessions) > 0 {
		imported := append([]session.Session(nil), discovery.Sessions...)
		session.Sort(imported)
		return imported, nil
	}

	providers := providers()
	imported := make([]session.Session, 0)
	for _, provider := range providers {
		result, err := provider.Discover()
		if err != nil {
			return imported, fmt.Errorf("discover %s for import: %w", provider.Name(), err)
		}
		sessions, err := provider.ImportSessions(result)
		if err != nil {
			return imported, fmt.Errorf("import %s sessions: %w", provider.Name(), err)
		}
		imported = append(imported, sessions...)
	}
	session.Sort(imported)
	return imported, nil
}
