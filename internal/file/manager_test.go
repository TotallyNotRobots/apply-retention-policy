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
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/TotallyNotRobots/apply-retention-policy/pkg/logger"
	"github.com/TotallyNotRobots/apply-retention-policy/pkg/util"
)

func init() {
	// Enable test mode for logger in all tests
	logger.SetTestMode(true)
}

const (
	testBackupPattern = "backup-{year}{month}{day}{hour}{minute}.zip"
)

// setReadOnly makes a file read-only in a platform-independent way
func setReadOnly(t *testing.T, path string) {
	plat := util.NewPlatform()
	err := plat.SetReadOnly(path)
	require.NoError(t, err, "Failed to set read-only attribute")
}

// removeReadOnly removes read-only attribute in a platform-independent way
func removeReadOnly(t *testing.T, path string) {
	plat := util.NewPlatform()
	err := plat.RemoveReadOnly(path)
	require.NoError(t, err, "Failed to remove read-only attribute")
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
		require.NotNil(t, m)
		require.Equal(t, "/tmp", m.directory)
		require.NotNil(t, m.filePattern)
	})

	t.Run("invalid pattern", func(t *testing.T) {
		_, err := NewManager("/tmp", "backup-[invalid")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidPattern)
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
		require.NotNil(t, m.logger)
	})
}

func mkfifo(path string, mode uint32) error {
	plat := util.NewPlatform()
	return plat.Mkfifo(path, mode)
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

		var f *os.File
		f, err = os.Create(path)
		require.NoError(t, err)

		// Ensure file handle is closed immediately after creation
		err = f.Close()
		require.NoError(t, err)
	}

	// Create a directory
	dirPath := filepath.Join(dir, "subdir")
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	// Platform-specific tests
	plat := util.NewPlatform()

	// Symlink test
	symlinkSupport, _ := plat.CheckSymlinkSupport()
	if symlinkSupport {
		t.Run("with symlink", func(t *testing.T) {
			// Create a symlink
			symlinkPath := filepath.Join(dir, "symlink")

			err = os.Symlink(filepath.Join(dir, files[0]), symlinkPath)
			if err != nil {
				t.Skip("Symlink creation failed - may need elevated privileges")
			}

			// List files and verify symlink is not included
			list, listErr := manager.ListFiles(ctx)
			require.NoError(t, listErr)
			require.Len(t, list, len(files))

			for _, file := range list {
				base := filepath.Base(file.Path)
				require.Contains(t, files, base)
				require.NotEqual(t, "symlink", base, "Symlink should not be included in results")
			}
		})
	}

	// FIFO test
	fifoSupport, _ := plat.CheckFIFOSupport()
	if fifoSupport {
		t.Run("with named pipe", func(t *testing.T) {
			// Create a named pipe
			pipePath := filepath.Join(dir, "pipe")
			err = mkfifo(pipePath, 0644)
			require.NoError(t, err)

			// Verify the pipe exists and is a FIFO
			pipeInfo, pipeErr := os.Lstat(pipePath)
			require.NoError(t, pipeErr)
			require.True(
				t,
				pipeInfo.Mode()&os.ModeNamedPipe != 0,
				"Created path is not a named pipe",
			)

			// List files and verify the pipe is not included
			list, listErr := manager.ListFiles(ctx)
			require.NoError(t, listErr)
			require.Len(t, list, len(files))

			for _, file := range list {
				base := filepath.Base(file.Path)
				require.Contains(t, files, base)
				require.NotEqual(t, "pipe", base, "Named pipe should not be included in results")
			}

			// Clean up the pipe
			err = os.Remove(pipePath)
			require.NoError(t, err)
		})
	}

	// Basic test that works on all platforms
	t.Run("list regular files", func(t *testing.T) {
		list, listErr := manager.ListFiles(ctx)
		require.NoError(t, listErr)
		require.Len(t, list, len(files))

		for _, file := range list {
			base := filepath.Base(file.Path)
			require.Contains(t, files, base)
		}
	})

	// Test with cancelled context
	t.Run("cancelled context", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = manager.ListFiles(cancelledCtx)
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
	})
}

// setupTestFile creates a test file and returns its path and info
func setupTestFile(t *testing.T, dir string, filename string) (string, Info) {
	path := filepath.Clean(filepath.Join(dir, filename))

	var f *os.File

	var err error

	f, err = os.Create(path)
	require.NoError(t, err)

	// Ensure file handle is closed immediately after creation
	err = f.Close()
	require.NoError(t, err)

	return path, Info{
		Path:      path,
		Timestamp: time.Now(),
		Size:      0,
	}
}

