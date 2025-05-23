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

package retention

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/TotallyNotRobots/apply-retention-policy/internal/config"
	"github.com/TotallyNotRobots/apply-retention-policy/internal/file"
	"github.com/TotallyNotRobots/apply-retention-policy/pkg/logger"
)

func TestPolicy_Apply(t *testing.T) {
	// Enable test mode for logger
	logger.SetTestMode(true)
	defer logger.SetTestMode(false)

	// Create a test logger that writes to a buffer
	log := &logger.Logger{Logger: zap.NewNop()}

	// Create a test config
	cfg := &config.Config{
		Retention: config.RetentionPolicy{
			Hourly:  2,
			Daily:   3,
			Weekly:  2,
			Monthly: 2,
			Yearly:  1,
		},
	}

	// Create test policy
	policy := NewPolicy(log, cfg)

	t.Run("empty file list", func(t *testing.T) {
		toDelete, err := policy.Apply([]file.Info{})
		require.NoError(t, err)
		assert.Empty(t, toDelete)
	})

	t.Run("basic retention policy", func(t *testing.T) {
		now := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
		files := []file.Info{
			// Hourly backups (keep 2)
			{
				Path:      "backup-2024-03-15-11-00.tar.gz",
				Timestamp: now.Add(-1 * time.Hour),
			},
			{
				Path:      "backup-2024-03-15-10-00.tar.gz",
				Timestamp: now.Add(-2 * time.Hour),
			},
			{
				Path:      "backup-2024-03-15-09-00.tar.gz",
				Timestamp: now.Add(-3 * time.Hour),
			},
			{
				Path:      "backup-2024-03-15-08-00.tar.gz",
				Timestamp: now.Add(-4 * time.Hour),
			},

			// Daily backups (keep 3)
			{
				Path:      "backup-2024-03-14-00-00.tar.gz",
				Timestamp: now.Add(-36 * time.Hour),
			},
			{
				Path:      "backup-2024-03-13-00-00.tar.gz",
				Timestamp: now.Add(-60 * time.Hour),
			},
			{
				Path:      "backup-2024-03-12-00-00.tar.gz",
				Timestamp: now.Add(-84 * time.Hour),
			},
			{
				Path:      "backup-2024-03-11-00-00.tar.gz",
				Timestamp: now.Add(-108 * time.Hour),
			},

			// Weekly backups (keep 2)
			{
				Path:      "backup-2024-03-08-00-00.tar.gz",
				Timestamp: now.Add(-7 * 24 * time.Hour),
			},
			{
				Path:      "backup-2024-03-01-00-00.tar.gz",
				Timestamp: now.Add(-14 * 24 * time.Hour),
			},
			{
				Path:      "backup-2024-02-23-00-00.tar.gz",
				Timestamp: now.Add(-21 * 24 * time.Hour),
			},

			// Monthly backups (keep 2)
			{
				Path:      "backup-2024-02-15-00-00.tar.gz",
				Timestamp: now.Add(-29 * 24 * time.Hour),
			},
			{
				Path:      "backup-2024-01-15-00-00.tar.gz",
				Timestamp: now.Add(-59 * 24 * time.Hour),
			},
			{
				Path:      "backup-2023-12-15-00-00.tar.gz",
				Timestamp: now.Add(-90 * 24 * time.Hour),
			},

			// Yearly backup (keep 1)
			{
				Path:      "backup-2023-03-15-00-00.tar.gz",
				Timestamp: now.Add(-366 * 24 * time.Hour),
			},
			{
				Path:      "backup-2022-03-15-00-00.tar.gz",
				Timestamp: now.Add(-731 * 24 * time.Hour),
			},
		}

		toDelete, err := policy.Apply(files)
		require.NoError(t, err)

		// Verify the correct files are marked for deletion
		expectedToDelete := []string{
			"backup-2024-03-15-08-00.tar.gz",
			"backup-2024-03-11-00-00.tar.gz",
			"backup-2024-02-15-00-00.tar.gz",
			"backup-2023-12-15-00-00.tar.gz",
			"backup-2023-03-15-00-00.tar.gz",
			"backup-2022-03-15-00-00.tar.gz",
		}

		assert.Len(t, toDelete, len(expectedToDelete))

		for _, expected := range expectedToDelete {
			found := false

			for _, f := range toDelete {
				if f.Path == expected {
					found = true
					break
				}
			}

			assert.True(
				t,
				found,
				"Expected file %s to be marked for deletion",
				expected,
			)
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		now := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)

		t.Run("files exactly on period boundaries", func(t *testing.T) {
			files := []file.Info{
				{Path: "backup-2024-03-15-12-00.tar.gz", Timestamp: now},
				{
					Path:      "backup-2024-03-15-11-00.tar.gz",
					Timestamp: now.Add(-1 * time.Hour),
				},
				{
					Path:      "backup-2024-03-15-10-00.tar.gz",
					Timestamp: now.Add(-2 * time.Hour),
				},
			}

			toDelete, err := policy.Apply(files)
			require.NoError(t, err)
			assert.Empty(t, toDelete)
		})

		t.Run("files spanning multiple periods", func(t *testing.T) {
			files := []file.Info{
				{Path: "backup-2024-03-15-12-00.tar.gz", Timestamp: now},
				{
					Path:      "backup-2024-03-14-12-00.tar.gz",
					Timestamp: now.Add(-24 * time.Hour),
				},
				{
					Path:      "backup-2024-03-13-12-00.tar.gz",
					Timestamp: now.Add(-48 * time.Hour),
				},
				{
					Path:      "backup-2024-03-12-12-00.tar.gz",
					Timestamp: now.Add(-72 * time.Hour),
				},
			}

			toDelete, err := policy.Apply(files)
			require.NoError(t, err)
			assert.Empty(t, toDelete)
		})

		t.Run("files out of order", func(t *testing.T) {
			files := []file.Info{
				{
					Path:      "backup-2024-03-15-10-00.tar.gz",
					Timestamp: now.Add(-2 * time.Hour),
				},
				{Path: "backup-2024-03-15-12-00.tar.gz", Timestamp: now},
				{
					Path:      "backup-2024-03-15-11-00.tar.gz",
					Timestamp: now.Add(-1 * time.Hour),
				},
			}

			toDelete, err := policy.Apply(files)
			require.NoError(t, err)
			assert.Empty(t, toDelete)
		})
	})
}

