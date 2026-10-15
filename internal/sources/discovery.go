package sources

import (
	"fmt"

	"github.com/adinhodovic/ai-dash/internal/config"
	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources/claude"
	"github.com/adinhodovic/ai-dash/internal/sources/codex"
	"github.com/adinhodovic/ai-dash/internal/sources/opencode"
	"github.com/adinhodovic/ai-dash/internal/sources/pi"
	"github.com/adinhodovic/ai-dash/internal/sources/shared"
)

type (
	Discovery      = shared.Discovery
	Source         = shared.Source
	TranscriptFile = shared.TranscriptFile
)

func providers(cfg config.Config) []shared.SessionProvider {
	return providersWithCache(cfg, nil)
}

func providersWithCache(cfg config.Config, cache *shared.Cache) []shared.SessionProvider {
	return []shared.SessionProvider{
		opencode.New(cfg),
		codex.New(cfg).WithCache(cache),
		claude.New(cfg).WithCache(cache),
		pi.New(cfg).WithCache(cache),
	}
}

func NewSessionArgs(cfg config.Config, tool, projectDir string) []string {
	for _, p := range providers(cfg) {
		if p.Name() == tool {
			return p.NewSessionArgs(projectDir)
		}
	}
	return nil
}

func ResumeArgs(cfg config.Config, tool, sessionID, projectDir string) []string {
	for _, p := range providers(cfg) {
		if p.Name() == tool {
			return p.ResumeArgs(sessionID, projectDir)
		}
	}
	return nil
}

func Discover(cfg config.Config) (Discovery, error) {
	// A cache that cannot be opened falls back to parsing every file.
	// The dashboard still has to start.
	cache, err := shared.Open(shared.DefaultPath())
	if err != nil {
		cache = nil
	} else {
		defer func() { _ = cache.Close() }()
	}
	providers := providersWithCache(cfg, cache)
	discovery := Discovery{}
	for _, provider := range providers {
		result, err := provider.Discover()
		discovery.Sources = append(discovery.Sources, result.Sources...)
		discovery.Transcripts = append(discovery.Transcripts, result.Transcripts...)
		classifySessions(provider, result.Sessions)
		discovery.Sessions = append(discovery.Sessions, result.Sessions...)
		if err != nil {
			return discovery, fmt.Errorf("discover %s sources: %w", provider.Name(), err)
		}
	}

	shared.SortTranscripts(discovery.Transcripts)
	session.Sort(discovery.Sessions)
	return discovery, nil
}

func classifySessions(provider shared.SessionProvider, sessions []session.Session) {
	classifier, ok := provider.(shared.SubagentClassifier)
	if !ok {
		return
	}
	for i := range sessions {
		if sessions[i].ParentID != "" {
			continue
		}
		if parentID := classifier.ParentSessionID(sessions[i]); parentID != "" {
			sessions[i].ParentID = parentID
			sessions[i].Tags = append(sessions[i].Tags, "subagent")
		}
	}
}
