package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kpkym/kup/internal/runner"
	"github.com/spf13/cobra"
)

var forgetCmd = &cobra.Command{
	Use:               "forget <profile>...",
	Short:             "Remove snapshots according to a policy",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: profileCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := resolveRepos(args)
		if err != nil {
			return err
		}

		keepLast, _ := cmd.Flags().GetInt("keep-last")
		if keepLast <= 0 {
			return fmt.Errorf("--keep-last is required and must be > 0")
		}

		resticArgs := []string{"forget", "--keep-last", fmt.Sprintf("%d", keepLast)}

		groupBy, _ := cmd.Flags().GetString("group-by")
		if groupBy == "" {
			fmt.Print("--group-by is empty, forget will apply to all snapshots without grouping. Continue? [y/N]: ")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			if !strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
				return fmt.Errorf("aborted")
			}
		}
		resticArgs = append(resticArgs, "--group-by", groupBy)

		prune, _ := cmd.Flags().GetBool("prune")
		if prune {
			resticArgs = append(resticArgs, "--prune")
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			resticArgs = append(resticArgs, "--dry-run")
		}

		return runner.RunResticForEachRepo(cfg.Global, repos, resticArgs)
	},
}

func init() {
	forgetCmd.Flags().Int("keep-last", 0, "number of latest snapshots to keep (required)")
	forgetCmd.Flags().String("group-by", "paths", "group snapshots by")
	forgetCmd.Flags().Bool("prune", false, "prune after forgetting")
	forgetCmd.Flags().Bool("dry-run", false, "do not delete, just print what would be done")
	rootCmd.AddCommand(forgetCmd)
}
