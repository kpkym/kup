package cmd

import (
	"github.com/kpkym/kup/internal/runner"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:               "prune <profile>...",
	Short:             "Remove unreferenced data from repository",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: profileCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := resolveRepos(args)
		if err != nil {
			return err
		}

		return runner.RunResticForEachRepo(cfg.Global, repos, []string{"prune"})
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}