func TestPolicy_groupFilesByPeriod(t *testing.T) {
	// Enable test mode for logger
	logger.SetTestMode(true)
	defer logger.SetTestMode(false)

	t.Run("basic grouping", func(t *testing.T) {
		now := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
		files := []file.Info{
			{Path: "file1", Timestamp: now},
			{Path: "file2", Timestamp: now.Add(-30 * time.Minute)},
			{Path: "file3", Timestamp: now.Add(-90 * time.Minute)},
			{Path: "file4", Timestamp: now.Add(-120 * time.Minute)},
		}

		selected, toDelete, unselected := groupFilesByPeriod(
			files,
			hourGrouper,
			2,
		)

		assert.Len(t, selected, 2)
		assert.Empty(t, toDelete)
		assert.Len(t, unselected, 2)

		assert.Equal(t, "file1", selected[0].Path)
		assert.Equal(t, "file2", selected[1].Path)
		assert.Equal(t, "file3", unselected[0].Path)
		assert.Equal(t, "file4", unselected[1].Path)
	})

	t.Run("empty input", func(t *testing.T) {
		selected, toDelete, unselected := groupFilesByPeriod(
			[]file.Info{},
			hourGrouper,
			2,
		)
		assert.Empty(t, selected)
		assert.Empty(t, toDelete)
		assert.Empty(t, unselected)
	})

	t.Run("keep all files", func(t *testing.T) {
		now := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
		files := []file.Info{
			{Path: "file1", Timestamp: now},
			{Path: "file2", Timestamp: now.Add(-60 * time.Minute)},
		}

		selected, toDelete, unselected := groupFilesByPeriod(
			files,
			hourGrouper,
			2,
		)

		assert.Len(t, selected, 2)
		assert.Empty(t, toDelete)
		assert.Empty(t, unselected)
	})
}

func TestGroupFilesByTimePeriod(t *testing.T) {
	now := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		files    []file.Info
		grouper  func(file.Info) int64
		expected [][]file.Info
	}{
		{
			name:     "empty files list",
			files:    []file.Info{},
			grouper:  hourGrouper,
			expected: nil,
		},
		{
			name: "single file",
			files: []file.Info{
				{Timestamp: now.Add(-30 * time.Minute)},
			},
			grouper: hourGrouper,
			expected: [][]file.Info{
				{{Timestamp: now.Add(-30 * time.Minute)}},
			},
		},
		{
			name: "files in same period",
			files: []file.Info{
				{Timestamp: now.Add(-30 * time.Minute)},
				{Timestamp: now.Add(-45 * time.Minute)},
				{Timestamp: now.Add(-15 * time.Minute)},
			},
			grouper: hourGrouper,
			expected: [][]file.Info{
				{
					{Timestamp: now.Add(-15 * time.Minute)},
					{Timestamp: now.Add(-30 * time.Minute)},
					{Timestamp: now.Add(-45 * time.Minute)},
				},
			},
		},
		{
			name: "files in different periods",
			files: []file.Info{
				{Timestamp: now.Add(-30 * time.Minute)},
				{Timestamp: now.Add(-90 * time.Minute)},
				{Timestamp: now.Add(-150 * time.Minute)},
			},
			grouper: hourGrouper,
			expected: [][]file.Info{
				{{Timestamp: now.Add(-30 * time.Minute)}},
				{{Timestamp: now.Add(-90 * time.Minute)}},
				{{Timestamp: now.Add(-150 * time.Minute)}},
			},
		},
		{
			name: "files in different periods with multiple files per period",
			files: []file.Info{
				{Timestamp: now.Add(-30 * time.Minute)},
				{Timestamp: now.Add(-45 * time.Minute)},
				{Timestamp: now.Add(-90 * time.Minute)},
				{Timestamp: now.Add(-95 * time.Minute)},
				{Timestamp: now.Add(-150 * time.Minute)},
				{Timestamp: now.Add(-155 * time.Minute)},
			},
			grouper: hourGrouper,
			expected: [][]file.Info{
				{
					{Timestamp: now.Add(-30 * time.Minute)},
					{Timestamp: now.Add(-45 * time.Minute)},
				},
				{
					{Timestamp: now.Add(-90 * time.Minute)},
					{Timestamp: now.Add(-95 * time.Minute)},
				},
				{
					{Timestamp: now.Add(-150 * time.Minute)},
					{Timestamp: now.Add(-155 * time.Minute)},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := groupFilesByTimePeriod(tt.files, tt.grouper)
			if tt.expected == nil {
				assert.Nil(t, groups)
			} else {
				assert.Equal(t, tt.expected, groups)
			}
		})
	}
}
