package scheduler_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/dhui/dktest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"nidavellir/libs"
	. "nidavellir/services/scheduler"
	"nidavellir/services/store"
)

type mockStore struct {
	sources map[int]*store.Source
	jobs    map[int]*store.Job
}

func newMockStore() *mockStore {
	now := time.Now()

	return &mockStore{
		sources: map[int]*store.Source{
			1: {
				Id:         1,
				Name:       pythonRepo.Name,
				UniqueName: libs.LowerTrimReplaceSpace(pythonRepo.Name),
				RepoUrl:    pythonRepo.Source,
				State:      store.ScheduleNoop,
				NextTime:   time.Now(),
				CronExpr:   fmt.Sprintf("0 %d %d * * * *", now.Minute(), now.Hour()),
				Secrets: []store.Secret{
					{Key: "POSTGRES_USER", Value: user},
					{Key: "POSTGRES_PASSWORD", Value: password},
					{Key: "POSTGRES_DB", Value: dbName},
				},
			},
		},
		jobs: make(map[int]*store.Job),
	}
}

func (m mockStore) GetSources(_ *store.GetSourceOption) ([]*store.Source, error) {
	var sources []*store.Source
	for _, source := range m.sources {
		sources = append(sources, source)
	}

	return sources, nil
}

func (m mockStore) GetSource(id int) (*store.Source, error) {
	return m.sources[id], nil
}

func (m mockStore) UpdateSource(source *store.Source) (*store.Source, error) {
	m.sources[source.Id] = source
	return source, nil
}

func (m mockStore) AddJob(sourceId int, trigger string) (*store.Job, error) {
	id := len(m.jobs) + 1
	job := &store.Job{
		Id:        id,
		SourceId:  sourceId,
		InitTime:  time.Now(),
		StartTime: time.Time{},
		EndTime:   time.Time{},
		State:     store.JobQueued,
		Trigger:   trigger,
	}
	m.jobs[id] = job

	return job, nil
}

func (m mockStore) GetJob(id int) (*store.Job, error) {
	if job, exists := m.jobs[id]; !exists {
		return nil, errors.New("job does not exist")
	} else {
		return job, nil
	}
}

func (m mockStore) UpdateJob(job *store.Job) (*store.Job, error) {
	m.jobs[job.Id] = job
	return job, nil
}

func TestNewJobManager(t *testing.T) {
	assert := require.New(t)
	db := newMockStore()
	manager, err := NewJobManager(db, context.Background(), appDir)
	assert.NoError(err)

	manager.Start()

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		_, port, err := info.FirstPort()
		assert.NoError(err)
		source, _ := db.GetSource(1)
		source.Secrets = append(source.Secrets,
			store.Secret{Key: "POSTGRES_HOST", Value: "172.17.0.1"},
			store.Secret{Key: "POSTGRES_PORT", Value: port},
		)
		_, _ = db.UpdateSource(source)

		err = manager.AddJob(source, store.TriggerSchedule)
		assert.NoError(err)

		timeout := time.After(3 * time.Minute)

	poll:
		for {
			select {
			case <-time.After(5 * time.Second):
				// job succeeded
				if len(manager.CompletedJobs) > 0 {
					break poll
				}
			case <-timeout:
				assert.FailNow("manager did not have a completed job after waiting for 3 minutes")
			}
		}

		manager.Close()
	})
}

// this test case is used for debugging. Useful for checking folder structures generated by the manager
func TestNewJobManager_NoTimeOut(t *testing.T) {
	assert := require.New(t)
	db := newMockStore()
	manager, err := NewJobManager(db, context.Background(), appDir)
	assert.NoError(err)

	manager.Start()

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		_, port, err := info.FirstPort()
		assert.NoError(err)
		source, _ := db.GetSource(1)
		source.Secrets = append(source.Secrets,
			store.Secret{Key: "POSTGRES_HOST", Value: "172.17.0.1"},
			store.Secret{Key: "POSTGRES_PORT", Value: port},
		)
		_, _ = db.UpdateSource(source)

		err = manager.AddJob(source, store.TriggerSchedule)
		assert.NoError(err)

	loop:
		for {
			select {
			case <-time.After(5 * time.Second):
				// job succeeded
				if len(manager.CompletedJobs) > 0 {
					// set break point at this line.
					manager.Close()
					break loop
				}
			}
		}

		logFilePath := filepath.Join(manager.AppFolderPath, "jobs", strconv.Itoa(source.Id), "1", "logs.txt")
		content, err := ioutil.ReadFile(logFilePath)
		assert.NoError(err)
		assert.NotEmpty(string(content))
	})
}
