package file

import (
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
	log := &logger.Logger{Logger: zap.NewNop()}
	dir := "/tmp/testdir"

	manager, err := NewManager(log, dir, testBackupPattern)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, dir, manager.directory)
	assert.NotNil(t, manager.filePattern)
}

func TestListFiles(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	dir := t.TempDir()
	manager, err := NewManager(log, dir, testBackupPattern)
	require.NoError(t, err)

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

	list, err := manager.ListFiles()
	require.NoError(t, err)
	assert.Len(t, list, len(files))
	assert.Equal(t, "backup-202501030000.zip", filepath.Base(list[0].Path))
	assert.Equal(t, "backup-202501010000.zip", filepath.Base(list[len(list)-1].Path))
}

func TestDeleteFile(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	dir := t.TempDir()
	manager, err := NewManager(log, dir, testBackupPattern)
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

	// Test dry run
	err = manager.DeleteFile(info, true)
	require.NoError(t, err)
	_, err = os.Stat(path)
	require.NoError(t, err)

	// Test actual deletion
	err = manager.DeleteFile(info, false)
	require.NoError(t, err)
	_, err = os.Stat(path)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestParseTimestamp(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	dir := "/tmp/testdir"
	manager, err := NewManager(log, dir, testBackupPattern)
	require.NoError(t, err)

	// Valid case
	matches := []string{"backup-202501010000.zip", "2025", "01", "01", "00", "00"}
	fieldNames := []string{"", "year", "month", "day", "hour", "minute"}

	timestamp, err := manager.parseTimestamp(matches, fieldNames)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), timestamp)

	// Missing year
	matches = []string{"backup--01010000.zip", "", "01", "01", "00", "00"}
	_, err = manager.parseTimestamp(matches, fieldNames)
	require.Error(t, err)

	// Invalid month
	matches = []string{"backup-202513010000.zip", "2025", "13", "01", "00", "00"}
	_, err = manager.parseTimestamp(matches, fieldNames)
	require.Error(t, err)

	// Invalid day
	matches = []string{"backup-202501320000.zip", "2025", "01", "32", "00", "00"}
	_, err = manager.parseTimestamp(matches, fieldNames)
	require.Error(t, err)

	// Invalid hour
	matches = []string{"backup-202501012500.zip", "2025", "01", "01", "25", "00"}
	_, err = manager.parseTimestamp(matches, fieldNames)
	require.Error(t, err)

	// Invalid minute
	matches = []string{"backup-202501010060.zip", "2025", "01", "01", "00", "60"}
	_, err = manager.parseTimestamp(matches, fieldNames)
	require.Error(t, err)

	// Missing matches
	matches = []string{"backup-202501010000.zip"}
	_, err = manager.parseTimestamp(matches, fieldNames)
	require.Error(t, err)

	// Extra fields in matches
	matches = []string{"backup-202501010000.zip", "2025", "01", "01", "00", "00", "extra"}
	_, err = manager.parseTimestamp(matches, fieldNames)
	require.Error(t, err)
}
