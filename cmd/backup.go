package cmd

import (
	"fmt"
	"os"

	"github.com/kpkym/kup/internal/runner"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:               "backup <profile>",
	Short:             "Run restic backup for a profile",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: profileCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		profile, err := cfg.GetProfile(profileName)
		if err != nil {
			return err
		}

		if len(profile.Paths) == 0 {
			return fmt.Errorf("profile %q has no paths configured", profileName)
		}

		// Write paths to temp file for --files-from
		tmpFile, err := os.CreateTemp("", "kup-files-from-*.txt")
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		for _, p := range profile.Paths {
			fmt.Fprintln(tmpFile, p)
		}
		tmpFile.Close()

		// Build restic args
		resticArgs := []string{
			"backup",
			"--files-from", tmpFile.Name(),
			"--group-by", "paths",
		}

		if cfg.Global.ResticExcludeFile != "" {
			resticArgs = append(resticArgs, "--exclude-file", cfg.Global.ResticExcludeFile)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			resticArgs = append(resticArgs, "--dry-run")
		}

		return runner.RunResticForEachRepo(cfg.Global, profile.Repos, resticArgs)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
}