// verifyACLs checks if the specified ACL entries are present in the file's ACLs
func verifyACLs(t *testing.T, path string, entriesToCheck [][]string) {
	const getfaclPath = "/usr/bin/getfacl"
	if _, err := os.Stat(getfaclPath); err != nil {
		t.Skip("getfacl command not available")
	}

	cmd := exec.Command(getfaclPath, filepath.Clean(path)) // #nosec G204
	output, err := cmd.Output()

	if err != nil {
		t.Skip("getfacl command failed:", err)
	}

	outputStr := string(output)

	for _, entryAlts := range entriesToCheck {
		found := false

		for _, alt := range entryAlts {
			if strings.Contains(outputStr, alt) {
				found = true
				break
			}
		}

		if !found {
			t.Skipf("None of the ACL entry alternatives %v found in getfacl output", entryAlts)
		}
	}
}

// setACLs sets ACL entries on a file using setfacl
func setACLs(t *testing.T, path string, aclEntries []string) {
	const setfaclPath = "/usr/bin/setfacl"
	if _, err := os.Stat(setfaclPath); err != nil {
		t.Skip("setfacl command not available")
	}

	cmd := exec.Command(
		setfaclPath,
		"-m",
		strings.Join(aclEntries, ","),
		filepath.Clean(path),
	) // #nosec G204
	if err := cmd.Run(); err != nil {
		t.Skip("setfacl command failed:", err)
	}
}

func testDeleteRegularFile(
	ctx context.Context,
	t *testing.T,
	manager *Manager,
	path string,
	info Info,
) {
	// Ensure any open file handles are closed before deletion
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)

	err = manager.DeleteFile(ctx, info, false)
	require.NoError(t, err)
	_, err = os.Stat(path)
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func testDeleteNonExistentFile(ctx context.Context, t *testing.T, manager *Manager, dir string) {
	nonExistentInfo := Info{
		Path:      filepath.Join(dir, "non-existent.zip"),
		Timestamp: time.Now(),
		Size:      0,
	}
	err := manager.DeleteFile(ctx, nonExistentInfo, false)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDeleteFile)
}

func testDeleteDirectory(ctx context.Context, t *testing.T, manager *Manager, dir string) {
	dirPath := filepath.Join(dir, "testdir")
	err := os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	dirInfo := Info{
		Path:      dirPath,
		Timestamp: time.Now(),
		Size:      0,
	}
	err = manager.DeleteFile(ctx, dirInfo, false)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotRegularFile)
}

func testDeleteSymlink(ctx context.Context, t *testing.T, manager *Manager, dir string) {
	if !checkSymlinkSupport() {
		t.Skip("Symlinks not supported on this system")
	}

	targetPath := filepath.Join(dir, "target.zip")
	_, err := os.Create(targetPath)
	require.NoError(t, err)

	symlinkPath := filepath.Join(dir, "symlink")
	err = os.Symlink(targetPath, symlinkPath)

	if err != nil {
		t.Skip("Symlink creation failed - may need elevated privileges")
	}

	linkInfo, err := os.Lstat(symlinkPath)
	require.NoError(t, err)
	require.True(t, linkInfo.Mode()&os.ModeSymlink != 0, "Created path is not a symlink")

	symlinkInfo := Info{
		Path:      symlinkPath,
		Timestamp: time.Now(),
		Size:      0,
	}
	err = manager.DeleteFile(ctx, symlinkInfo, false)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotRegularFile)
	_, err = os.Lstat(symlinkPath)
	require.NoError(t, err, "Symlink was unexpectedly deleted")
}

func testDeleteReadOnlyFile(ctx context.Context, t *testing.T, manager *Manager, dir string) {
	readOnlyPath := filepath.Join(dir, "readonly.zip")
	_, err := os.Create(readOnlyPath)
	require.NoError(t, err)
	setReadOnly(t, readOnlyPath)
	chownErr := os.Chown(readOnlyPath, 65534, -1)

	if chownErr != nil {
		t.Skipf("Skipping test: unable to chown file to another user: %v", chownErr)
	}

	readOnlyInfo := Info{
		Path:      readOnlyPath,
		Timestamp: time.Now(),
		Size:      0,
	}
	err = manager.DeleteFile(ctx, readOnlyInfo, false)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAccessDenied)
	removeReadOnly(t, readOnlyPath)
}

func testDeleteFileWithGroupWrite(ctx context.Context, t *testing.T, manager *Manager, dir string) {
	groupWritePath := filepath.Join(dir, "group-write.zip")
	_, err := os.Create(groupWritePath)
	require.NoError(t, err)
	err = os.Chmod(groupWritePath, 0664)
	require.NoError(t, err)

	groupWriteInfo := Info{
		Path:      groupWritePath,
		Timestamp: time.Now(),
		Size:      0,
	}
	err = manager.DeleteFile(ctx, groupWriteInfo, false)

	if err != nil {
		require.ErrorIs(t, err, ErrAccessDenied)
	}
}

