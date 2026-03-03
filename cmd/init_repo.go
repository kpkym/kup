package cmd

import (
	"github.com/kpkym/kup/internal/runner"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:               "init <profile>",
	Short:             "Initialize a new restic repository",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: profileCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := cfg.GetRepos(args[0])
		if err != nil {
			return err
		}

		return runner.RunResticForEachRepo(cfg.Global, repos, []string{"init"})
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
