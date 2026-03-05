package cmd

import (
	"github.com/kpkym/kup/internal/runner"
	"github.com/spf13/cobra"
)

var snapshotsCmd = &cobra.Command{
	Use:               "snapshots <profile>...",
	Short:             "List snapshots",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: profileCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := resolveRepos(args)
		if err != nil {
			return err
		}

		resticArgs := []string{"snapshots", "--group-by", "paths"}
		return runner.RunResticForEachRepo(cfg.Global, repos, resticArgs)
	},
}

func init() {
	rootCmd.AddCommand(snapshotsCmd)
}
