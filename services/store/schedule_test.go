package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	. "nidavellir/services/store"
)

func TestNewSchedule(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	tests := []struct {
		Day   string
		Time  string
		Error string
	}{
		{"SUNDAY", "09:30", ""},
		{"SUNDAY", "9:3", ""},
		{"MONDAY", "-1:30", "Invalid time"},
		{"MONDAY", "24:30", "Invalid time"},
		{"MONDAY", "09:-1", "Invalid time"},
		{"MONDAY", "09:60", "Invalid time"},
		{"MONDAY", "ab:60", "Invalid time"},
		{"MONDAY", "09:cd", "Invalid time"},
		{"INVALID", "09:30", "Invalid day"},
	}

	for _, test := range tests {
		s, err := NewSchedule(1, test.Day, test.Time)

		if test.Error != "" {
			assert.Error(err, test.Error)
		} else {
			assert.NoError(err)
			assert.IsType(&Schedule{}, s)
		}
	}
}

func TestSchedule_NextTime(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	makeDate := func(day, hour, min int) time.Time {
		return time.Date(2020, 3, day, hour, min, 0, 0, time.Local)
	}
	// Tuesday 3 March 2020, 09:00
	now := makeDate(3, 9, 0)

	tests := []struct {
		Offset   int // days to offset, in case we need to make it test for Friday or Saturday edge cases
		Day      string
		Time     string
		NextTime time.Time
	}{
		{0, Wednesday, "09:00", now.AddDate(0, 0, 1)},
		{0, Tuesday, "08:00", now.AddDate(0, 0, 7).Add(-1 * time.Hour)},
		{0, Tuesday, "09:00", now.AddDate(0, 0, 7)},
		{0, Tuesday, "10:00", now.Add(1 * time.Hour)},
		{0, Weekday, "09:00", now.AddDate(0, 0, 1)},
		{0, Weekday, "08:00", now.AddDate(0, 0, 1).Add(-1 * time.Hour)},
		{0, Weekday, "10:00", now.Add(1 * time.Hour)},
		{3, Weekday, "09:00", now.AddDate(0, 0, 6)},
		{4, Weekday, "09:00", now.AddDate(0, 0, 6)},
		{5, Weekday, "09:00", now.AddDate(0, 0, 6)},
		{0, Everyday, "09:00", now.AddDate(0, 0, 1)},
	}

	for _, test := range tests {
		s, err := NewSchedule(1, test.Day, test.Time)
		assert.NoError(err)

		nextTime := s.NextTime(now.AddDate(0, 0, test.Offset))
		assert.Truef(nextTime.Equal(test.NextTime), "Expected: %v, Actual: %v", test.NextTime, nextTime)
	}
}
