package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Worktree    WorktreeConfig `mapstructure:"worktree"`
	Branch      BranchConfig   `mapstructure:"branch"`
	Open        OpenConfig     `mapstructure:"open"`
	Hooks       HooksConfig    `mapstructure:"hooks"`
	Cache       CacheConfig    `mapstructure:"cache"`
	TUI         TUIConfig      `mapstructure:"tui"`
}

type TUIConfig struct {
	Keybindings []Keybinding `mapstructure:"keybindings"`
}

type Keybinding struct {
	Key         string   `mapstructure:"key"`
	Label       string   `mapstructure:"label"`
	Command     string   `mapstructure:"command"`
	Input       string   `mapstructure:"input"`
	Confirm     bool     `mapstructure:"confirm"`
	Requires    []string `mapstructure:"requires"`
	Interactive bool     `mapstructure:"interactive"`
	Output      bool     `mapstructure:"output"`
}

type WorktreeConfig struct {
	Path        string            `mapstructure:"path"`
	CopyIgnored CopyIgnoredConfig `mapstructure:"copy-ignored"`
}

type CopyIgnoredConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	Exclude []string `mapstructure:"exclude"`
	Symlink []string `mapstructure:"symlink"`
}

type BranchConfig struct {
	Base string `mapstructure:"base"`
}

type OpenConfig struct {
	Command string `mapstructure:"command"`
	OnStart bool   `mapstructure:"on_start"`
}

type HooksConfig struct {
	PreStart  []HookStep `mapstructure:"pre-start"`
	PostStart []HookStep `mapstructure:"post-start"`
}

type HookStep struct {
	Run string `mapstructure:"run"`
}

type CacheConfig struct {
	TTL int `mapstructure:"ttl"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	// Defaults
	v.SetDefault("worktree.path", "../{{.RepoName}}-wt/{{.Branch}}")
	v.SetDefault("worktree.copy-ignored.enabled", false)
	v.SetDefault("branch.base", "")
	v.SetDefault("open.command", "")
	v.SetDefault("open.on_start", false)
	v.SetDefault("cache.ttl", 60)

	// User-level config (base)
	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		userConfigPath := filepath.Join(userConfigDir, "gh-wt", "config.yml")
		if _, err := os.Stat(userConfigPath); err == nil {
			v.SetConfigFile(userConfigPath)
			v.ReadInConfig()
		}
	}

	// Repo-level config (overrides user)
	repoViper := viper.New()
	repoViper.SetConfigType("yaml")
	repoViper.SetConfigName(".gh-wt")
	repoViper.AddConfigPath(".")
	if err := repoViper.ReadInConfig(); err == nil {
		v.MergeConfigMap(repoViper.AllSettings())
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
