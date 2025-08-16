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

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/TotallyNotRobots/apply-retention-policy/pkg/logger"
)

func TestPruneCommand(t *testing.T) {
	// Enable test mode for logger
	logger.SetTestMode(true)
	defer logger.SetTestMode(false)

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Create test files
	testFiles := []struct {
		name      string
		timestamp time.Time
		content   string
	}{
		{
			name:      "backup-2024-03-15-12-00.tar.gz",
			timestamp: time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			content:   "test1",
		},
		{
			name:      "backup-2024-03-15-11-00.tar.gz",
			timestamp: time.Date(2024, 3, 15, 11, 0, 0, 0, time.UTC),
			content:   "test2",
		},
		{
			name:      "backup-2024-03-15-10-00.tar.gz",
			timestamp: time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
			content:   "test3",
		},
		{
			name:      "backup-2024-03-14-00-00.tar.gz",
			timestamp: time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC),
			content:   "test4",
		},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tmpDir, tf.name)
		err := os.WriteFile(path, []byte(tf.content), 0o600)
		require.NoError(t, err)
	}

	// Create a test config file
	configContent := `retention:
  hourly: 2
  daily: 1
  weekly: 0
  monthly: 0
  yearly: 0
file_pattern: "backup-{year}-{month}-{day}-{hour}-{minute}.tar.gz"
directory: "` + filepath.ToSlash(tmpDir) + `"
dry_run: false
log_level: "debug"
`
	configFile := filepath.Join(tmpDir, "retention-policy.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Set up viper for all tests
	viper.Reset()
	viper.SetConfigFile(configFile)
	err = viper.ReadInConfig()
	require.NoError(t, err)

	t.Run("dry run", func(t *testing.T) {
		viper.Set("dry_run", true)

		cmd := pruneCmd
		err := cmd.Flags().Set("config", configFile)
		require.NoError(t, err)
		err = cmd.RunE(cmd, nil)
		require.NoError(t, err)

		for _, tf := range testFiles {
			path := filepath.Join(tmpDir, tf.name)
			_, err := os.Stat(path)
			require.NoError(
				t,
				err,
				"File should not be deleted in dry run mode: %s",
				tf.name,
			)
		}
	})

	t.Run("actual run", func(t *testing.T) {
		viper.Set("dry_run", false)

		cmd := pruneCmd
		err := cmd.Flags().Set("config", configFile)
		require.NoError(t, err)
		err = cmd.RunE(cmd, nil)
		require.NoError(t, err)

		// Verify files were deleted according to retention policy
		expectedDeleted := []string{
			"backup-2024-03-14-00-00.tar.gz", // Daily backup (not needed)
		}

		for _, name := range expectedDeleted {
			path := filepath.Join(tmpDir, name)
			_, err := os.Stat(path)
			require.Error(t, err)
			require.ErrorIs(t, err, os.ErrNotExist)
		}

		// Verify retained files
		expectedRetained := []string{
			"backup-2024-03-15-12-00.tar.gz", // Newest hourly backup
			"backup-2024-03-15-11-00.tar.gz", // Second newest hourly backup
			"backup-2024-03-15-10-00.tar.gz", // Oldest hourly backup
		}

		for _, name := range expectedRetained {
			path := filepath.Join(tmpDir, name)
			_, err := os.Stat(path)
			require.NoError(t, err, "File should be retained: %s", name)
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		viper.Reset()
		viper.SetConfigFile("non-existent.yaml")

		cmd := pruneCmd
		err := cmd.Flags().Set("config", "non-existent.yaml")
		require.NoError(t, err)
		err = cmd.RunE(cmd, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to load config")
	})

	t.Run("context cancellation", func(t *testing.T) {
		viper.Reset()
		viper.SetConfigFile(configFile)
		err := viper.ReadInConfig()
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cmd := pruneCmd
		cmd.SetContext(ctx)
		err = cmd.Flags().Set("config", configFile)
		require.NoError(t, err)
		err = cmd.RunE(cmd, nil)
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestPruneCommandFlags(t *testing.T) {
	// Enable test mode for logger
	logger.SetTestMode(true)
	defer logger.SetTestMode(false)

	viper.Reset()
	t.Run("dry run flag", func(t *testing.T) {
		cmd := pruneCmd
		err := cmd.Flags().Set("dry-run", "true")
		require.NoError(t, err)
		err = viper.BindPFlag("dry_run", cmd.Flags().Lookup("dry-run"))
		require.NoError(t, err)
		require.True(t, viper.GetBool("dry_run"))
	})
	t.Run("log level flag", func(t *testing.T) {
		cmd := pruneCmd
		err := cmd.Flags().Set("log-level", "debug")
		require.NoError(t, err)
		err = viper.BindPFlag("log_level", cmd.Flags().Lookup("log-level"))
		require.NoError(t, err)
		require.Equal(t, "debug", viper.GetString("log_level"))
	})
}
