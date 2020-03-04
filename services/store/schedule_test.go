package store_test

import (
	"testing"
	"time"

	"github.com/dhui/dktest"
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

func TestPostgres_AddSchedule(t *testing.T) {
	t.Parallel()

	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		_, err := newTestDb(info, seedSources, seedSchedules)
		assert.NoError(err)
	})
}

func TestPostgres_RemoveSchedule(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources, seedSchedules)
		assert.NoError(err)

		// there should be a source with id 2 and with more than 0 schedules
		source, err := db.GetSource(2)
		assert.NoError(err)

		n := len(source.Schedules)
		assert.True(n > 0)

		// Remove schedule
		err = db.RemoveSchedule(source.Schedules[0].Id)
		assert.NoError(err)

		// fetch source again and check that it's schedule len dropped
		source, err = db.GetSource(2)
		assert.NoError(err)
		assert.Len(source.Schedules, n-1)
	})
}

func TestPostgres_UpdateSchedule(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources, seedSchedules)
		assert.NoError(err)

		source, err := db.GetSource(2)
		assert.NoError(err)
		assert.True(len(source.Schedules) > 0)

		schedule := &Schedule{
			Id:       source.Schedules[0].Id,
			SourceId: source.Schedules[0].SourceId,
			Day:      Weekday,
			Time:     "09:30",
		}

		output, err := db.UpdateSchedule(schedule)
		assert.NoError(err)
		assert.Equal(output, schedule)
	})
}

func seedSchedules(db *Postgres) error {
	sources, err := db.GetSources(nil)
	if err != nil {
		return err
	}

	// Every source will get (id - 1) * 2 schedules for Everyday. The time will be the (current time + index * 5 minute)
	for _, source := range sources {
		if source.Id == 1 {
			continue
		}

		for j := 0; j < (source.Id-1)*2; j++ {
			t := time.Now().Add(time.Duration(5*j) * time.Minute).Format("15:04")
			schedule, err := NewSchedule(source.Id, Everyday, t)
			if err != nil {
				return err
			}
			if _, err := db.AddSchedule(schedule); err != nil {
				return err
			}
		}
	}

	return nil
}
