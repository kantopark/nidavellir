package store_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	. "nidavellir/services/store"
)

func TestNewSource(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	tests := []struct {
		Name     string
		RepoUrl  string
		Secrets  []Secret
		CronExpr string
		Error    string
	}{
		{"Project", "https://git-repo", nil, "0 0 0 * * * *", ""},
		{"Project", "https://git-repo", []Secret{{Key: "Key", Value: "Value"}}, "0 0 0 * * * *", ""},
		{"Project", "https://git-repo", []Secret{{Key: "Key", Value: "Value"}}, "0 * 0 * * * *", "Interval between jobs too short (Every Minute)"},
		{"Project", "https://git-repo", []Secret{{Key: "Key", Value: "Value"}}, "0 0/2 0 * * * *", "Interval between jobs too short (Every 2 Minute)"},
		{"123  ", "https://git-repo", nil, "0 0 0 * * * *", "name length must be >= 4 characters"},
		{"Project", "git-repo", nil, "0 0 0 * * * *", "invalid repo url"},
		{"Project", "https://git-repo", nil, "bad cron expression", "invalid cron expression url"},
	}

	for _, test := range tests {
		s, err := NewSource(test.Name, test.RepoUrl, time.Now(), test.Secrets, test.CronExpr)
		if test.Error != "" {
			assert.Error(err, test.Error)
			assert.Nil(s)
		} else {
			assert.NoError(err, test.Error)
			assert.IsType(Source{}, *s)
		}
	}
}

func TestNewSource_NextTime(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	now := time.Now()

	// daily cron expression
	s, err := NewSource("Project", "https://git-repo", now, nil, "0 0 0 * * * *")
	assert.NoError(err)
	s.ToCompleted()

	nextTime := now.AddDate(0, 0, 1)
	nextTime = time.Date(nextTime.Year(), nextTime.Month(), nextTime.Day(), 0, 0, 0, 0, nextTime.Location())
	assert.Equal(nextTime, s.NextTime)
}

func TestPostgres_AddSource(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	sources, err := newSources()
	assert.NoError(err)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info)
		assert.NoError(err)

		for _, s := range sources {
			s, err := db.AddSource(s)
			assert.NoError(err)
			assert.IsType(&Source{}, s)
		}
	})
}

func TestPostgres_GetSource(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources)
		assert.NoError(err)

		source, err := db.GetSource(1)
		assert.NoError(err)
		assert.IsType(&Source{}, source)

		source, err = db.GetSource(0)
		assert.Error(err)
		assert.Nil(source)
	})
}

func TestPostgres_GetSources(t *testing.T) {
	t.Parallel()

	assert := require.New(t)
	sources, err := newSources()
	assert.NoError(err)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info)
		assert.NoError(err)

		for i, s := range sources {
			// Creating these differences to test list with scheduled option
			if i == 0 {
				s.NextTime = time.Now().Add(-10 * time.Hour)
			} else {
				s.NextTime = time.Now().Add(10 * time.Hour)
			}
			_, err := db.AddSource(s)
			assert.NoError(err)
		}

		list, err := db.GetSources(nil)
		assert.NoError(err)
		assert.Len(list, len(sources))
		assert.IsType(&Source{}, sources[0])

		list, err = db.GetSources(&GetSourceOption{ScheduledToRun: true})
		assert.NoError(err)
		assert.NotEmpty(list) // in the setup, there is more than 1 task where schedules is specified
	})
}

func TestPostgres_GetSources_MaskedSecret(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources, seedSecrets)
		assert.NoError(err)

		for _, mask := range []bool{true, false} {
			sources, err := db.GetSources(&GetSourceOption{MaskSecrets: mask})
			assert.NoError(err)
			assert.Len(sources, len(sources))

			for _, source := range sources[1:] {
				for _, secret := range source.Secrets {
					if mask {
						assert.EqualValues(strings.Repeat("*", len(secret.Value)), secret.Value)
					} else {
						assert.NotEqual(strings.Repeat("*", len(secret.Value)), secret.Value)
					}
				}
			}
		}
	})
}

func TestPostgres_RemoveSource(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	sources, err := newSources()
	assert.NoError(err)

	type testRow struct {
		Id       int
		HasError bool
	}

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info)
		assert.NoError(err)

		tests := []testRow{
			{0, true},
		}

		for _, s := range sources {
			s, err := db.AddSource(s)
			assert.NoError(err)
			// no errors when removing first time, but empty second time so error
			tests = append(tests, testRow{s.Id, false}, testRow{s.Id, true})
		}

		for _, test := range tests {
			err = db.RemoveSource(test.Id)
			if test.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		}

		// double check nothing left
		list, err := db.GetSources(nil)
		assert.NoError(err)
		assert.Len(list, 0)
	})
}

func TestPostgres_UpdateSource(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources)
		assert.NoError(err)

		s, err := db.GetSource(1)
		assert.NoError(err)
		numSec := len(s.Secrets)

		name := fmt.Sprintf("New-Project-Name-%d", s.Id)
		s.Name = name
		s.Secrets = append(s.Secrets, Secret{
			SourceId: s.Id,
			Key:      "NewKey",
			Value:    "NewValue",
		})
		s, err = db.UpdateSource(s)
		assert.NoError(err)
		assert.Len(s.Secrets, numSec+1)
		assert.EqualValues(s.Name, name)

		// test changes in the source or secret directly
		s.Secrets[0].Value = "1234"
		s.Secrets = append(s.Secrets, Secret{
			Key:   "NewKey2",
			Value: "13:30",
		})
		s, err = db.UpdateSource(s)
		assert.NoError(err)
		assert.Equal(s.Secrets[0].Value, "1234")
		assert.Len(s.Secrets, numSec+2)
	})
}

func newSources() ([]*Source, error) {
	var sources []*Source

	for _, i := range []struct {
		Name     string
		RepoUrl  string
		Secrets  []Secret
		CronExpr string
	}{
		{"Project 1", "https://git-repo", nil, "0 0 0 * * * *"},
		{"Project 2", "https://git-repo", []Secret{{Key: "Key", Value: "Value"}}, "0 0 0 * * * *"},
	} {
		s, err := NewSource(i.Name, i.RepoUrl, time.Now(), i.Secrets, i.CronExpr)
		if err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, nil
}
