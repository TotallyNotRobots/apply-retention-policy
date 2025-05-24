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

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test config file
	configContent := `
retention:
  hourly: 2
  daily: 3
  weekly: 2
  monthly: 2
  yearly: 1
file_pattern: "backup-{year}-{month}-{day}-{hour}-{minute}.tar.gz"
directory: "/backups"
dry_run: true
log_level: "debug"
`
	configFile := filepath.Join(tmpDir, "retention-policy.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0o600)
	require.NoError(t, err)

	var cfg *Config

	t.Run("load from file", func(t *testing.T) {
		cfg, err = LoadConfig(configFile)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Verify config values
		require.Equal(t, 2, cfg.Retention.Hourly)
		require.Equal(t, 3, cfg.Retention.Daily)
		require.Equal(t, 2, cfg.Retention.Weekly)
		require.Equal(t, 2, cfg.Retention.Monthly)
		require.Equal(t, 1, cfg.Retention.Yearly)
		require.Equal(
			t,
			"backup-{year}-{month}-{day}-{hour}-{minute}.tar.gz",
			cfg.FilePattern,
		)
		require.Equal(t, "/backups", cfg.Directory)
		require.True(t, cfg.DryRun)
		require.Equal(t, "debug", cfg.LogLevel)
	})

	t.Run("load from default locations", func(t *testing.T) {
		// Create config in current directory
		err = os.WriteFile(
			"retention-policy.yaml",
			[]byte(configContent),
			0o600,
		)
		require.NoError(t, err)

		defer func() {
			if err = os.Remove("retention-policy.yaml"); err != nil {
				t.Errorf("Failed to remove test file: %v", err)
			}
		}()

		cfg, err = LoadConfig("")
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(
			t,
			"backup-{year}-{month}-{day}-{hour}-{minute}.tar.gz",
			cfg.FilePattern,
		)
	})

	t.Run("invalid config file", func(t *testing.T) {
		_, err = LoadConfig("non-existent.yaml")
		require.Error(t, err)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		invalidConfig := filepath.Join(tmpDir, "invalid.yaml")
		err = os.WriteFile(
			invalidConfig,
			[]byte("invalid: yaml: content"),
			0o600,
		)
		require.NoError(t, err)

		_, err := LoadConfig(invalidConfig)
		require.Error(t, err)
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			Retention: RetentionPolicy{
				Hourly:  2,
				Daily:   3,
				Weekly:  2,
				Monthly: 2,
				Yearly:  1,
			},
			FilePattern: "backup-{year}-{month}-{day}.tar.gz",
			Directory:   "/backups",
		}

		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("negative retention values", func(t *testing.T) {
		testCases := []struct {
			name  string
			cfg   *Config
			field string
		}{
			{
				name: "negative hourly",
				cfg: &Config{
					Retention:   RetentionPolicy{Hourly: -1},
					FilePattern: "backup.tar.gz",
					Directory:   "/backups",
				},
				field: "hourly",
			},
			{
				name: "negative daily",
				cfg: &Config{
					Retention:   RetentionPolicy{Daily: -1},
					FilePattern: "backup.tar.gz",
					Directory:   "/backups",
				},
				field: "daily",
			},
			{
				name: "negative weekly",
				cfg: &Config{
					Retention:   RetentionPolicy{Weekly: -1},
					FilePattern: "backup.tar.gz",
					Directory:   "/backups",
				},
				field: "weekly",
			},
			{
				name: "negative monthly",
				cfg: &Config{
					Retention:   RetentionPolicy{Monthly: -1},
					FilePattern: "backup.tar.gz",
					Directory:   "/backups",
				},
				field: "monthly",
			},
			{
				name: "negative yearly",
				cfg: &Config{
					Retention:   RetentionPolicy{Yearly: -1},
					FilePattern: "backup.tar.gz",
					Directory:   "/backups",
				},
				field: "yearly",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.cfg.Validate()
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.field)
			})
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		testCases := []struct {
			name string
			cfg  *Config
			msg  string
		}{
			{
				name: "missing file pattern",
				cfg: &Config{
					Retention: RetentionPolicy{Hourly: 1},
					Directory: "/backups",
				},
				msg: "file pattern must be specified",
			},
			{
				name: "missing directory",
				cfg: &Config{
					Retention:   RetentionPolicy{Hourly: 1},
					FilePattern: "backup.tar.gz",
				},
				msg: "directory must be specified",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.cfg.Validate()
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.msg)
			})
		}
	})
}

func TestConfig_GetRetentionDuration(t *testing.T) {
	t.Run("all periods set", func(t *testing.T) {
		cfg := &Config{
			Retention: RetentionPolicy{
				Hourly:  2,
				Daily:   3,
				Weekly:  2,
				Monthly: 2,
				Yearly:  1,
			},
		}

		duration := cfg.GetRetentionDuration()
		// Yearly retention should be the longest
		require.Equal(t, 365*24*time.Hour, duration)
	})

	t.Run("only hourly set", func(t *testing.T) {
		cfg := &Config{
			Retention: RetentionPolicy{
				Hourly: 48, // 2 days
			},
		}

		duration := cfg.GetRetentionDuration()
		require.Equal(t, 48*time.Hour, duration)
	})

	t.Run("no retention set", func(t *testing.T) {
		cfg := &Config{
			Retention: RetentionPolicy{},
		}

		duration := cfg.GetRetentionDuration()
		require.Equal(t, time.Duration(0), duration)
	})

	t.Run("mixed periods", func(t *testing.T) {
		cfg := &Config{
			Retention: RetentionPolicy{
				Hourly:  24, // 1 day
				Daily:   7,  // 7 days
				Weekly:  2,  // 14 days
				Monthly: 1,  // ~30 days
				Yearly:  0,  // not set
			},
		}

		duration := cfg.GetRetentionDuration()
		// Monthly retention should be the longest
		require.Equal(t, 30*24*time.Hour, duration)
	})
}
