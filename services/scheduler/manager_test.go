package scheduler_test

import (
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
	return &mockStore{
		sources: map[int]*store.Source{
			1: {
				Id:         1,
				Name:       pythonRepo.Name,
				UniqueName: libs.LowerTrimReplaceSpace(pythonRepo.Name),
				RepoUrl:    pythonRepo.Source,
				Interval:   3000,
				State:      store.ScheduleNoop,
				NextTime:   time.Now(),
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

func (m mockStore) UpdateSource(source store.Source) (*store.Source, error) {
	m.sources[source.Id] = &source
	return &source, nil
}

func (m mockStore) AddJob(sourceId int, trigger string) (*store.Job, error) {
	id := len(m.jobs) + 1
	job := &store.Job{
		Id:        id,
		SourceId:  sourceId,
		InitTime:  time.Now().UTC(),
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

func (m mockStore) UpdateJob(job store.Job) (*store.Job, error) {
	m.jobs[job.Id] = &job
	return &job, nil
}

func TestNewJobManager(t *testing.T) {
	assert := require.New(t)
	db := newMockStore()
	manager := NewJobManager(db)

	err := manager.Start()
	assert.NoError(err)

	err = manager.Start() // should have error when starting again
	assert.Error(err)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		_, port, err := info.FirstPort()
		assert.NoError(err)
		source, _ := db.GetSource(1)
		source.Secrets = append(source.Secrets,
			store.Secret{Key: "POSTGRES_HOST", Value: "172.17.0.1"},
			store.Secret{Key: "POSTGRES_PORT", Value: port},
		)
		_, _ = db.UpdateSource(*source)

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
				assert.FailNow("manager did not have a completed job after waiting for 5 minutes")
			}
		}

		manager.Close()
	})
}
