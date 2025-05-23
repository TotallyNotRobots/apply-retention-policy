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

// Package file provides file system operations for managing backup files.
// It handles file discovery, filtering, and operations based on retention
// policies.
package file

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/TotallyNotRobots/apply-retention-policy/pkg/logger"
	"golang.org/x/sys/unix"
)

// Common errors
var (
	ErrInvalidPattern = errors.New("invalid file pattern")
	ErrListFiles      = errors.New("failed to list files")
	ErrParseTimestamp = errors.New("failed to parse timestamp")
	ErrDeleteFile     = errors.New("failed to delete file")
	ErrNotRegularFile = errors.New("not a regular file")
	ErrAccessDenied   = errors.New("access denied")
)

// Info represents a backup file with its parsed timestamp
type Info struct {
	Path      string
	Timestamp time.Time
	Size      int64
}

// ManagerOption is a function that configures a Manager
type ManagerOption func(*Manager)

// Manager handles file operations for the retention policy
type Manager struct {
	logger      *logger.Logger
	directory   string
	filePattern *regexp.Regexp
}

// WithLogger sets the logger for the Manager
func WithLogger(logger *logger.Logger) ManagerOption {
	return func(m *Manager) {
		m.logger = logger
	}
}

// NewManager creates a new file manager
func NewManager(
	directory, pattern string,
	opts ...ManagerOption,
) (*Manager, error) {
	// Convert the pattern to a regex pattern
	// Replace {year}, {month}, etc. with regex patterns
	regexPattern := pattern
	regexPattern = strings.ReplaceAll(regexPattern, "{year}", `(?P<year>\d{4})`)
	regexPattern = strings.ReplaceAll(
		regexPattern,
		"{month}",
		`(?P<month>\d{2})`,
	)
	regexPattern = strings.ReplaceAll(regexPattern, "{day}", `(?P<day>\d{2})`)
	regexPattern = strings.ReplaceAll(regexPattern, "{hour}", `(?P<hour>\d{2})`)
	regexPattern = strings.ReplaceAll(
		regexPattern,
		"{minute}",
		`(?P<minute>\d{2})`,
	)
	regexPattern = "^" + regexPattern + "$"

	compiledPattern, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidPattern, err)
	}

	// Create manager with default values
	m := &Manager{
		logger: &logger.Logger{
			Logger: zap.NewNop(),
		}, // Default no-op logger
		directory:   directory,
		filePattern: compiledPattern,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

// isRegularFile checks if the file is a regular file and the current user has write access
func (m *Manager) isRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDeleteFile, err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is not a regular file", ErrNotRegularFile, path)
	}

	// Check if we have write permission using os.Access
	// This properly handles group and other permissions
	if err := unix.Access(path, unix.W_OK); err != nil {
		return fmt.Errorf("%w: no write permission for %s", ErrAccessDenied, path)
	}

	return nil
}

// DeleteFile deletes a file and logs the operation
func (m *Manager) DeleteFile(
	ctx context.Context,
	file Info,
	dryRun bool,
) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if dryRun {
		m.logger.Info("would delete file (dry run)",
			zap.String("file", file.Path),
			zap.Time("timestamp", file.Timestamp),
			zap.Int64("size", file.Size))

		return nil
	}

	// Verify file safety before deletion
	if err := m.isRegularFile(file.Path); err != nil {
		return err
	}

	if err := os.Remove(file.Path); err != nil {
		return fmt.Errorf("%w %s: %w", ErrDeleteFile, file.Path, err)
	}

	m.logger.Info("deleted file",
		zap.String("file", file.Path),
		zap.Time("timestamp", file.Timestamp),
		zap.Int64("size", file.Size))

	return nil
}

// processFile processes a single file and adds it to the files slice if it matches the pattern
func (m *Manager) processFile(
	ctx context.Context,
	path string,
	d os.DirEntry,
	files *[]Info,
) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Skip directories and symlinks
	if d.IsDir() || d.Type()&os.ModeSymlink != 0 {
		return nil
	}

	// Get relative path from the backup directory
	relPath, err := filepath.Rel(m.directory, path)
	if err != nil {
		return err
	}

	// Check if the file matches our pattern
	matches := m.filePattern.FindStringSubmatch(relPath)
	if matches == nil {
		return nil
	}

	// Get file info for size and type
	info, err := d.Info()
	if err != nil {
		m.logger.Warn("failed to get file info",
			zap.String("file", relPath),
			zap.Error(err))

		return nil
	}

	// Skip if not a regular file
	if !info.Mode().IsRegular() {
		m.logger.Debug("skipping non-regular file",
			zap.String("file", relPath),
			zap.String("mode", info.Mode().String()))
		return nil
	}

	// Parse the timestamp from the filename
	timestamp, err := m.parseTimestamp(matches, m.filePattern.SubexpNames())
	if err != nil {
		m.logger.Warn("failed to parse timestamp from filename",
			zap.String("file", relPath),
			zap.Error(err))

		return nil
	}

	*files = append(*files, Info{
		Path:      path,
		Timestamp: timestamp,
		Size:      info.Size(),
	})

	return nil
}

// ListFiles lists all files in the directory that match the pattern
func (m *Manager) ListFiles(ctx context.Context) ([]Info, error) {
	// Check for context cancellation first
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var files []Info

	err := filepath.WalkDir(m.directory, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		return m.processFile(ctx, path, d, &files)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListFiles, err)
	}

	// Sort files by timestamp (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Timestamp.After(files[j].Timestamp)
	})

	return files, nil
}

// parseTimestamp parses the timestamp from the regex matches
func (m *Manager) parseTimestamp(
	matches []string,
	fieldNames []string,
) (time.Time, error) {
	if len(matches) != len(fieldNames) {
		return time.Time{}, fmt.Errorf(
			"%w: mismatch between matches and fieldNames: got %d matches, expected %d",
			ErrParseTimestamp,
			len(matches),
			len(fieldNames),
		)
	}

	// Prepare default values
	parts := map[string]string{
		"year":   "0000",
		"month":  "01",
		"day":    "01",
		"hour":   "00",
		"minute": "00",
	}

	// Fill values from matches
	for field, defaultVal := range parts {
		if idx := slices.Index(fieldNames, field); idx >= 0 {
			parts[field] = matches[idx]
		} else {
			parts[field] = defaultVal
		}
	}

	// Format timestamp string
	timestampStr := fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		parts["year"],
		parts["month"],
		parts["day"],
		parts["hour"],
		parts["minute"],
	)

	// Parse the timestamp
	timestamp, err := time.Parse("2006-01-02-15-04", timestampStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %w", ErrParseTimestamp, err)
	}

	return timestamp, nil
}
