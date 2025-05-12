package retention

import (
	"reflect"
	"testing"
	"time"

	"github.com/TotallyNotRobots/apply-retention-policy/internal/config"
	"github.com/TotallyNotRobots/apply-retention-policy/internal/file"
	"github.com/TotallyNotRobots/apply-retention-policy/pkg/logger"
	"go.uber.org/zap/zaptest"
)

func TestPolicy_Apply(t *testing.T) {
	type fields struct {
		config *config.Config
	}

	type args struct {
		files []file.FileInfo
		now   time.Time
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []file.FileInfo
		wantErr bool
	}{
		{
			"empty files",
			fields{&config.Config{
				Retention: config.RetentionPolicy{
					Hourly:  2,
					Daily:   3,
					Weekly:  6,
					Monthly: 5,
					Yearly:  4,
				},
				FilePattern: "",
				Directory:   "",
			}},
			args{[]file.FileInfo{}, time.Date(2025, 5, 5, 15, 43, 23, 0, time.UTC)},
			nil,
			false,
		},
		{
			"no prune - less than",
			fields{&config.Config{
				Retention: config.RetentionPolicy{
					Hourly:  2,
					Daily:   3,
					Weekly:  6,
					Monthly: 5,
					Yearly:  4,
				},
			}},
			args{
				[]file.FileInfo{
					// Hourly
					{
						Timestamp: time.Date(2025, 5, 5, 15, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 5, 5, 14, 43, 0, 0, time.UTC),
					},
					// Daily
					{
						Timestamp: time.Date(2025, 5, 5, 13, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 5, 4, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 5, 3, 13, 43, 0, 0, time.UTC),
					},
					// Weekly
					{
						Timestamp: time.Date(2025, 5, 2, 12, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 25, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 18, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 11, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 4, 14, 43, 0, 0, time.UTC),
					},
					// Monthly
					{
						Timestamp: time.Date(2025, 2, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 1, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2024, 12, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2024, 11, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2024, 10, 5, 14, 43, 0, 0, time.UTC),
					},
					// Yearly
					{
						Timestamp: time.Date(2024, 9, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2023, 12, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2022, 12, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2021, 12, 5, 14, 43, 0, 0, time.UTC),
					},
				},
				time.Date(2025, 5, 5, 15, 43, 23, 0, time.UTC),
			},
			nil,
			false,
		},
		{
			"no prune - exact",
			fields{&config.Config{
				Retention: config.RetentionPolicy{
					Hourly:  2,
					Daily:   3,
					Weekly:  6,
					Monthly: 5,
					Yearly:  4,
				},
			}},
			args{
				[]file.FileInfo{
					// Hourly
					{
						Timestamp: time.Date(2025, 5, 5, 15, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 5, 5, 14, 43, 0, 0, time.UTC),
					},
					// Daily
					{
						Timestamp: time.Date(2025, 5, 5, 13, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 5, 4, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 5, 3, 13, 43, 0, 0, time.UTC),
					},
					// Weekly
					{
						Timestamp: time.Date(2025, 5, 2, 12, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 25, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 18, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 11, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 4, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 3, 29, 14, 43, 0, 0, time.UTC),
					},
					// Monthly
					{
						Timestamp: time.Date(2025, 2, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 1, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2024, 12, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2024, 11, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2024, 10, 5, 14, 43, 0, 0, time.UTC),
					},
					// Yearly
					{
						Timestamp: time.Date(2024, 9, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2023, 12, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2022, 12, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2021, 12, 5, 14, 43, 0, 0, time.UTC),
					},
				},
				time.Date(2025, 5, 5, 15, 43, 23, 0, time.UTC),
			},
			nil,
			false,
		},
		{
			"prune 1 hourly",
			fields{&config.Config{
				Retention: config.RetentionPolicy{
					Hourly:  2,
					Daily:   3,
					Weekly:  6,
					Monthly: 5,
					Yearly:  4,
				},
			}},
			args{
				[]file.FileInfo{
					// Hourly
					{
						Timestamp: time.Date(2025, 5, 5, 15, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 5, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 5, 5, 13, 43, 0, 0, time.UTC),
					},
					// Daily
					{
						Timestamp: time.Date(2025, 5, 5, 10, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 5, 4, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 5, 3, 13, 43, 0, 0, time.UTC),
					},
					// Weekly
					{
						Timestamp: time.Date(2025, 5, 2, 12, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 25, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 18, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 11, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 4, 4, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 3, 29, 14, 43, 0, 0, time.UTC),
					},
					// Monthly
					{
						Timestamp: time.Date(2025, 2, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2025, 1, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2024, 12, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2024, 11, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2024, 10, 5, 14, 43, 0, 0, time.UTC),
					},
					// Yearly
					{
						Timestamp: time.Date(2024, 9, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2023, 12, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2022, 12, 5, 14, 43, 0, 0, time.UTC),
					},
					{
						Timestamp: time.Date(2021, 12, 5, 14, 43, 0, 0, time.UTC),
					},
				},
				time.Date(2025, 5, 5, 15, 43, 23, 0, time.UTC),
			},
			[]file.FileInfo{
				{
					Timestamp: time.Date(2025, 5, 5, 10, 43, 0, 0, time.UTC),
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := zaptest.NewLogger(t)
			p := &Policy{
				logger: &logger.Logger{Logger: tl},
				config: tt.fields.config,
			}

			got, err := p.Apply(tt.args.files, tt.args.now)
			if (err != nil) != tt.wantErr {
				t.Errorf("Policy.Apply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Policy.Apply() = %v, want %v", got, tt.want)
			}
		})
	}
}
