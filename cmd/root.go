package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/kpkym/kup/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "kup",
	Short: "Encrypted backup tool combining restic and rclone",
	Long:  "kup orchestrates restic (encryption/deduplication) and rclone (cloud transport) for profile-based backups.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for help commands
		if cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "version" {
			return nil
		}
		// For passthrough commands (DisableFlagParsing), skip config if --help/-h in raw args
		if cmd.DisableFlagParsing {
			for _, arg := range args {
				if arg == "--help" || arg == "-h" {
					return nil
				}
			}
		}
		cfgFile, _ := cmd.Root().PersistentFlags().GetString("config")
		return loadConfig(cfgFile)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "config file (default: $KUP_CONFIG_DIR/config.toml)")
	rootCmd.PersistentFlags().Bool("dry-run", false, "pass --dry-run to restic")
}

func repoCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(cfg.Profiles) == 0 {
		cfgFile, _ := cmd.Root().PersistentFlags().GetString("config")
		if err := loadConfig(cfgFile); err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
	}
	var allRepos []string
	for _, profile := range cfg.Profiles {
		allRepos = append(allRepos, profile.Repos...)
	}
	sort.Strings(allRepos)
	return allRepos, cobra.ShellCompDirectiveNoFileComp
}

func profileCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if len(cfg.Profiles) == 0 {
		cfgFile, _ := cmd.Root().PersistentFlags().GetString("config")
		if err := loadConfig(cfgFile); err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
	}
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func loadConfig(cfgFile string) error {
	var configDir string

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		configDir = filepath.Dir(cfgFile)
	} else {
		envDir := os.Getenv("KUP_CONFIG_DIR")
		if envDir == "" {
			return fmt.Errorf("KUP_CONFIG_DIR environment variable is not set")
		}
		configDir = envDir
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(configDir)
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	// Merge per-profile files from profiles/ subdirectory.
	profilesDir := filepath.Join(configDir, "profiles")
	if entries, err := os.ReadDir(profilesDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".toml" {
				continue
			}
			name := e.Name()[:len(e.Name())-len(".toml")]
			v2 := viper.New()
			v2.SetConfigFile(filepath.Join(profilesDir, e.Name()))
			if err := v2.ReadInConfig(); err != nil {
				return fmt.Errorf("reading profiles/%s: %w", e.Name(), err)
			}
			if err := viper.MergeConfigMap(map[string]any{
				"profiles": map[string]any{name: v2.AllSettings()},
			}); err != nil {
				return fmt.Errorf("merging profiles/%s: %w", e.Name(), err)
			}
		}
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find home directory: %w", err)
	}
	expandHome := func(p string) string {
		if len(p) >= 2 && p[:2] == "~/" {
			return home + p[1:]
		}
		return p
	}
	cfg.Global.RcloneConfig = expandHome(cfg.Global.RcloneConfig)
	cfg.Global.ResticExcludeFile = expandHome(cfg.Global.ResticExcludeFile)

	return cfg.Validate()
}
