package cmd

import (
	"fmt"

	"github.com/kpkym/kup/internal/runner"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore <snapshot-id>",
	Short: "Restore a snapshot to a target directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		if repo == "" {
			return fmt.Errorf("--repo is required for restore")
		}

		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			return fmt.Errorf("--target is required for restore")
		}

		snapshotID := args[0]
		resticArgs := []string{"restore", snapshotID, "--target", target}

		return runner.RunRestic(cfg.Global, repo, resticArgs)
	},
}

func init() {
	restoreCmd.Flags().String("repo", "", "repo URI (required)")
	restoreCmd.Flags().String("target", "", "restore target directory (required)")
	restoreCmd.MarkFlagRequired("repo")
	restoreCmd.MarkFlagRequired("target")
	restoreCmd.RegisterFlagCompletionFunc("repo", repoCompletionFunc)
	rootCmd.AddCommand(restoreCmd)
}
