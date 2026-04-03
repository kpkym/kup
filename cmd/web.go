package cmd

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/kpkym/kup/internal/web"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web --repo <repo>",
	Short: "Serve a web UI for browsing snapshots and streaming files",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		listen, _ := cmd.Flags().GetString("listen")

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		srv := web.NewServer(cfg.Global, repo)
		return srv.ListenAndServe(ctx, listen)
	},
}

func init() {
	webCmd.Flags().String("repo", "", "repo URI (required)")
	webCmd.MarkFlagRequired("repo")
	webCmd.RegisterFlagCompletionFunc("repo", repoCompletionFunc)
	webCmd.Flags().String("listen", ":8080", "HTTP listen address")
	rootCmd.AddCommand(webCmd)
}
