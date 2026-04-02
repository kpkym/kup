package cmd

import (
	"github.com/kpkym/kup/internal/runner"
	"github.com/spf13/cobra"
)

var mountCmd = &cobra.Command{
	Use:   "mount --repo <repo> <mountpoint>",
	Short: "Mount a repository as a FUSE filesystem",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")

		mountpoint := args[0]
		resticArgs := []string{"mount", mountpoint}

		return runner.RunRestic(cfg.Global, repo, resticArgs)
	},
}

func init() {
	mountCmd.Flags().String("repo", "", "repo URI (required)")
	mountCmd.MarkFlagRequired("repo")
	mountCmd.RegisterFlagCompletionFunc("repo", repoCompletionFunc)
	rootCmd.AddCommand(mountCmd)
}
