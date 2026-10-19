package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/adinhodovic/ai-dash/internal/update"
)

func newUpdateCommand() *cobra.Command {
	var check, force bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update ai-dash to the latest release",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd, check, force)
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Only report whether an update is available")
	cmd.Flags().
		BoolVar(&force, "force", false, "Install the latest release even over a source build")
	return cmd
}

func runUpdate(cmd *cobra.Command, check, force bool) error {
	out := cmd.OutOrStdout()
	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	client := update.NewClient()
	rel, err := client.Latest(ctx)
	if err != nil {
		return fmt.Errorf("check latest release: %w", err)
	}
	if !update.Newer(aiDashVersion, rel.Tag) && !force {
		_, _ = fmt.Fprintf(out, "ai-dash %s is up to date\n", aiDashVersion)
		return nil
	}
	if !update.IsReleaseVersion(aiDashVersion) && !force {
		_, _ = fmt.Fprintf(
			out,
			"ai-dash %s was built from source. Update it with:\n"+
				"  go install github.com/%s/cmd/ai-dash@latest\n"+
				"or run `ai-dash update --force` to replace it with the %s release binary.\n",
			aiDashVersion, update.Repo, rel.Tag,
		)
		return nil
	}

	if check {
		_, _ = fmt.Fprintf(out, "Update available: %s -> %s\n", aiDashVersion, rel.Tag)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}

	name := update.AssetName(runtime.GOOS, runtime.GOARCH)
	_, _ = fmt.Fprintf(out, "Downloading %s %s...\n", name, rel.Tag)
	data, err := client.Download(ctx, rel, name)
	if errors.Is(err, update.ErrNoChecksum) {
		_, _ = fmt.Fprintln(out, "Warning: release has no checksum, skipping verification")
	} else if err != nil {
		return err
	}

	if err := update.Replace(exe, data); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("no permission to replace %s; rerun with sudo: %w", exe, err)
		}
		return fmt.Errorf("replace %s: %w", exe, err)
	}
	_, _ = fmt.Fprintf(out, "Updated ai-dash %s -> %s (%s)\n", aiDashVersion, rel.Tag, exe)
	return nil
}
