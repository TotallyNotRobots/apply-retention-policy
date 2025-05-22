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

// Package retention implements retention policy logic for backup files.
// It provides functionality for evaluating and applying retention rules to backup files.
package retention

import (
	"slices"
	"time"

	"go.uber.org/zap"

	"github.com/TotallyNotRobots/apply-retention-policy/internal/config"
	"github.com/TotallyNotRobots/apply-retention-policy/internal/file"
	"github.com/TotallyNotRobots/apply-retention-policy/pkg/logger"
)

// Policy implements the retention policy logic
type Policy struct {
	logger *logger.Logger
	config *config.Config
}

// NewPolicy creates a new retention policy
func NewPolicy(logger *logger.Logger, config *config.Config) *Policy {
	return &Policy{
		logger: logger,
		config: config,
	}
}

// weekMultiplier is used to combine year and week numbers into a single integer
// by multiplying the year by 100 and adding the week number
const weekMultiplier = 100

// grouper functions for different time periods
var (
	// hourGrouper groups files by hour
	hourGrouper = func(f file.Info) int64 {
		return time.Date(
			f.Timestamp.Year(),
			f.Timestamp.Month(),
			f.Timestamp.Day(),
			f.Timestamp.Hour(),
			0,
			0,
			0,
			f.Timestamp.Location(),
		).Unix()
	}

	// dayGrouper groups files by day
	dayGrouper = func(f file.Info) int64 {
		return time.Date(
			f.Timestamp.Year(),
			f.Timestamp.Month(),
			f.Timestamp.Day(),
			0,
			0,
			0,
			0,
			f.Timestamp.Location(),
		).Unix()
	}

	// weekGrouper groups files by ISO week
	weekGrouper = func(f file.Info) int {
		year, week := f.Timestamp.ISOWeek()
		return (year * weekMultiplier) + week
	}

	// monthGrouper groups files by month
	monthGrouper = func(f file.Info) int64 {
		return time.Date(
			f.Timestamp.Year(),
			f.Timestamp.Month(),
			1,
			0,
			0,
			0,
			0,
			f.Timestamp.Location(),
		).Unix()
	}

	// yearGrouper groups files by year
	yearGrouper = func(f file.Info) int64 {
		return time.Date(
			f.Timestamp.Year(),
			1,
			1,
			0,
			0,
			0,
			0,
			f.Timestamp.Location(),
		).Unix()
	}
)

// Apply applies the retention policy to the given files
func (p *Policy) Apply(files []file.Info) ([]file.Info, error) {
	if len(files) == 0 {
		return nil, nil
	}

	var toDelete []file.Info

	// Group files by time period
	hourlyFiles, pruned, files := groupFilesByPeriod(
		files,
		hourGrouper,
		p.config.Retention.Hourly,
	)
	toDelete = append(toDelete, pruned...)

	dailyFiles, pruned, files := groupFilesByPeriod(
		files,
		dayGrouper,
		p.config.Retention.Daily,
	)
	toDelete = append(toDelete, pruned...)

	weeklyFiles, pruned, files := groupFilesByPeriod(
		files,
		weekGrouper,
		p.config.Retention.Weekly,
	)
	toDelete = append(toDelete, pruned...)

	monthlyFiles, pruned, files := groupFilesByPeriod(
		files,
		monthGrouper,
		p.config.Retention.Monthly,
	)
	toDelete = append(toDelete, pruned...)

	yearlyFiles, pruned, files := groupFilesByPeriod(
		files,
		yearGrouper,
		p.config.Retention.Yearly,
	)
	toDelete = append(toDelete, pruned...)

	// All extra files should be pruned
	toDelete = append(toDelete, files...)

	// Log summary
	p.logger.Info("retention policy summary",
		zap.Int("total_files", len(files)),
		zap.Int("files_to_delete", len(toDelete)),
		zap.Int("hourly_retained", len(hourlyFiles)),
		zap.Int("daily_retained", len(dailyFiles)),
		zap.Int("weekly_retained", len(weeklyFiles)),
		zap.Int("monthly_retained", len(monthlyFiles)),
		zap.Int("yearly_retained", len(yearlyFiles)))

	return toDelete, nil
}

// groupFilesByTimePeriod groups files into time periods based on the given duration.
// Files are sorted by timestamp in descending order and grouped by their time period.
// Returns a slice of file groups, where each group contains files from the same time period.
func groupFilesByTimePeriod[T comparable](
	files []file.Info,
	grouper func(file.Info) T,
) [][]file.Info {
	var groups [][]file.Info
	currentGroup := []file.Info{}

	slices.SortFunc(files, func(a, b file.Info) int {
		return b.Timestamp.Compare(a.Timestamp)
	})

	for _, f := range files {
		// If this is the first file or it's in the same period as the previous file
		if len(currentGroup) == 0 ||
			grouper(f) == grouper(currentGroup[0]) {
			currentGroup = append(currentGroup, f)
		} else {
			// Start a new group
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
			}
			currentGroup = []file.Info{f}
		}
	}

	// Add the last group
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// groupFilesByPeriod groups files by the specified time period
func groupFilesByPeriod[T comparable](
	files []file.Info,
	grouper func(file.Info) T,
	keepCount int,
) ([]file.Info, []file.Info, []file.Info) {
	groups := groupFilesByTimePeriod(files, grouper)

	selected := []file.Info{}
	unselected := []file.Info{}
	toDelete := []file.Info{}

	for _, group := range groups {
		if len(selected) == keepCount {
			unselected = append(unselected, group...)
		} else {
			selected = append(selected, group[0])

			if len(group) > 1 {
				toDelete = append(toDelete, group[1:]...)
			}
		}
	}

	return selected, toDelete, unselected
}
