package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Global   GlobalConfig             `mapstructure:"global"`
	Profiles map[string]ProfileConfig `mapstructure:"profiles"`
}

type GlobalConfig struct {
	RcloneConfig             string `mapstructure:"rclone_config"`
	RcloneNoCheckCertificate bool   `mapstructure:"rclone_no_check_certificate"`
	ResticPassword           string `mapstructure:"restic_password"`
	ResticPackSize           int    `mapstructure:"restic_pack_size"`
	ResticExcludeFile        string `mapstructure:"restic_exclude_file"`
}

type ProfileConfig struct {
	Enabled *bool    `mapstructure:"enabled"`
	Repos   []string `mapstructure:"repos"`
	Paths   []string `mapstructure:"paths"`
}

// IsEnabled returns true if the profile is enabled (default when unset).
func (p *ProfileConfig) IsEnabled() bool {
	return p.Enabled == nil || *p.Enabled
}

// GetRepos resolves a profile name to a list of repos.
func (c *Config) GetRepos(arg string) ([]string, error) {
	if arg == "" {
		return nil, fmt.Errorf("profile name is required")
	}

	profile, ok := c.Profiles[arg]
	if !ok {
		names := make([]string, 0, len(c.Profiles))
		for k := range c.Profiles {
			names = append(names, k)
		}
		return nil, fmt.Errorf("profile %q not found; available: %s", arg, strings.Join(names, ", "))
	}

	if len(profile.Repos) == 0 {
		return nil, fmt.Errorf("profile %q has no repos configured", arg)
	}

	return profile.Repos, nil
}

// GetProfile returns the profile config for a given name.
func (c *Config) GetProfile(name string) (*ProfileConfig, error) {
	profile, ok := c.Profiles[name]
	if !ok {
		names := make([]string, 0, len(c.Profiles))
		for k := range c.Profiles {
			names = append(names, k)
		}
		return nil, fmt.Errorf("profile %q not found; available: %s", name, strings.Join(names, ", "))
	}
	return &profile, nil
}

func (c *Config) Validate() error {
	if c.Global.RcloneConfig == "" {
		return fmt.Errorf("global.rclone_config is required")
	}
	if c.Global.ResticPassword == "" {
		return fmt.Errorf("global.restic_password is required")
	}
	return nil
}
