package calendar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    time.Time
	}{
		{
			name:    "valid date",
			input:   "2026-01-24",
			wantErr: false,
			want:    time.Date(2026, 1, 24, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "valid date leap year",
			input:   "2024-02-29",
			wantErr: false,
			want:    time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "valid date year start",
			input:   "2026-01-01",
			wantErr: false,
			want:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "valid date year end",
			input:   "2026-12-31",
			wantErr: false,
			want:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "invalid format - slash separator",
			input:   "2026/01/24",
			wantErr: true,
		},
		{
			name:    "invalid format - wrong order",
			input:   "24-01-2026",
			wantErr: true,
		},
		{
			name:    "invalid format - missing leading zero",
			input:   "2026-1-24",
			wantErr: true,
		},
		{
			name:    "invalid date - month 13",
			input:   "2026-13-01",
			wantErr: true,
		},
		{
			name:    "invalid date - day 32",
			input:   "2026-01-32",
			wantErr: true,
		},
		{
			name:    "invalid format - empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format - text",
			input:   "tomorrow",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDate(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid date format")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.Year(), result.Year())
				assert.Equal(t, tt.want.Month(), result.Month())
				assert.Equal(t, tt.want.Day(), result.Day())
			}
		})
	}
}

func TestEndOfDay(t *testing.T) {
	tests := []struct {
		name  string
		input time.Time
		want  time.Time
	}{
		{
			name:  "regular day",
			input: time.Date(2026, 1, 24, 0, 0, 0, 0, time.UTC),
			want:  time.Date(2026, 1, 24, 23, 59, 59, 0, time.UTC),
		},
		{
			name:  "day with existing time",
			input: time.Date(2026, 1, 24, 10, 30, 0, 0, time.UTC),
			want:  time.Date(2026, 1, 25, 10, 29, 59, 0, time.UTC),
		},
		{
			name:  "end of month",
			input: time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
			want:  time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:  "end of year",
			input: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			want:  time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := endOfDay(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestWeekBounds(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name      string
		input     time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "monday",
			input:     time.Date(2026, 1, 26, 10, 30, 0, 0, loc), // Monday
			wantStart: time.Date(2026, 1, 26, 0, 0, 0, 0, loc),   // Monday
			wantEnd:   time.Date(2026, 2, 1, 23, 59, 59, 0, loc), // Sunday
		},
		{
			name:      "tuesday",
			input:     time.Date(2026, 1, 27, 14, 0, 0, 0, loc),  // Tuesday
			wantStart: time.Date(2026, 1, 26, 0, 0, 0, 0, loc),   // Monday
			wantEnd:   time.Date(2026, 2, 1, 23, 59, 59, 0, loc), // Sunday
		},
		{
			name:      "wednesday",
			input:     time.Date(2026, 1, 28, 9, 0, 0, 0, loc),   // Wednesday
			wantStart: time.Date(2026, 1, 26, 0, 0, 0, 0, loc),   // Monday
			wantEnd:   time.Date(2026, 2, 1, 23, 59, 59, 0, loc), // Sunday
		},
		{
			name:      "thursday",
			input:     time.Date(2026, 1, 29, 12, 0, 0, 0, loc),  // Thursday
			wantStart: time.Date(2026, 1, 26, 0, 0, 0, 0, loc),   // Monday
			wantEnd:   time.Date(2026, 2, 1, 23, 59, 59, 0, loc), // Sunday
		},
		{
			name:      "friday",
			input:     time.Date(2026, 1, 30, 17, 0, 0, 0, loc),  // Friday
			wantStart: time.Date(2026, 1, 26, 0, 0, 0, 0, loc),   // Monday
			wantEnd:   time.Date(2026, 2, 1, 23, 59, 59, 0, loc), // Sunday
		},
		{
			name:      "saturday",
			input:     time.Date(2026, 1, 31, 10, 0, 0, 0, loc),  // Saturday
			wantStart: time.Date(2026, 1, 26, 0, 0, 0, 0, loc),   // Monday
			wantEnd:   time.Date(2026, 2, 1, 23, 59, 59, 0, loc), // Sunday
		},
		{
			name:      "sunday - edge case",
			input:     time.Date(2026, 2, 1, 20, 0, 0, 0, loc),   // Sunday
			wantStart: time.Date(2026, 1, 26, 0, 0, 0, 0, loc),   // Monday
			wantEnd:   time.Date(2026, 2, 1, 23, 59, 59, 0, loc), // Sunday
		},
		{
			name:      "week spanning month boundary",
			input:     time.Date(2026, 1, 28, 10, 0, 0, 0, loc),  // Wednesday Jan 28
			wantStart: time.Date(2026, 1, 26, 0, 0, 0, 0, loc),   // Monday Jan 26
			wantEnd:   time.Date(2026, 2, 1, 23, 59, 59, 0, loc), // Sunday Feb 1
		},
		{
			name:      "week spanning year boundary",
			input:     time.Date(2025, 12, 31, 10, 0, 0, 0, loc), // Wednesday Dec 31
			wantStart: time.Date(2025, 12, 29, 0, 0, 0, 0, loc),  // Monday Dec 29
			wantEnd:   time.Date(2026, 1, 4, 23, 59, 59, 0, loc), // Sunday Jan 4
		},
		{
			name:      "first monday of month",
			input:     time.Date(2026, 2, 2, 8, 0, 0, 0, loc),    // Monday Feb 2
			wantStart: time.Date(2026, 2, 2, 0, 0, 0, 0, loc),    // Monday Feb 2
			wantEnd:   time.Date(2026, 2, 8, 23, 59, 59, 0, loc), // Sunday Feb 8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := weekBounds(tt.input)

			assert.Equal(t, tt.wantStart, start, "start mismatch")
			assert.Equal(t, tt.wantEnd, end, "end mismatch")

			// Verify start is Monday
			assert.Equal(t, time.Monday, start.Weekday(), "start should be Monday")

			// Verify end is Sunday
			assert.Equal(t, time.Sunday, end.Weekday(), "end should be Sunday")

			// Verify start is at 00:00:00
			assert.Equal(t, 0, start.Hour())
			assert.Equal(t, 0, start.Minute())
			assert.Equal(t, 0, start.Second())

			// Verify end is at 23:59:59
			assert.Equal(t, 23, end.Hour())
			assert.Equal(t, 59, end.Minute())
			assert.Equal(t, 59, end.Second())
		})
	}
}

func TestWeekBoundsSundayEdgeCase(t *testing.T) {
	// Specific test for the Sunday edge case which requires special handling
	loc := time.UTC

	// Test multiple Sundays
	sundays := []time.Time{
		time.Date(2026, 2, 1, 10, 0, 0, 0, loc),  // Sunday Feb 1
		time.Date(2026, 2, 8, 10, 0, 0, 0, loc),  // Sunday Feb 8
		time.Date(2026, 2, 15, 10, 0, 0, 0, loc), // Sunday Feb 15
		time.Date(2026, 1, 4, 10, 0, 0, 0, loc),  // Sunday Jan 4
	}

	for _, sunday := range sundays {
		t.Run(sunday.Format("2006-01-02"), func(t *testing.T) {
			start, end := weekBounds(sunday)

			// The Sunday should be included in the week
			assert.Equal(t, sunday.Year(), end.Year())
			assert.Equal(t, sunday.Month(), end.Month())
			assert.Equal(t, sunday.Day(), end.Day())

			// The Monday should be 6 days before the Sunday
			expectedMonday := sunday.AddDate(0, 0, -6)
			assert.Equal(t, expectedMonday.Year(), start.Year())
			assert.Equal(t, expectedMonday.Month(), start.Month())
			assert.Equal(t, expectedMonday.Day(), start.Day())
		})
	}
}

func TestTodayBounds(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name      string
		input     time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "regular day morning",
			input:     time.Date(2026, 1, 24, 8, 30, 0, 0, loc),
			wantStart: time.Date(2026, 1, 24, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 1, 24, 23, 59, 59, 0, loc),
		},
		{
			name:      "regular day evening",
			input:     time.Date(2026, 1, 24, 20, 45, 30, 0, loc),
			wantStart: time.Date(2026, 1, 24, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 1, 24, 23, 59, 59, 0, loc),
		},
		{
			name:      "midnight",
			input:     time.Date(2026, 1, 24, 0, 0, 0, 0, loc),
			wantStart: time.Date(2026, 1, 24, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 1, 24, 23, 59, 59, 0, loc),
		},
		{
			name:      "end of day",
			input:     time.Date(2026, 1, 24, 23, 59, 59, 0, loc),
			wantStart: time.Date(2026, 1, 24, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 1, 24, 23, 59, 59, 0, loc),
		},
		{
			name:      "first day of year",
			input:     time.Date(2026, 1, 1, 12, 0, 0, 0, loc),
			wantStart: time.Date(2026, 1, 1, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 1, 1, 23, 59, 59, 0, loc),
		},
		{
			name:      "last day of year",
			input:     time.Date(2026, 12, 31, 12, 0, 0, 0, loc),
			wantStart: time.Date(2026, 12, 31, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 12, 31, 23, 59, 59, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := todayBounds(tt.input)

			assert.Equal(t, tt.wantStart, start)
			assert.Equal(t, tt.wantEnd, end)

			// Verify same day
			assert.Equal(t, tt.input.Year(), start.Year())
			assert.Equal(t, tt.input.Month(), start.Month())
			assert.Equal(t, tt.input.Day(), start.Day())

			assert.Equal(t, tt.input.Year(), end.Year())
			assert.Equal(t, tt.input.Month(), end.Month())
			assert.Equal(t, tt.input.Day(), end.Day())
		})
	}
}
