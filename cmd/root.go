package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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
	rootCmd.PersistentFlags().String("config", "", "config file (default: $KPK_CONFIG_DIR/kup/config.toml or ~/.config/kup/config.toml)")
	rootCmd.PersistentFlags().Bool("dry-run", false, "pass --dry-run to restic")
}

func passthroughCompletion(binary string, args []string) ([]string, cobra.ShellCompDirective) {
	out, err := exec.Command(binary, append([]string{"__complete"}, args...)...).Output()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var completions []string
	directive := cobra.ShellCompDirectiveNoFileComp
	for line := range strings.SplitSeq(strings.TrimRight(string(out), "\n"), "\n") {
		if strings.HasPrefix(line, ":") {
			if d, err := strconv.Atoi(line[1:]); err == nil {
				directive = cobra.ShellCompDirective(d)
			}
			break
		}
		if line != "" {
			completions = append(completions, line)
		}
	}
	return completions, directive
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

func resolveRepos(args []string) ([]string, error) {
	var repos []string
	for _, arg := range args {
		r, err := cfg.GetRepos(arg)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r...)
	}
	return repos, nil
}

func loadConfig(cfgFile string) error {
	var configDir string

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		configDir = filepath.Dir(cfgFile)
	} else {
		envDir := os.Getenv("KPK_CONFIG_DIR")
		if envDir != "" {
			configDir = filepath.Join(envDir, "kup")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot find home directory: %w", err)
			}
			configDir = filepath.Join(home, ".config", "kup")
		}
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(configDir)
	}

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	// Merge per-profile files from profiles/ subdirectory (recursive).
	profilesDir := filepath.Join(configDir, "profiles")
	_ = filepath.WalkDir(profilesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(d.Name()) != ".toml" {
			return nil
		}
		rel, _ := filepath.Rel(profilesDir, path)
		name := rel[:len(rel)-len(".toml")]
		v2 := viper.New()
		v2.SetConfigFile(path)
		if err := v2.ReadInConfig(); err != nil {
			return fmt.Errorf("reading profiles/%s: %w", rel, err)
		}
		if err := viper.MergeConfigMap(map[string]any{
			"profiles": map[string]any{name: v2.AllSettings()},
		}); err != nil {
			return fmt.Errorf("merging profiles/%s: %w", rel, err)
		}
		return nil
	})

	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	for name, profile := range cfg.Profiles {
		if !profile.IsEnabled() {
			delete(cfg.Profiles, name)
		}
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
