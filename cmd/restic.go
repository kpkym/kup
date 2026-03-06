package cmd

import (
	"github.com/kpkym/kup/internal/runner"
	"github.com/spf13/cobra"
)

var resticCmd = &cobra.Command{
	Use:                "restic -- [restic args...]",
	Short:              "Run restic directly with kup environment",
	DisableFlagParsing: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 && args[len(args)-1] == "--repo" {
			return repoCompletionFunc(cmd, args, toComplete)
		}

		// Strip --repo <val> before delegating to restic
		var resticArgs []string
		for i := 0; i < len(args); i++ {
			if args[i] == "--repo" {
				i++
				continue
			}
			resticArgs = append(resticArgs, args[i])
		}
		return passthroughCompletion("restic", append(resticArgs, toComplete))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle --help/-h explicitly since DisableFlagParsing prevents cobra from intercepting it
		for _, arg := range args {
			if arg == "--help" || arg == "-h" {
				return cmd.Help()
			}
		}

		// Find "--" separator and extract --repo before it
		var repo string
		var resticArgs []string
		dashDash := false

		for i := 0; i < len(args); i++ {
			if args[i] == "--" {
				dashDash = true
				resticArgs = args[i+1:]
				break
			}
			if args[i] == "--repo" && i+1 < len(args) {
				repo = args[i+1]
				i++
			}
		}

		if !dashDash {
			// No --, treat all args as restic args
			resticArgs = args
		}

		if repo != "" {
			return runner.RunRestic(cfg.Global, repo, resticArgs)
		}
		return runner.RunResticRaw(cfg.Global, resticArgs)
	},
}

func init() {
	rootCmd.AddCommand(resticCmd)
}
