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

package file

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"github.com/TotallyNotRobots/apply-retention-policy/pkg/logger"
)

func init() {
	// Enable test mode for logger in all tests
	logger.SetTestMode(true)
}

const (
	testBackupPattern = "backup-{year}{month}{day}{hour}{minute}.zip"
)

// checkACLSupport checks if the system supports ACLs
func checkACLSupport(t *testing.T) bool {
	// Try to get ACL support via statfs
	var stat unix.Statfs_t
	err := unix.Statfs(".", &stat)
	if err != nil {
		t.Skip("Cannot check ACL support:", err)
		return false
	}

	// Check if filesystem supports ACLs
	// This is a basic check - some filesystems might support ACLs
	// even if this check fails
	return stat.Type == unix.XFS_SUPER_MAGIC ||
		stat.Type == unix.EXT4_SUPER_MAGIC ||
		stat.Type == unix.BTRFS_SUPER_MAGIC
}

// checkSymlinkSupport checks if the system supports symlinks
func checkSymlinkSupport(t *testing.T) bool {
	if runtime.GOOS == "windows" {
		// On Windows, check if we have symlink privileges
		// This is a basic check - in practice, you might need to check
		// for specific Windows versions or user privileges
		return true // We'll let the symlink creation fail if not supported
	}
	return true
}

// checkFIFOSupport checks if the system supports named pipes (FIFOs)
func checkFIFOSupport(t *testing.T) bool {
	if runtime.GOOS == "windows" {
		// Windows supports named pipes but with different semantics
		// We'll skip the test on Windows for now
		t.Skip("Named pipes test not implemented for Windows")
		return false
	}

	// Try to create a temporary FIFO
	dir := t.TempDir()
	pipePath := filepath.Join(dir, "testpipe")
	err := syscall.Mkfifo(pipePath, 0644)
	if err != nil {
		t.Log("FIFO not supported:", err)
		return false
	}

	// Clean up the test FIFO
	_ = os.Remove(pipePath)
	return true
}

// setReadOnly makes a file read-only in a platform-independent way
func setReadOnly(t *testing.T, path string) {
	if runtime.GOOS == "windows" {
		// On Windows, use attrib command
		cmd := exec.Command("attrib", "+R", path)
		err := cmd.Run()
		require.NoError(t, err, "Failed to set read-only attribute on Windows")
	} else {
		// On Unix-like systems, use chmod
		err := os.Chmod(path, 0444)
		require.NoError(t, err, "Failed to set read-only mode")
	}
}

// removeReadOnly removes read-only attribute in a platform-independent way
func removeReadOnly(t *testing.T, path string) {
	if runtime.GOOS == "windows" {
		// On Windows, use attrib command
		cmd := exec.Command("attrib", "-R", path)
		err := cmd.Run()
		require.NoError(t, err, "Failed to remove read-only attribute on Windows")
	} else {
		// On Unix-like systems, use chmod
		err := os.Chmod(path, 0644)
		require.NoError(t, err, "Failed to remove read-only mode")
	}
}

func TestNewManager(t *testing.T) {
	t.Run("valid pattern", func(t *testing.T) {
		// Use a no-op logger for testing
		log := &logger.Logger{Logger: zap.NewNop()}

		m, err := NewManager(
			"/tmp",
			"backup-{year}-{month}-{day}-{hour}-{minute}.tar.gz",
			WithLogger(log),
		)
		require.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, "/tmp", m.directory)
		assert.NotNil(t, m.filePattern)
	})

	t.Run("invalid pattern", func(t *testing.T) {
		_, err := NewManager("/tmp", "backup-[invalid")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPattern)
	})

	t.Run("with logger option", func(t *testing.T) {
		// Use a no-op logger for testing
		log := &logger.Logger{Logger: zap.NewNop()}

		m, err := NewManager(
			"/tmp",
			"backup-{year}-{month}-{day}.tar.gz",
			WithLogger(log),
		)
		require.NoError(t, err)
		assert.NotNil(t, m.logger)
	})
}

