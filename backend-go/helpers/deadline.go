package helpers

import (
	"strings"
	"time"
)

// ParseDate - tương đương DeadlineHelper.ParseDate trong C#
func ParseDate(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}
	raw = strings.TrimSpace(raw)
	for _, format := range formats {
		if t, err := time.Parse(format, raw); err == nil {
			return &t
		}
	}
	return nil
}

// CalcDaysLeft - tương đương DeadlineHelper.CalcDaysLeft trong C#
func CalcDaysLeft(deadline *time.Time) *int {
	if deadline == nil {
		return nil
	}
	diff := int(time.Until(*deadline).Hours() / 24)
	return &diff
}
