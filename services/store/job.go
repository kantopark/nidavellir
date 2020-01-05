package store

import (
	"time"

	"github.com/pkg/errors"
)

const (
	JobQueued  = "QUEUED"
	JobRunning = "RUNNING"
	JobFailure = "FAILURE"
	JobSuccess = "SUCCESS"

	TriggerManual   = "MANUAL"
	TriggerSchedule = "SCHEDULE"
)

type Job struct {
	Id        int       `json:"id"`
	SourceId  int       `json:"source_id"`
	InitTime  time.Time `json:"init_time"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	State     string    `json:"state"`
	Trigger   string    `json:"trigger"`
}

func (j *Job) ToStartState() error {
	if j.State != JobQueued {
		return errors.Errorf("cannot reach '%s' state from '%s' state", JobRunning, j.State)
	}

	j.StartTime = time.Now().UTC()
	j.State = JobRunning

	return nil
}

func (j *Job) ToFailureState() error {
	if j.State != JobRunning {
		return errors.Errorf("cannot reach '%s' state from '%s' state", JobFailure, j.State)
	}

	j.EndTime = time.Now().UTC()
	j.State = JobFailure

	return nil
}

func (j *Job) ToSuccessState() error {
	if j.State != JobRunning {
		return errors.Errorf("cannot reach '%s' state from '%s' state", JobSuccess, j.State)
	}

	j.EndTime = time.Now().UTC()
	j.State = JobSuccess

	return nil
}

func (p *Postgres) AddJob(sourceId int, trigger string) (*Job, error) {
	if trigger != TriggerSchedule && trigger != TriggerManual {
		return nil, errors.Errorf("'%s' is not a valid trigger", trigger)
	}

	job := &Job{
		SourceId:  sourceId,
		InitTime:  time.Now().UTC(),
		StartTime: time.Time{},
		EndTime:   time.Time{},
		State:     JobQueued,
		Trigger:   trigger,
	}

	if err := p.db.Create(job).Error; err != nil {
		return nil, errors.Wrap(err, "could not create new job")
	}

	return job, nil
}

func (p *Postgres) UpdateJob(job *Job) (*Job, error) {
	if job.Id <= 0 {
		return nil, errors.New("job id must be specified")
	}

	err := p.db.
		Model(job).
		Where("id = ?", job.Id).
		Update(*job).
		Error
	if err != nil {
		return nil, errors.Wrap(err, "could not update job")
	}

	return job, nil
}

type ListJobOption struct {
	Trigger  string
	State    string
	SourceId int
}

func (p *Postgres) GetJobs(options *ListJobOption) ([]*Job, error) {
	var jobs []*Job

	if options == nil {
		options = &ListJobOption{}
	}

	query := p.db
	if options.State != "" {
		query = query.Where("state = ?", options.State)
	}
	if options.Trigger != "" {
		query = query.Where("trigger = ?", options.Trigger)
	}
	if options.SourceId != 0 {
		query = query.Where("source_id = ?", options.SourceId)
	}

	if err := query.Find(&jobs).Error; err != nil {
		return nil, errors.Wrap(err, "could not get jobs")
	}

	return jobs, nil
}
