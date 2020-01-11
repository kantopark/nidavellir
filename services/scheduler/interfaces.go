package scheduler

import "nidavellir/services/store"

type IStore interface {
	// Used to get all job sources. Set options to get all outdated sources
	// in implementation
	GetSources(options *store.GetSourceOption) ([]*store.Source, error)

	// Gets source with the specified Id
	GetSource(id int) (*store.Source, error)

	// Used to update source state
	UpdateSource(source store.Source) (*store.Source, error)

	// Adds a new job
	AddJob(sourceId int, trigger string) (*store.Job, error)

	// Gets a job by its id
	GetJob(id int) (*store.Job, error)

	// Updates the job state
	UpdateJob(job store.Job) (*store.Job, error)
}

type IManager interface {
	// Adds jobs to the overall list of to dos
	AddJob(job *store.Source, trigger string) error
}
