package calendar

import (
	"testing"
	"time"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestParseDate(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result, err := parseDate(tt.input)

			if tt.wantErr {
				testutil.Error(t, err)
				testutil.Contains(t, err.Error(), "invalid date format")
			} else {
				testutil.NoError(t, err)
				testutil.Equal(t, result.Year(), tt.want.Year())
				testutil.Equal(t, result.Month(), tt.want.Month())
				testutil.Equal(t, result.Day(), tt.want.Day())
			}
		})
	}
}

func TestEndOfDay(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := endOfDay(tt.input)
			testutil.Equal(t, result, tt.want)
		})
	}
}

func TestWeekBounds(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			start, end := weekBounds(tt.input)

			testutil.Equal(t, start, tt.wantStart)
			testutil.Equal(t, end, tt.wantEnd)

			// Verify start is Monday
			testutil.Equal(t, start.Weekday(), time.Monday)

			// Verify end is Sunday
			testutil.Equal(t, end.Weekday(), time.Sunday)

			// Verify start is at 00:00:00
			testutil.Equal(t, start.Hour(), 0)
			testutil.Equal(t, start.Minute(), 0)
			testutil.Equal(t, start.Second(), 0)

			// Verify end is at 23:59:59
			testutil.Equal(t, end.Hour(), 23)
			testutil.Equal(t, end.Minute(), 59)
			testutil.Equal(t, end.Second(), 59)
		})
	}
}

func TestWeekBoundsSundayEdgeCase(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			start, end := weekBounds(sunday)

			// The Sunday should be included in the week
			testutil.Equal(t, end.Year(), sunday.Year())
			testutil.Equal(t, end.Month(), sunday.Month())
			testutil.Equal(t, end.Day(), sunday.Day())

			// The Monday should be 6 days before the Sunday
			expectedMonday := sunday.AddDate(0, 0, -6)
			testutil.Equal(t, start.Year(), expectedMonday.Year())
			testutil.Equal(t, start.Month(), expectedMonday.Month())
			testutil.Equal(t, start.Day(), expectedMonday.Day())
		})
	}
}

func TestTodayBounds(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			start, end := todayBounds(tt.input)

			testutil.Equal(t, start, tt.wantStart)
			testutil.Equal(t, end, tt.wantEnd)

			// Verify same day
			testutil.Equal(t, start.Year(), tt.input.Year())
			testutil.Equal(t, start.Month(), tt.input.Month())
			testutil.Equal(t, start.Day(), tt.input.Day())

			testutil.Equal(t, end.Year(), tt.input.Year())
			testutil.Equal(t, end.Month(), tt.input.Month())
			testutil.Equal(t, end.Day(), tt.input.Day())
		})
	}
}
