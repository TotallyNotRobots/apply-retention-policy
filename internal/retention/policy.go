package retention

import (
	"slices"
	"time"

	"go.uber.org/zap"

	"github.com/TotallyNotRobots/apply-retention-policy/internal/config"
	"github.com/TotallyNotRobots/apply-retention-policy/internal/consts"
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

// Apply applies the retention policy to the given files
func (p *Policy) Apply(files []file.Info, now time.Time) ([]file.Info, error) {
	if len(files) == 0 {
		return nil, nil
	}

	var toDelete []file.Info

	// Group files by time period
	hourlyFiles, pruned, files := p.groupFilesByPeriod(
		files,
		now,
		consts.HOUR,
		p.config.Retention.Hourly,
	)
	toDelete = append(toDelete, pruned...)

	dailyFiles, pruned, files := p.groupFilesByPeriod(
		files,
		now,
		consts.DAY,
		p.config.Retention.Daily,
	)
	toDelete = append(toDelete, pruned...)

	weeklyFiles, pruned, files := p.groupFilesByPeriod(
		files,
		now,
		consts.WEEK,
		p.config.Retention.Weekly,
	)
	toDelete = append(toDelete, pruned...)

	monthlyFiles, pruned, files := p.groupFilesByPeriod(
		files,
		now,
		consts.MONTH,
		p.config.Retention.Monthly,
	)
	toDelete = append(toDelete, pruned...)

	yearlyFiles, pruned, files := p.groupFilesByPeriod(
		files,
		now,
		consts.YEAR,
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

// groupFilesByPeriod groups files by the specified time period
func (p *Policy) groupFilesByPeriod(
	files []file.Info,
	now time.Time,
	period time.Duration,
	keepCount int,
) ([]file.Info, []file.Info, []file.Info) {
	var groups [][]file.Info
	currentGroup := []file.Info{}

	slices.SortFunc(files, func(a, b file.Info) int {
		return b.Timestamp.Compare(a.Timestamp)
	})

	for _, f := range files {
		// If this is the first file or it's in the same period as the previous file
		if len(currentGroup) == 0 ||
			(now.Sub(f.Timestamp)/period) == (now.Sub(currentGroup[0].Timestamp)/period) {
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
