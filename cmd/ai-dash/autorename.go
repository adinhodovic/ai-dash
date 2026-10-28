package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/adinhodovic/ai-dash/internal/config"
	"github.com/adinhodovic/ai-dash/internal/session"
	"github.com/adinhodovic/ai-dash/internal/sources"
)

const (
	autorenameWorkers = 4
	autorenameTimeout = 2 * time.Minute
)

type autorenameResult struct {
	session session.Session
	title   string
	err     error
}

func autorenameCommand() *cobra.Command {
	var (
		limit  int
		agent  string
		force  bool
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "autorename",
		Short: "Title the most recent sessions with an AI agent",
		Long: "Titles the most recently active sessions by asking an agent CLI to summarize " +
			"what the user typed. Sessions you renamed yourself are skipped unless --force.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Init()
			cfg := config.Load()
			if agent != "" {
				cfg.RenameAgent = agent
			}
			return runAutorename(cmd, cfg, limit, force, dryRun)
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "n", 30, "how many recent sessions to title")
	cmd.Flags().StringVar(&agent, "agent", "", "agent to write titles: claude, codex or pi")
	cmd.Flags().BoolVar(&force, "force", false, "retitle sessions that were already renamed")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print titles without saving them")
	return cmd
}

func runAutorename(cmd *cobra.Command, cfg config.Config, limit int, force, dryRun bool) error {
	agent, err := sources.TitleAgent(cfg)
	if err != nil {
		return err
	}
	discovery, err := sources.Discover(cfg)
	if err != nil {
		return err
	}
	renamesPath := session.DefaultRenamesPath()
	renames, err := session.LoadRenames(renamesPath)
	if err != nil {
		return fmt.Errorf("load session renames: %w", err)
	}

	sessions := discovery.Sessions
	session.SortBy(sessions, session.SortUpdated, true)
	var targets []session.Session
	for _, s := range sessions {
		if len(targets) == limit {
			break
		}
		if !sources.CanTitle(cfg, s) {
			continue
		}
		if _, renamed := renames[session.RenameKey(s)]; renamed && !force {
			continue
		}
		targets = append(targets, s)
	}
	out := cmd.OutOrStdout()
	if len(targets) == 0 {
		_, _ = fmt.Fprintln(out, "No sessions to rename")
		return nil
	}
	_, _ = fmt.Fprintf(out, "Titling %d sessions with %s\n", len(targets), agent)

	failed := 0
	for r := range titleAll(cmd.Context(), cfg, agent, targets) {
		key := session.RenameKey(r.session)
		if r.err != nil {
			failed++
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", key, r.err)
			continue
		}
		if !dryRun {
			if _, err := session.SetRename(renamesPath, key, r.title); err != nil {
				return fmt.Errorf("save rename: %w", err)
			}
		}
		_, _ = fmt.Fprintf(out, "%s: %s\n", key, r.title)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d sessions not renamed", failed, len(targets))
	}
	return nil
}

// titleAll titles sessions concurrently, sending results as they finish.
func titleAll(
	ctx context.Context,
	cfg config.Config,
	agent string,
	targets []session.Session,
) <-chan autorenameResult {
	jobs := make(chan session.Session)
	results := make(chan autorenameResult)
	var wg sync.WaitGroup
	for range autorenameWorkers {
		wg.Go(func() {
			for s := range jobs {
				callCtx, cancel := context.WithTimeout(ctx, autorenameTimeout)
				title, err := sources.Title(callCtx, cfg, agent, s)
				cancel()
				results <- autorenameResult{session: s, title: title, err: err}
			}
		})
	}
	go func() {
		for _, s := range targets {
			jobs <- s
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	return results
}
