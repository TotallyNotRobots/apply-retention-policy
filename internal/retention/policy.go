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

// Package retention provides functionality for applying retention policies to
// backup files. It implements various retention strategies based on time
// periods (hourly, daily, weekly, monthly, yearly).
package retention

import (
	"slices"
	"time"

	"go.uber.org/zap"

	"github.com/TotallyNotRobots/apply-retention-policy/internal/config"
	"github.com/TotallyNotRobots/apply-retention-policy/internal/file"
	"github.com/TotallyNotRobots/apply-retention-policy/pkg/log"
)

// Policy implements the retention policy logic
type Policy struct {
	logger *log.Logger
	config *config.Config
}

// NewPolicy creates a new retention policy
func NewPolicy(logger *log.Logger, conf *config.Config) *Policy {
	return &Policy{
		logger: logger,
		config: conf,
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
	hourlyFiles := groupFilesByPeriod(
		files,
		hourGrouper,
		p.config.Retention.Hourly,
	)
	toDelete = append(toDelete, hourlyFiles.toDelete...)

	dailyFiles := groupFilesByPeriod(
		hourlyFiles.unselected,
		dayGrouper,
		p.config.Retention.Daily,
	)
	toDelete = append(toDelete, dailyFiles.toDelete...)

	weeklyFiles := groupFilesByPeriod(
		dailyFiles.unselected,
		weekGrouper,
		p.config.Retention.Weekly,
	)
	toDelete = append(toDelete, weeklyFiles.toDelete...)

	monthlyFiles := groupFilesByPeriod(
		weeklyFiles.unselected,
		monthGrouper,
		p.config.Retention.Monthly,
	)
	toDelete = append(toDelete, monthlyFiles.toDelete...)

	yearlyFiles := groupFilesByPeriod(
		monthlyFiles.unselected,
		yearGrouper,
		p.config.Retention.Yearly,
	)
	toDelete = append(toDelete, yearlyFiles.toDelete...)

	// All extra files should be pruned
	toDelete = append(toDelete, yearlyFiles.unselected...)

	// Log summary
	p.logger.Info("retention policy summary",
		zap.Int("total_files", len(files)),
		zap.Int("files_to_delete", len(toDelete)),
		zap.Int("hourly_retained", len(hourlyFiles.selected)),
		zap.Int("daily_retained", len(dailyFiles.selected)),
		zap.Int("weekly_retained", len(weeklyFiles.selected)),
		zap.Int("monthly_retained", len(monthlyFiles.selected)),
		zap.Int("yearly_retained", len(yearlyFiles.selected)))

	return toDelete, nil
}

// groupFilesByTimePeriod groups files into time periods based on the given
// duration. Files are sorted by timestamp in descending order and grouped by
// their time period. Returns a slice of file groups, where each group contains
// files from the same time period.
func groupFilesByTimePeriod[T comparable](
	files []file.Info,
	grouper func(file.Info) T,
) [][]file.Info {
	var groups [][]file.Info

	currentGroup := []file.Info{}

	files = slices.Clone(files)
	slices.SortFunc(files, func(a, b file.Info) int {
		return b.Timestamp.Compare(a.Timestamp)
	})

	for _, f := range files {
		// If this is the first file or it's in the same period as the previous
		// file
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

type groupResult struct {
	selected   []file.Info
	toDelete   []file.Info
	unselected []file.Info
}

// groupFilesByPeriod groups files by the specified time period
func groupFilesByPeriod[T comparable](
	files []file.Info,
	grouper func(file.Info) T,
	keepCount int,
) *groupResult {
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

	return &groupResult{
		selected:   selected,
		toDelete:   toDelete,
		unselected: unselected,
	}
}
