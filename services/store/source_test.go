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
		Name    string
		RepoUrl string
		Error   string
	}{
		{"Project", "https://git-repo", ""},
		{"123  ", "https://git-repo", "name length must be >= 4 characters"},
		{"Project", "git-repo", "invalid repo url"},
	}

	for _, test := range tests {
		s, err := NewSource(test.Name, test.RepoUrl, time.Now())
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
		assert.Len(list, 2)

		list, err = db.GetSources(&GetSourceOption{ScheduledToRun: true})
		assert.NoError(err)
		assert.Len(list, 1)
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
			assert.Len(sources, 2)

			for _, source := range sources[1:] {
				assert.NotZero(len(source.Secrets))

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
	sources, err := newSources()
	assert.NoError(err)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info)
		assert.NoError(err)

		for i, s := range sources {
			s, err := db.AddSource(s)
			assert.NoError(err)
			sources[i] = *s
		}

		prefix := "New-Project"
		for i, s := range sources {
			s.Name = fmt.Sprintf("%s-%d", prefix, i+2)
			_, err := db.UpdateSource(s)
			assert.NoError(err)
		}

		list, err := db.GetSources(nil)
		assert.NoError(err)
		for _, s := range list {
			assert.True(strings.HasPrefix(s.Name, prefix))
		}
	})
}

func newSources() ([]Source, error) {
	var sources []Source
	for _, i := range []struct {
		Name      string
		RepoUrl   string
		Schedules []Schedule
	}{
		{"Project 1", "https://git-repo", nil},
		{"Project 2", "https://git-repo", []Schedule{{Day: "Everyday", Time: "09:00"}}},
	} {
		s, err := NewSource(i.Name, i.RepoUrl, time.Now())
		if err != nil {
			return nil, err
		}

		s.Schedules = i.Schedules
		sources = append(sources, *s)
	}
	return sources, nil
}
