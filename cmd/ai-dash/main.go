package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/adin/ai-dash/internal/config"
	"github.com/adin/ai-dash/internal/sources"
	"github.com/adin/ai-dash/internal/ui"
	"github.com/adin/ai-dash/internal/ui/icon"
)

var (
	buildTimestamp = "dev"
	aiDashVersion  = "dev"
)

func main() {
	root := &cobra.Command{
		Use:     "ai-dash",
		Short:   "TUI dashboard for AI coding sessions",
		Version: aiDashVersion,
		RunE:    runDashboard,
	}

	schema := &cobra.Command{
		Use:   "schema",
		Short: "Print JSON Schema for the config file",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.GenerateSchema())
		},
	}

	root.AddCommand(schema)
	root.CompletionOptions.HiddenDefaultCmd = true

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runDashboard(cmd *cobra.Command, args []string) error {
	config.Init()
	cfg := config.Load()
	icon.Init(cfg.NerdFont)

	discovery, discoveryErr := sources.Discover(cfg)
	sessions, err := sources.ImportSessions(discovery)
	if err == nil && len(sessions) == 0 {
		err = fmt.Errorf("no sessions found from configured providers")
	}
	if err == nil && discoveryErr != nil {
		err = discoveryErr
	}

	m := ui.NewModel(ui.Options{
		Sessions:       sessions,
		Discovery:      discovery,
		Config:         cfg,
		Err:            err,
		Version:        aiDashVersion,
		BuildTimestamp: buildTimestamp,
	})

	p := tea.NewProgram(m)
	_, runErr := p.Run()
	return runErr
}