func testDeleteFileWithOtherWrite(ctx context.Context, t *testing.T, manager *Manager, dir string) {
	otherWritePath := filepath.Join(dir, "other-write.zip")

	var f *os.File

	var err error

	f, err = os.Create(otherWritePath)
	require.NoError(t, err)

	// Ensure file handle is closed immediately after creation
	err = f.Close()
	require.NoError(t, err)

	err = os.Chmod(otherWritePath, 0666)
	require.NoError(t, err)

	otherWriteInfo := Info{
		Path:      otherWritePath,
		Timestamp: time.Now(),
		Size:      0,
	}

	// Ensure any open file handles are closed before deletion
	f, err = os.OpenFile(otherWritePath, os.O_RDONLY, 0)
	if err == nil {
		err = f.Close()
		require.NoError(t, err)
	}

	err = manager.DeleteFile(ctx, otherWriteInfo, false)
	require.NoError(t, err)
	_, err = os.Stat(otherWritePath)
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func checkACLSupport() bool {
	plat := util.NewPlatform()

	support, err := plat.CheckACLSupport()
	if err != nil {
		return false
	}

	return support
}

func checkSymlinkSupport() bool {
	plat := util.NewPlatform()

	support, err := plat.CheckSymlinkSupport()
	if err != nil {
		return false
	}

	return support
}

func testDeleteFileWithACLWrite(ctx context.Context, t *testing.T, manager *Manager, dir string) {
	if !checkACLSupport() {
		t.Skip("ACLs not supported on this filesystem")
	}

	uid := os.Getuid()
	gid := os.Getgid()
	userObj, _ := user.LookupId(fmt.Sprintf("%d", uid))
	groupObj, _ := user.LookupGroupId(fmt.Sprintf("%d", gid))
	username := fmt.Sprintf("%d", uid)

	if userObj != nil {
		username = userObj.Username
	}

	groupname := fmt.Sprintf("%d", gid)

	if groupObj != nil {
		groupname = groupObj.Name
	}

	t.Logf("Setting ACLs for user %s (uid %d) and group %s (gid %d)", username, uid, groupname, gid)

	aclPath := filepath.Join(dir, "acl-write.zip")
	_, err := os.Create(aclPath)
	require.NoError(t, err)
	err = os.Chmod(aclPath, 0400)
	require.NoError(t, err)

	aclEntries := []string{
		fmt.Sprintf("user:%d:rw-", uid),
		fmt.Sprintf("group:%d:r--", gid),
		"mask::rw-",
		"other::r--",
	}
	setACLs(t, aclPath, aclEntries)

	entriesToCheck := [][]string{
		{fmt.Sprintf("user:%d:rw-", uid), fmt.Sprintf("user:%s:rw-", username)},
		{fmt.Sprintf("group:%d:r--", gid), fmt.Sprintf("group:%s:r--", groupname)},
		{"mask::rw-"},
		{"other::r--"},
	}
	verifyACLs(t, aclPath, entriesToCheck)

	// Check write permission using os.Access instead of unix.Access
	_, err = os.OpenFile(aclPath, os.O_WRONLY, 0)
	if err != nil {
		t.Logf("Warning: os.OpenFile reports no write permission: %v", err)
	} else {
		t.Log("os.OpenFile reports write permission is available")
	}

	aclInfo := Info{
		Path:      aclPath,
		Timestamp: time.Now(),
		Size:      0,
	}
	err = manager.DeleteFile(ctx, aclInfo, false)

	if err != nil {
		t.Logf("DeleteFile failed with error: %v", err)
	}

	require.NoError(t, err)
	_, err = os.Stat(aclPath)
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func testDeleteFileWithACLDenyWrite(
	ctx context.Context,
	t *testing.T,
	manager *Manager,
	dir string,
) {
	if !checkACLSupport() {
		t.Skip("ACLs not supported on this filesystem")
	}

	uid := os.Getuid()
	userObj, _ := user.LookupId(fmt.Sprintf("%d", uid))
	username := fmt.Sprintf("%d", uid)

	if userObj != nil {
		username = userObj.Username
	}

	t.Logf("Setting ACLs for user %s (uid %d)", username, uid)

	aclPath := filepath.Join(dir, "acl-deny.zip")
	_, err := os.Create(aclPath)
	require.NoError(t, err)
	err = os.Chmod(aclPath, 0666)
	require.NoError(t, err)

	aclEntries := []string{
		fmt.Sprintf("user:%d:r--", uid),
		"mask::r--",
		"other::r--",
	}
	setACLs(t, aclPath, aclEntries)

	entriesToCheck := [][]string{
		{fmt.Sprintf("user:%d:r--", uid), fmt.Sprintf("user:%s:r--", username)},
		{"mask::r--"},
		{"other::r--"},
	}
	verifyACLs(t, aclPath, entriesToCheck)
	chownErr := os.Chown(aclPath, 65534, -1)

	if chownErr != nil {
		t.Skipf("Skipping test: unable to chown file to another user: %v", chownErr)
	}

	// Check write permission using os.Access instead of unix.Access
	_, err = os.OpenFile(aclPath, os.O_WRONLY, 0)
	if err != nil {
		t.Log("os.OpenFile reports no write permission (expected)")
	} else {
		t.Log("Warning: os.OpenFile reports write permission is available (unexpected)")
	}

	aclInfo := Info{
		Path:      aclPath,
		Timestamp: time.Now(),
		Size:      0,
	}
	err = manager.DeleteFile(ctx, aclInfo, false)

	if err == nil {
		t.Log("DeleteFile succeeded when it should have failed")
	} else {
		t.Logf("DeleteFile failed with error: %v", err)
	}

	require.Error(t, err)
	require.ErrorIs(t, err, ErrAccessDenied)
}

func testContextCancellation(t *testing.T, manager *Manager, info Info) {
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := manager.DeleteFile(cancelledCtx, info, false)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}

func testDryRun(ctx context.Context, t *testing.T, manager *Manager, path string, info Info) {
	err := manager.DeleteFile(ctx, info, true)
	require.NoError(t, err)
	_, err = os.Stat(path)
	require.NoError(t, err)
}

func TestDeleteFile(t *testing.T) {
	t.Parallel()

	// Basic file operations that work on all platforms
	t.Run("delete regular file", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()
		log := &logger.Logger{Logger: zap.NewNop()}
		manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
		require.NoError(t, err)
		path, info := setupTestFile(t, dir, "backup-202501010000.zip")
		testDeleteRegularFile(ctx, t, manager, path, info)
	})
	t.Run("delete non-existent file", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()
		log := &logger.Logger{Logger: zap.NewNop()}
		manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
		require.NoError(t, err)
		testDeleteNonExistentFile(ctx, t, manager, dir)
	})
	t.Run("delete directory", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()
		log := &logger.Logger{Logger: zap.NewNop()}
		manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
		require.NoError(t, err)
		testDeleteDirectory(ctx, t, manager, dir)
	})
	t.Run("delete file with other write permission", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()
		log := &logger.Logger{Logger: zap.NewNop()}
		manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
		require.NoError(t, err)
		testDeleteFileWithOtherWrite(ctx, t, manager, dir)
	})
	t.Run("context cancellation", func(t *testing.T) {
		dir := t.TempDir()
		log := &logger.Logger{Logger: zap.NewNop()}
		manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
		require.NoError(t, err)
		_, info := setupTestFile(t, dir, "backup-202501010000.zip")
		testContextCancellation(t, manager, info)
	})
	t.Run("dry run", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()
		log := &logger.Logger{Logger: zap.NewNop()}
		manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
		require.NoError(t, err)
		path, info := setupTestFile(t, dir, "backup-202501010000.zip")
		testDryRun(ctx, t, manager, path, info)
	})

	// Platform-specific tests that should be skipped on Windows
	plat := util.NewPlatform()

	symlinkSupport, _ := plat.CheckSymlinkSupport()
	if symlinkSupport {
		t.Run("delete symlink", func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			log := &logger.Logger{Logger: zap.NewNop()}
			manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
			require.NoError(t, err)
			testDeleteSymlink(ctx, t, manager, dir)
		})
	}

	aclSupport, _ := plat.CheckACLSupport()
	if aclSupport {
		t.Run("delete file with ACL write permission", func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			log := &logger.Logger{Logger: zap.NewNop()}
			manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
			require.NoError(t, err)
			testDeleteFileWithACLWrite(ctx, t, manager, dir)
		})
		t.Run("delete file with ACL deny write", func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			log := &logger.Logger{Logger: zap.NewNop()}
			manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
			require.NoError(t, err)
			testDeleteFileWithACLDenyWrite(ctx, t, manager, dir)
		})
	}

	// Group write test may fail on Windows due to different permission model
	if runtime.GOOS != "windows" {
		t.Run("delete file with group write permission", func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			log := &logger.Logger{Logger: zap.NewNop()}
			manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
			require.NoError(t, err)
			testDeleteFileWithGroupWrite(ctx, t, manager, dir)
		})
	}

	// Read-only test may need platform-specific handling
	if runtime.GOOS != "windows" {
		t.Run("delete read-only file", func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			log := &logger.Logger{Logger: zap.NewNop()}
			manager, err := NewManager(dir, testBackupPattern, WithLogger(log))
			require.NoError(t, err)
			testDeleteReadOnlyFile(ctx, t, manager, dir)
		})
	}
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
				require.Equal(t, testCase.expected, timestamp)
			}
		})
	}
}
