/*
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
package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/TotallyNotRobots/apply-retention-policy/pkg/logger"
)

const (
	testBackupPattern = "backup-{year}{month}{day}{hour}{minute}.zip"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	log := &logger.Logger{Logger: zap.NewNop()}
	dir := t.TempDir()

	// Test with options
	manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, dir, manager.directory)
	assert.NotNil(t, manager.filePattern)

	// Test with invalid pattern
	_, err = NewManager(dir, "(?invalid", WithLogger(log))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidPattern)
}

func TestListFiles(t *testing.T) {
	t.Parallel()
	// Setup
	ctx := context.Background()
	log := &logger.Logger{Logger: zap.NewNop()}
	dir := t.TempDir()
	manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
	require.NoError(t, err)

	// Test with empty directory
	emptyList, err := manager.ListFiles(ctx)
	require.NoError(t, err)
	assert.Empty(t, emptyList)

	// Create test files
	files := []string{
		"backup-202501010000.zip",
		"backup-202501020000.zip",
		"backup-202501030000.zip",
	}
	for _, file := range files {
		path := filepath.Join(dir, file)
		_, err := os.Create(path)
		require.NoError(t, err)
	}

	// Also create a non-matching file
	nonMatchingFile := filepath.Join(dir, "not-a-backup.txt")
	_, err = os.Create(nonMatchingFile)
	require.NoError(t, err)

	// Execute
	list, err := manager.ListFiles(ctx)

	// Verify
	require.NoError(t, err)
	assert.Len(t, list, len(files))
	assert.Equal(t, "backup-202501030000.zip", filepath.Base(list[0].Path))
	assert.Equal(t, "backup-202501010000.zip", filepath.Base(list[len(list)-1].Path))

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	_, err = manager.ListFiles(cancelledCtx)
	require.Error(t, err)
}

func TestDeleteFile(t *testing.T) {
	t.Parallel()
	// Setup
	ctx := context.Background()
	log := &logger.Logger{Logger: zap.NewNop()}
	dir := t.TempDir()
	manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
	require.NoError(t, err)

	// Create a test file
	file := "backup-202501010000.zip"
	path := filepath.Join(dir, file)
	_, err = os.Create(path)
	require.NoError(t, err)

	info := Info{
		Path:      path,
		Timestamp: time.Now(),
		Size:      1234,
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	err = manager.DeleteFile(cancelledCtx, info, false)
	require.Error(t, err)

	// Test dry run
	err = manager.DeleteFile(ctx, info, true)
	require.NoError(t, err)
	_, err = os.Stat(path)
	require.NoError(t, err)

	// Test actual deletion
	err = manager.DeleteFile(ctx, info, false)
	require.NoError(t, err)
	_, err = os.Stat(path)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	// Test deleting non-existent file
	nonExistentInfo := Info{
		Path:      filepath.Join(dir, "non-existent.zip"),
		Timestamp: time.Now(),
		Size:      0,
	}
	err = manager.DeleteFile(ctx, nonExistentInfo, false)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDeleteFile)
}

func TestParseTimestamp(t *testing.T) {
	t.Parallel()
	// Setup
	log := &logger.Logger{Logger: zap.NewNop()}
	dir := t.TempDir()
	manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
	require.NoError(t, err)

	// Test cases table
	testCases := []struct {
		name       string
		matches    []string
		fieldNames []string
		expected   time.Time
		expectErr  bool
	}{
		{
			name:       "valid case",
			matches:    []string{"backup-202501010000.zip", "2025", "01", "01", "00", "00"},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expected:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expectErr:  false,
		},
		{
			name:       "missing year",
			matches:    []string{"backup--01010000.zip", "", "01", "01", "00", "00"},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
		{
			name:       "invalid month",
			matches:    []string{"backup-202513010000.zip", "2025", "13", "01", "00", "00"},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
		{
			name:       "invalid day",
			matches:    []string{"backup-202501320000.zip", "2025", "01", "32", "00", "00"},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
		{
			name:       "invalid hour",
			matches:    []string{"backup-202501012500.zip", "2025", "01", "01", "25", "00"},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
		{
			name:       "invalid minute",
			matches:    []string{"backup-202501010060.zip", "2025", "01", "01", "00", "60"},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
		{
			name:       "missing matches",
			matches:    []string{"backup-202501010000.zip"},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
		{
			name: "extra fields in matches",
			matches: []string{
				"backup-202501010000.zip",
				"2025", "01", "01", "00", "00", "extra",
			},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		testCase := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			timestamp, err := manager.parseTimestamp(testCase.matches, testCase.fieldNames)
			if testCase.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.expected, timestamp)
			}
		})
	}
}
