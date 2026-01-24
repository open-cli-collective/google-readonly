package calendar

import (
	"fmt"
	"time"
)

// parseDate parses a date string in YYYY-MM-DD format
func parseDate(dateStr string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
	}
	return t, nil
}

// endOfDay returns the time at 23:59:59 on the given day
func endOfDay(t time.Time) time.Time {
	return t.Add(24*time.Hour - time.Second)
}

// weekBounds returns the start (Monday 00:00:00) and end (Sunday 23:59:59) of the week
// containing the given time.
func weekBounds(t time.Time) (start time.Time, end time.Time) {
	// Find Monday of this week
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday becomes 7
	}
	monday := t.AddDate(0, 0, -weekday+1)
	start = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, t.Location())

	// Find Sunday of this week
	sunday := start.AddDate(0, 0, 6)
	end = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, 0, t.Location())

	return start, end
}

// todayBounds returns the start (00:00:00) and end (23:59:59) of the given day
func todayBounds(t time.Time) (start time.Time, end time.Time) {
	start = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	end = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
	return start, end
}
