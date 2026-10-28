package sources

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/adinhodovic/ai-dash/internal/config"
	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources/shared"
)

// titleAgents is the order agents are tried in when rename_agent is unset.
var titleAgents = []string{"claude", "codex", "pi"}

const (
	titleHeadPrompts = 4
	titleTailPrompts = 2
	titlePromptRunes = 400
	titleMaxRunes    = 80
)

func provider(cfg config.Config, tool string) shared.SessionProvider {
	for _, p := range providers(cfg, nil) {
		if p.Name() == tool {
			return p
		}
	}
	return nil
}

// TitleAgent picks the agent that writes titles: rename_agent if set, else the
// first of titleAgents found on PATH.
func TitleAgent(cfg config.Config) (string, error) {
	if cfg.RenameAgent != "" {
		if _, ok := provider(cfg, cfg.RenameAgent).(shared.HeadlessRunner); !ok {
			return "", fmt.Errorf(
				"rename_agent %q can't title sessions; use one of %s",
				cfg.RenameAgent, strings.Join(titleAgents, ", "),
			)
		}
		if _, err := exec.LookPath(cfg.RenameAgent); err != nil {
			return "", fmt.Errorf("rename_agent %q not found on PATH", cfg.RenameAgent)
		}
		return cfg.RenameAgent, nil
	}
	for _, agent := range titleAgents {
		if _, err := exec.LookPath(agent); err == nil {
			return agent, nil
		}
	}
	return "", fmt.Errorf("no agent found on PATH to write titles (tried %s)",
		strings.Join(titleAgents, ", "))
}

// CanTitle reports whether s is a top-level session whose prompts can be read back.
func CanTitle(cfg config.Config, s session.Session) bool {
	_, ok := provider(cfg, s.Tool).(shared.PromptReader)
	return ok && s.ParentID == "" && s.TranscriptPath != ""
}

// Title asks agent for a short title summarizing what the user did in s.
func Title(
	ctx context.Context,
	cfg config.Config,
	agent string,
	s session.Session,
) (string, error) {
	reader, ok := provider(cfg, s.Tool).(shared.PromptReader)
	if !ok {
		return "", fmt.Errorf("can't read %s transcripts", s.Tool)
	}
	runner, ok := provider(cfg, agent).(shared.HeadlessRunner)
	if !ok {
		return "", fmt.Errorf("%s can't write titles", agent)
	}
	prompts, err := reader.UserPrompts(s)
	if err != nil {
		return "", fmt.Errorf("read transcript: %w", err)
	}
	if len(prompts) == 0 {
		return "", fmt.Errorf("no user messages in transcript")
	}

	args := runner.HeadlessArgs(titlePrompt(s, prompts))
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	// Outside the project, so the agent doesn't load its instructions or settings.
	cmd.Dir = os.TempDir()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", agent, err, lastLine(stderr.String()))
	}
	title := cleanTitle(string(out))
	if title == "" {
		return "", fmt.Errorf("%s returned no title", agent)
	}
	return title, nil
}

// titlePrompt shows the agent the first and last few user messages, each truncated.
func titlePrompt(s session.Session, prompts []string) string {
	var b strings.Builder
	b.WriteString("Write a title for a coding-agent session from the user's messages below. ")
	b.WriteString("At most 8 words. Name the project or topic first, then what was done. ")
	b.WriteString("No quotes, no trailing period. Reply with the title only.\n\n")
	fmt.Fprintf(&b, "Project: %s\nUser messages:\n", s.Project)
	for i, prompt := range prompts {
		if i == titleHeadPrompts && len(prompts) > titleHeadPrompts+titleTailPrompts {
			b.WriteString("- ...\n")
		}
		if i >= titleHeadPrompts && i < len(prompts)-titleTailPrompts {
			continue
		}
		fmt.Fprintf(
			&b,
			"- %s\n",
			truncateRunes(strings.Join(strings.Fields(prompt), " "), titlePromptRunes),
		)
	}
	return b.String()
}

// cleanTitle strips the wrapping models tend to add around a one-line answer.
//
//	cleanTitle("Title: \"Fix CI on PR #12.\"\n") -> "Fix CI on PR #12"
func cleanTitle(out string) string {
	title := lastLine(out)
	title = strings.TrimPrefix(title, "Title:")
	title = strings.Trim(title, " \t\"'`*")
	title = strings.TrimSuffix(title, ".")
	return truncateRunes(title, titleMaxRunes)
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
