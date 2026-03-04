package cmd

import (
	"github.com/kpkym/kup/internal/runner"
	"github.com/spf13/cobra"
)

var rcloneCmd = &cobra.Command{
	Use:                "rclone -- [rclone args...]",
	Short:              "Run rclone directly with kup environment",
	DisableFlagParsing: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return passthroughCompletion("rclone", append(args, toComplete))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle --help/-h explicitly since DisableFlagParsing prevents cobra from intercepting it
		for _, arg := range args {
			if arg == "--help" || arg == "-h" {
				return cmd.Help()
			}
		}

		// Strip leading "--" if present
		if len(args) > 0 && args[0] == "--" {
			args = args[1:]
		}

		return runner.RunRclone(cfg.Global, args)
	},
}

func init() {
	rootCmd.AddCommand(rcloneCmd)
}
