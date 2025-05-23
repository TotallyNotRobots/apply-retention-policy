/*
The MIT License (MIT)

Copyright © 2025 linuxdaemon <linuxdaemon.irc@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package config provides configuration management for the retention policy
// tool. It handles loading and parsing of configuration files and environment
// variables.
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"

	"github.com/TotallyNotRobots/apply-retention-policy/internal/consts"
)

// RetentionPolicy defines how many backups to keep for each time period
type RetentionPolicy struct {
	Hourly  int `mapstructure:"hourly"  yaml:"hourly"`
	Daily   int `mapstructure:"daily"   yaml:"daily"`
	Weekly  int `mapstructure:"weekly"  yaml:"weekly"`
	Monthly int `mapstructure:"monthly" yaml:"monthly"`
	Yearly  int `mapstructure:"yearly"  yaml:"yearly"`
}

// Config represents the application configuration
type Config struct {
	Retention   RetentionPolicy `mapstructure:"retention"    yaml:"retention"`
	FilePattern string          `mapstructure:"file_pattern" yaml:"file_pattern"`
	Directory   string          `mapstructure:"directory"    yaml:"directory"`
	DryRun      bool            `mapstructure:"dry_run"      yaml:"dry_run"`
	LogLevel    string          `mapstructure:"log_level"    yaml:"log_level"`
}

// LoadConfig loads the configuration from the specified file
func LoadConfig(configFile string) (*Config, error) {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("retention-policy")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.apply-retention-policy")
		viper.AddConfigPath("/etc/apply-retention-policy")
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Retention.Hourly < 0 {
		return fmt.Errorf("hourly retention must be non-negative")
	}

	if c.Retention.Daily < 0 {
		return fmt.Errorf("daily retention must be non-negative")
	}

	if c.Retention.Weekly < 0 {
		return fmt.Errorf("weekly retention must be non-negative")
	}

	if c.Retention.Monthly < 0 {
		return fmt.Errorf("monthly retention must be non-negative")
	}

	if c.Retention.Yearly < 0 {
		return fmt.Errorf("yearly retention must be non-negative")
	}

	if c.FilePattern == "" {
		return fmt.Errorf("file pattern must be specified")
	}

	if c.Directory == "" {
		return fmt.Errorf("directory must be specified")
	}

	return nil
}

// GetRetentionDuration returns the duration for which files should be retained
// based on the retention policy
func (c *Config) GetRetentionDuration() time.Duration {
	// Calculate the maximum retention period
	// This is used to determine how far back we need to look for files
	maxDuration := time.Duration(0)

	if c.Retention.Yearly > 0 {
		maxDuration = time.Duration(c.Retention.Yearly) * consts.YEAR
	}

	if c.Retention.Monthly > 0 {
		duration := time.Duration(c.Retention.Monthly) * consts.MONTH
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	if c.Retention.Weekly > 0 {
		duration := time.Duration(c.Retention.Weekly) * consts.WEEK
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	if c.Retention.Daily > 0 {
		duration := time.Duration(c.Retention.Daily) * consts.DAY
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	if c.Retention.Hourly > 0 {
		duration := time.Duration(c.Retention.Hourly) * consts.HOUR
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	return maxDuration
}
