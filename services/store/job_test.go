package store_test

import (
	"testing"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	. "nidavellir/services/store"
)

func TestPostgres_AddJobs(t *testing.T) {
	t.Parallel()

	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources, seedJobs)
		assert.NoError(err)

		_, err = db.AddJob(999, TriggerSchedule)
		assert.Error(err)

		_, err = db.AddJob(1, "BAD_TRIGGER")
		assert.Error(err)
	})
}

func TestPostgres_GetJob(t *testing.T) {
	t.Parallel()

	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources, seedJobs)
		assert.NoError(err)

		job, err := db.GetJob(1)
		assert.NoError(err)
		assert.IsType(&Job{}, job)

		job, err = db.GetJob(0)
		assert.Error(err)
		assert.Nil(job)
	})
}

func TestPostgres_GetJobs(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	sources, _ := newSources()
	numJobs := len(sources)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources, seedJobs)
		assert.NoError(err)

		jobs, err := db.GetJobs(nil)
		assert.NoError(err)
		assert.Len(jobs, numJobs)

		jobs, err = db.GetJobs(&ListJobOption{
			Trigger: TriggerSchedule,
		})
		assert.NoError(err)
		assert.NotEmpty(jobs)

		jobs, err = db.GetJobs(&ListJobOption{
			SourceId: 1,
		})
		assert.NoError(err)
		assert.Len(jobs, 1)

		jobs, err = db.GetJobs(&ListJobOption{
			State: []string{JobQueued},
		})
		assert.NoError(err)
		assert.Len(jobs, numJobs)
	})
}

func TestPostgres_UpdateJob(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	sources, _ := newSources()
	numJobs := len(sources)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources, seedJobs)
		assert.NoError(err)

		jobs, err := db.GetJobs(nil)
		assert.NoError(err)
		assert.Len(jobs, numJobs)

		job := jobs[0]
		err = job.ToStartState()
		assert.NoError(err)

		job, err = db.UpdateJob(job)
		assert.NoError(err)
		assert.EqualValues(job.State, JobRunning)

		err = job.ToSuccessState()
		assert.NoError(err)
		job, err = db.UpdateJob(job)
		assert.NoError(err)
		assert.EqualValues(job.State, JobSuccess)

		job = jobs[1]
		err = job.ToStartState()
		assert.NoError(err)

		job, err = db.UpdateJob(job)
		assert.NoError(err)
		assert.EqualValues(job.State, JobRunning)

		err = job.ToFailureState()
		assert.NoError(err)
		job, err = db.UpdateJob(job)
		assert.NoError(err)
		assert.EqualValues(job.State, JobFailure)
	})
}

func seedJobs(db *Postgres) error {
	sources, err := db.GetSources(nil)
	if err != nil {
		return err
	}

	for i, s := range sources {
		var trigger string
		if i%2 == 0 {
			trigger = TriggerManual
		} else {
			trigger = TriggerSchedule
		}

		_, err := db.AddJob(s.Id, trigger)
		if err != nil {
			return err
		}
	}

	return nil
}
