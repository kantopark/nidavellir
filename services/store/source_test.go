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
		CommitTag string
		Interval  int
		HasError  bool
	}{
		{"Project", "https://git-repo", "", 30, false},
		{"123  ", "https://git-repo", "", 30, true},
		{"Project", "git-repo", "", 30, true},
		{"Project", "http://git-repo", strings.Repeat("a", 41), 30, true},
		{"Project", "https://git-repo", "random-tag", 29, true},
	}

	for _, test := range tests {
		s, err := NewSource(test.Name, test.RepoUrl, test.CommitTag, time.Now(), test.Interval)
		if test.HasError {
			assert.Error(err)
			assert.Nil(s)
		} else {
			assert.NoError(err)
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
			assert.IsType(Source{}, *s)
		}
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
				s.NextTime = time.Now().UTC().Add(-10 * time.Hour)
			} else {
				s.NextTime = time.Now().UTC().Add(10 * time.Hour)
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
			sources[i] = s
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

func newSources() ([]*Source, error) {
	var sources []*Source
	for _, i := range []struct {
		Name      string
		RepoUrl   string
		CommitTag string
		Interval  int
	}{
		{"Project 1", "https://git-repo", "", 30},
		{"Project 2", "https://git-repo", "0.0.1", 30},
	} {
		s, err := NewSource(i.Name, i.RepoUrl, i.CommitTag, time.Now(), i.Interval)
		if err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, nil
}