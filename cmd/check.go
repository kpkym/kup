package cmd

import (
	"github.com/kpkym/kup/internal/runner"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:               "check <profile>...",
	Short:             "Check repository integrity",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: profileCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := resolveRepos(args)
		if err != nil {
			return err
		}

		resticArgs := []string{"check"}

		readData, _ := cmd.Flags().GetBool("read-data")
		if readData {
			resticArgs = append(resticArgs, "--read-data")
		}

		return runner.RunResticForEachRepo(cfg.Global, repos, resticArgs)
	},
}

func init() {
	checkCmd.Flags().Bool("read-data", false, "verify data integrity (slow)")
	rootCmd.AddCommand(checkCmd)
}
