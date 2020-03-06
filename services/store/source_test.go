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
		Name      string
		RepoUrl   string
		Secrets   []Secret
		Schedules []Schedule
		Error     string
	}{
		{"Project", "https://git-repo", nil, nil, ""},
		{"Project", "https://git-repo", []Secret{{
			Key:   "Key",
			Value: "Value",
		}}, []Schedule{{
			Day:  Everyday,
			Time: "09:30",
		}}, ""},
		{"123  ", "https://git-repo", nil, nil, "name length must be >= 4 characters"},
		{"Project", "git-repo", nil, nil, "invalid repo url"},
	}

	for _, test := range tests {
		s, err := NewSource(test.Name, test.RepoUrl, time.Now(), test.Secrets, test.Schedules)
		if test.Error != "" {
			assert.Error(err, test.Error)
			assert.Nil(s)
		} else {
			assert.NoError(err, test.Error)
			assert.IsType(Source{}, *s)
		}
	}
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
		nSch := len(s.Schedules)
		nSec := len(s.Secrets)

		name := fmt.Sprintf("New-Project-Name-%d", s.Id)
		s.Name = name
		s.Secrets = append(s.Secrets, Secret{
			SourceId: s.Id,
			Key:      "NewKey",
			Value:    "NewValue",
		})
		s.Schedules = append(s.Schedules, Schedule{
			SourceId: s.Id,
			Day:      Monday,
			Time:     "12:30",
		})

		s, err = db.UpdateSource(s)
		assert.NoError(err)
		assert.Len(s.Schedules, nSch+1)
		assert.Len(s.Secrets, nSec+1)
		assert.EqualValues(s.Name, name)

		// test changes in the source or secret directly
		s.Schedules[0].Time = "12:31"
		s.Schedules = append(s.Schedules, Schedule{
			Day:  Tuesday,
			Time: "13:30",
		})
		s, err = db.UpdateSource(s)
		assert.NoError(err)
		assert.Equal(s.Schedules[0].Time, "12:31")
		assert.Len(s.Schedules, nSch+2)
	})
}

func newSources() ([]*Source, error) {
	var sources []*Source

	t := time.Now().Add(-2 * time.Minute).Format("15:04")
	for _, i := range []struct {
		Name      string
		RepoUrl   string
		Secrets   []Secret
		Schedules []Schedule
	}{
		{"Project 1", "https://git-repo", nil, nil},
		{"Project 2", "https://git-repo", []Secret{{Key: "Key", Value: "Value"}}, nil},
		{"Project 3", "https://git-repo", nil, []Schedule{{Day: Everyday, Time: t}}},
		{"Project 4", "https://git-repo", []Secret{{Key: "Key", Value: "Value"}}, []Schedule{{Day: Everyday, Time: t}}},
	} {
		s, err := NewSource(i.Name, i.RepoUrl, time.Now(), i.Secrets, i.Schedules)
		if err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, nil
}
