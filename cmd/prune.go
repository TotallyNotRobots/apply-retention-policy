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

// Package cmd implements the command-line interface for the retention policy
// tool. It provides commands for managing and applying retention policies to
// backup files.
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/TotallyNotRobots/apply-retention-policy/internal/config"
	"github.com/TotallyNotRobots/apply-retention-policy/internal/file"
	"github.com/TotallyNotRobots/apply-retention-policy/internal/retention"
	"github.com/TotallyNotRobots/apply-retention-policy/pkg/logger"
	"github.com/TotallyNotRobots/apply-retention-policy/pkg/must"
)

// pruneCmd represents the prune command
var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Apply retention policy to backup files",
	Long: `Apply retention policy to backup files based on the configured policy.
The policy specifies how many hourly, daily, weekly, monthly, and yearly backups to retain.
Files that don't meet the retention policy will be deleted.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Create context
		ctx := cmd.Context()

		if ctx == nil {
			ctx = context.Background()
		}

		// Load configuration
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize logger
		log, err := logger.New(cfg.LogLevel)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer log.MustSync()

		// Initialize file manager
		fileManager, err := file.NewManager(
			cfg.Directory,
			cfg.FilePattern,
			file.WithLogger(log),
		)
		if err != nil {
			return fmt.Errorf("failed to initialize file manager: %w", err)
		}

		// List files
		files, err := fileManager.ListFiles(ctx)
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}

		log.Info("config", zap.Any("config", cfg))

		if len(files) == 0 {
			log.Info("no backup files found")
			return nil
		}

		// Initialize retention policy
		policy := retention.NewPolicy(log, cfg)

		// Apply retention policy
		toDelete, err := policy.Apply(files)
		if err != nil {
			return fmt.Errorf("failed to apply retention policy: %w", err)
		}

		// Delete files
		for _, file := range toDelete {
			if err := fileManager.DeleteFile(ctx, file, cfg.DryRun); err != nil {
				log.Error("failed to delete file",
					zap.String("file", file.Path),
					zap.Error(err))
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)

	// Add flags
	pruneCmd.Flags().
		BoolP("dry-run", "d", false, "Show what would be deleted without actually deleting")
	pruneCmd.Flags().
		StringP("log-level", "l", "info", "Log level (debug, info, warn, error)")
	pruneCmd.Flags().
		StringVarP(&cfgFile, "config", "c", "", "Path to config file")

	// Bind flags to config
	must.Must(viper.BindPFlag("dry_run", pruneCmd.Flags().Lookup("dry-run")))
	must.Must(
		viper.BindPFlag("log_level", pruneCmd.Flags().Lookup("log-level")),
	)
}