func TestListFiles(t *testing.T) {
	t.Parallel()
	// Setup
	ctx := context.Background()
	log := &logger.Logger{Logger: zap.NewNop()}
	dir := t.TempDir()
	manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
	require.NoError(t, err)

	// Create test files
	files := []string{
		"backup-202501010000.zip",
		"backup-202501020000.zip",
		"backup-202501030000.zip",
	}
	for _, file := range files {
		path := filepath.Clean(filepath.Join(dir, file))
		_, err = os.Create(path)
		require.NoError(t, err)
	}

	// Create a directory
	dirPath := filepath.Join(dir, "subdir")
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	t.Run("with symlink", func(t *testing.T) {
		if !checkSymlinkSupport(t) {
			t.Skip("Symlinks not supported on this system")
		}

		// Create a symlink
		symlinkPath := filepath.Join(dir, "symlink")
		err = os.Symlink(filepath.Join(dir, files[0]), symlinkPath)
		if err != nil {
			if runtime.GOOS == "windows" {
				t.Skip("Symlink creation failed on Windows - may need elevated privileges")
			}
			require.NoError(t, err)
		}

		// List files and verify symlink is not included
		list, err := manager.ListFiles(ctx)
		require.NoError(t, err)
		assert.Len(t, list, len(files))
		for _, file := range list {
			base := filepath.Base(file.Path)
			assert.Contains(t, files, base)
			assert.NotEqual(t, "symlink", base, "Symlink should not be included in results")
		}
	})

	t.Run("with named pipe", func(t *testing.T) {
		if !checkFIFOSupport(t) {
			t.Skip("Named pipes not supported on this system")
		}

		// Create a named pipe
		pipePath := filepath.Join(dir, "pipe")
		err = syscall.Mkfifo(pipePath, 0644)
		require.NoError(t, err)

		// Verify the pipe exists and is a FIFO
		pipeInfo, err := os.Lstat(pipePath)
		require.NoError(t, err)
		require.True(t, pipeInfo.Mode()&os.ModeNamedPipe != 0, "Created path is not a named pipe")

		// List files and verify the pipe is not included
		list, err := manager.ListFiles(ctx)
		require.NoError(t, err)
		assert.Len(t, list, len(files))
		for _, file := range list {
			base := filepath.Base(file.Path)
			assert.Contains(t, files, base)
			assert.NotEqual(t, "pipe", base, "Named pipe should not be included in results")
		}

		// Clean up the pipe
		err = os.Remove(pipePath)
		require.NoError(t, err)
	})

	// Execute
	list, err := manager.ListFiles(ctx)

	// Verify
	require.NoError(t, err)
	assert.Len(t, list, len(files))
	for _, file := range list {
		base := filepath.Base(file.Path)
		assert.Contains(t, files, base)
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = manager.ListFiles(cancelledCtx)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
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
	path := filepath.Clean(filepath.Join(dir, file))
	_, err = os.Create(path)
	require.NoError(t, err)

	info := Info{
		Path:      path,
		Timestamp: time.Now(),
		Size:      1234,
	}

	t.Run("delete regular file", func(t *testing.T) {
		err = manager.DeleteFile(ctx, info, false)
		require.NoError(t, err)
		_, err = os.Stat(path)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete non-existent file", func(t *testing.T) {
		nonExistentInfo := Info{
			Path:      filepath.Join(dir, "non-existent.zip"),
			Timestamp: time.Now(),
			Size:      0,
		}
		err = manager.DeleteFile(ctx, nonExistentInfo, false)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDeleteFile)
	})

	t.Run("delete directory", func(t *testing.T) {
		// Create a directory
		dirPath := filepath.Join(dir, "testdir")
		err = os.Mkdir(dirPath, 0755)
		require.NoError(t, err)

		dirInfo := Info{
			Path:      dirPath,
			Timestamp: time.Now(),
			Size:      0,
		}
		err = manager.DeleteFile(ctx, dirInfo, false)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotRegularFile)
	})

	t.Run("delete symlink", func(t *testing.T) {
		if !checkSymlinkSupport(t) {
			t.Skip("Symlinks not supported on this system")
		}

		// Create a target file first
		targetPath := filepath.Join(dir, "target.zip")
		_, err = os.Create(targetPath)
		require.NoError(t, err)

		// Create a symlink to the target file
		symlinkPath := filepath.Join(dir, "symlink")
		err = os.Symlink(targetPath, symlinkPath)
		if err != nil {
			if runtime.GOOS == "windows" {
				t.Skip("Symlink creation failed on Windows - may need elevated privileges")
			}
			require.NoError(t, err)
		}

		// Verify the symlink exists and is a symlink
		linkInfo, err := os.Lstat(symlinkPath)
		require.NoError(t, err)
		if runtime.GOOS == "windows" {
			// On Windows, check for reparse point
			require.True(t, linkInfo.Mode()&os.ModeSymlink != 0, "Created path is not a symlink")
		} else {
			require.True(t, linkInfo.Mode()&os.ModeSymlink != 0, "Created path is not a symlink")
		}

		symlinkInfo := Info{
			Path:      symlinkPath,
			Timestamp: time.Now(),
			Size:      0,
		}
		err = manager.DeleteFile(ctx, symlinkInfo, false)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotRegularFile)

		// Verify the symlink still exists (since we shouldn't delete it)
		_, err = os.Lstat(symlinkPath)
		require.NoError(t, err, "Symlink was unexpectedly deleted")
	})

	t.Run("delete read-only file", func(t *testing.T) {
		// Create a read-only file
		readOnlyPath := filepath.Join(dir, "readonly.zip")
		_, err = os.Create(readOnlyPath)
		require.NoError(t, err)
		setReadOnly(t, readOnlyPath)

		readOnlyInfo := Info{
			Path:      readOnlyPath,
			Timestamp: time.Now(),
			Size:      0,
		}
		err = manager.DeleteFile(ctx, readOnlyInfo, false)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrAccessDenied)

		// Clean up by removing read-only attribute
		removeReadOnly(t, readOnlyPath)
	})

	t.Run("delete file with group write permission", func(t *testing.T) {
		// Create a file with group write permission
		groupWritePath := filepath.Join(dir, "group-write.zip")
		_, err = os.Create(groupWritePath)
		require.NoError(t, err)
		err = os.Chmod(groupWritePath, 0664) // rw-rw-r--
		require.NoError(t, err)

		groupWriteInfo := Info{
			Path:      groupWritePath,
			Timestamp: time.Now(),
			Size:      0,
		}
		err = manager.DeleteFile(ctx, groupWriteInfo, false)
		// This test might pass or fail depending on the user's group membership
		// We just verify it doesn't panic
		if err != nil {
			require.ErrorIs(t, err, ErrAccessDenied)
		}
	})

	t.Run("delete file with other write permission", func(t *testing.T) {
		// Create a file with other write permission
		otherWritePath := filepath.Join(dir, "other-write.zip")
		_, err = os.Create(otherWritePath)
		require.NoError(t, err)
		err = os.Chmod(otherWritePath, 0666) // rw-rw-rw-
		require.NoError(t, err)

		otherWriteInfo := Info{
			Path:      otherWritePath,
			Timestamp: time.Now(),
			Size:      0,
		}
		err = manager.DeleteFile(ctx, otherWriteInfo, false)
		// This test should pass since other has write permission
		require.NoError(t, err)
		_, err = os.Stat(otherWritePath)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete file with ACL write permission", func(t *testing.T) {
		if !checkACLSupport(t) {
			t.Skip("ACLs not supported on this filesystem")
		}

		// Create a file with restrictive base permissions
		aclPath := filepath.Join(dir, "acl-write.zip")
		_, err = os.Create(aclPath)
		require.NoError(t, err)
		err = os.Chmod(aclPath, 0400) // r--------
		require.NoError(t, err)

		// Get current user info
		uid := os.Getuid()
		gid := os.Getgid()

		// Create ACL entries
		// Note: This is a simplified test. In a real system, you'd need to:
		// 1. Get the current user's groups
		// 2. Create appropriate ACL entries for those groups
		// 3. Handle different ACL types (ACCESS, DEFAULT, etc.)
		aclEntries := []string{
			fmt.Sprintf("user:%d:rw-", uid),  // Give current user rw-
			fmt.Sprintf("group:%d:r--", gid), // Give current group r--
			"mask::rw-",                      // Set mask to rw-
			"other::r--",                     // Others get r--
		}

		// Set ACL using setfacl command
		// Note: This requires the acl package and setfacl command to be available
		cmd := exec.Command("setfacl", append([]string{"-m"}, aclEntries...)...)
		cmd.Args = append(cmd.Args, aclPath)
		if err := cmd.Run(); err != nil {
			t.Skip("setfacl command not available or failed:", err)
		}

		aclInfo := Info{
			Path:      aclPath,
			Timestamp: time.Now(),
			Size:      0,
		}

		// Try to delete the file
		// This should succeed because the ACL gives us write permission
		err = manager.DeleteFile(ctx, aclInfo, false)
		require.NoError(t, err)
		_, err = os.Stat(aclPath)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete file with ACL deny write", func(t *testing.T) {
		if !checkACLSupport(t) {
			t.Skip("ACLs not supported on this filesystem")
		}

		// Create a file with permissive base permissions
		aclPath := filepath.Join(dir, "acl-deny.zip")
		_, err = os.Create(aclPath)
		require.NoError(t, err)
		err = os.Chmod(aclPath, 0666) // rw-rw-rw-
		require.NoError(t, err)

		// Get current user info
		uid := os.Getuid()

		// Create ACL entries that deny write access
		aclEntries := []string{
			fmt.Sprintf("user:%d:r--", uid), // Give current user r-- only
			"mask::r--",                     // Set mask to r--
			"other::r--",                    // Others get r--
		}

		// Set ACL using setfacl command
		cmd := exec.Command("setfacl", append([]string{"-m"}, aclEntries...)...)
		cmd.Args = append(cmd.Args, aclPath)
		if err := cmd.Run(); err != nil {
			t.Skip("setfacl command not available or failed:", err)
		}

		aclInfo := Info{
			Path:      aclPath,
			Timestamp: time.Now(),
			Size:      0,
		}

		// Try to delete the file
		// This should fail because the ACL denies write permission
		err = manager.DeleteFile(ctx, aclInfo, false)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrAccessDenied)
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = manager.DeleteFile(cancelledCtx, info, false)
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("dry run", func(t *testing.T) {
		// Recreate the test file
		_, err = os.Create(path)
		require.NoError(t, err)

		err = manager.DeleteFile(ctx, info, true)
		require.NoError(t, err)
		_, err = os.Stat(path)
		require.NoError(t, err) // File should still exist
	})
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
			name: "valid case",
			matches: []string{
				"backup-202501010000.zip",
				"2025",
				"01",
				"01",
				"00",
				"00",
			},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expected:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expectErr:  false,
		},
		{
			name: "missing year",
			matches: []string{
				"backup--01010000.zip",
				"",
				"01",
				"01",
				"00",
				"00",
			},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
		{
			name: "invalid month",
			matches: []string{
				"backup-202513010000.zip",
				"2025",
				"13",
				"01",
				"00",
				"00",
			},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
		{
			name: "invalid day",
			matches: []string{
				"backup-202501320000.zip",
				"2025",
				"01",
				"32",
				"00",
				"00",
			},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
		{
			name: "invalid hour",
			matches: []string{
				"backup-202501012500.zip",
				"2025",
				"01",
				"01",
				"25",
				"00",
			},
			fieldNames: []string{"", "year", "month", "day", "hour", "minute"},
			expectErr:  true,
		},
		{
			name: "invalid minute",
			matches: []string{
				"backup-202501010060.zip",
				"2025",
				"01",
				"01",
				"00",
				"60",
			},
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

			timestamp, err := manager.parseTimestamp(
				testCase.matches,
				testCase.fieldNames,
			)

			if testCase.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.expected, timestamp)
			}
		})
	}
}
