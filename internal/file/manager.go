package file

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/TotallyNotRobots/apply-retention-policy/pkg/logger"
	"go.uber.org/zap"
)

// Info represents a backup file with its parsed timestamp
type Info struct {
	Path      string
	Timestamp time.Time
	Size      int64
}

// Manager handles file operations for the retention policy
type Manager struct {
	logger      *logger.Logger
	directory   string
	filePattern *regexp.Regexp
}

// NewManager creates a new file manager
func NewManager(logger *logger.Logger, directory, pattern string) (*Manager, error) {
	// Convert the pattern to a regex pattern
	// Replace {year}, {month}, etc. with regex patterns
	regexPattern := pattern
	regexPattern = strings.ReplaceAll(regexPattern, "{year}", `(\d{4})`)
	regexPattern = strings.ReplaceAll(regexPattern, "{month}", `(\d{2})`)
	regexPattern = strings.ReplaceAll(regexPattern, "{day}", `(\d{2})`)
	regexPattern = strings.ReplaceAll(regexPattern, "{hour}", `(\d{2})`)
	regexPattern = strings.ReplaceAll(regexPattern, "{minute}", `(\d{2})`)
	regexPattern = "^" + regexPattern + "$"

	compiledPattern, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid file pattern: %w", err)
	}

	return &Manager{
		logger:      logger,
		directory:   directory,
		filePattern: compiledPattern,
	}, nil
}

// ListFiles returns a list of backup files sorted by timestamp
func (m *Manager) ListFiles() ([]Info, error) {
	var files []Info

	err := filepath.Walk(m.directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
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

		// Parse the timestamp from the filename
		timestamp, err := m.parseTimestamp(matches)
		if err != nil {
			m.logger.Warn("failed to parse timestamp from filename",
				zap.String("file", relPath),
				zap.Error(err))
			return nil
		}

		files = append(files, Info{
			Path:      path,
			Timestamp: timestamp,
			Size:      info.Size(),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Sort files by timestamp (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Timestamp.After(files[j].Timestamp)
	})

	return files, nil
}

// DeleteFile deletes a file and logs the operation
func (m *Manager) DeleteFile(file Info, dryRun bool) error {
	if dryRun {
		m.logger.Info("would delete file (dry run)",
			zap.String("file", file.Path),
			zap.Time("timestamp", file.Timestamp),
			zap.Int64("size", file.Size))
		return nil
	}

	if err := os.Remove(file.Path); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", file.Path, err)
	}

	m.logger.Info("deleted file",
		zap.String("file", file.Path),
		zap.Time("timestamp", file.Timestamp),
		zap.Int64("size", file.Size))

	return nil
}

// parseTimestamp parses the timestamp from the regex matches
func (m *Manager) parseTimestamp(matches []string) (time.Time, error) {
	// The first match is the full string, so we start from index 1
	// We expect the pattern to be: year, month, day, hour, minute
	if len(matches) < 6 {
		return time.Time{}, fmt.Errorf("invalid number of matches: %d", len(matches))
	}

	year := matches[1]
	month := matches[2]
	day := matches[3]
	hour := matches[4]
	minute := matches[5]

	// Parse the timestamp
	timestamp, err := time.Parse("2006-01-02-15-04",
		fmt.Sprintf("%s-%s-%s-%s-%s", year, month, day, hour, minute))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return timestamp, nil
}
