package store

import (
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"nidavellir/libs"
)

const (
	ScheduleQueued  = "QUEUED"
	ScheduleRunning = "RUNNING"
	ScheduleNoop    = "NOOP"
)

type Source struct {
	Id        int       `json:"id"`
	Name      string    `json:"name"`
	RepoUrl   string    `json:"repo_url"`
	CommitTag string    `json:"commit_tag"`
	NextTime  time.Time `json:"next_time"`
	Interval  int       `json:"interval"`
	State     string    `json:"state"`
}

func NewSource(name, repoUrl, commitTag string, startTime time.Time, interval int) (*Source, error) {
	s := &Source{
		Name:      name,
		RepoUrl:   repoUrl,
		CommitTag: commitTag,
		NextTime:  startTime,
		Interval:  interval,
		State:     ScheduleNoop,
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Source) Validate() error {
	s.Name = strings.TrimSpace(s.Name)
	if len(s.Name) < 4 {
		return errors.New("name length must be >= 4 characters")
	}

	if matched := regexp.MustCompile(`^https?://\S+$`).MatchString(s.RepoUrl); !matched {
		return errors.Errorf("expected '%s' git remote to be accessible through http", s.RepoUrl)
	}

	if !libs.IsIn(s.State, []string{ScheduleNoop, ScheduleRunning, ScheduleQueued}) {
		return errors.Errorf("'%s' is an invalid schedule state", s.State)
	}

	s.CommitTag = strings.TrimSpace(s.CommitTag)
	if len(s.CommitTag) > 40 {
		return errors.New("commit tag must be <= 40 characters")
	}

	if s.Interval < 30 {
		return errors.Errorf("interval must be >= 30 (seconds)")
	}

	return nil
}

func (s *Source) NextRuntime() time.Time {
	lastRuntime := s.NextTime
	for lastRuntime.Before(time.Now().UTC()) {
		lastRuntime = lastRuntime.Add(time.Duration(s.Interval) * time.Second)
	}
	return lastRuntime
}

func (s *Source) ToRunState() error {
	if s.State != ScheduleQueued {
		return errors.Errorf("cannot reach '%s' state from '%s'", ScheduleRunning, s.State)
	}
	s.State = ScheduleRunning

	return nil
}

func (s *Source) ToQueueState() error {
	if s.State != ScheduleNoop {
		return errors.Errorf("cannot reach '%s' state from '%s'", ScheduleQueued, s.State)
	}
	s.State = ScheduleQueued

	return nil
}

func (s *Source) ToNoopState() error {
	if s.State != ScheduleQueued {
		return errors.Errorf("cannot reach '%s' state from '%s'", ScheduleRunning, s.State)
	}
	s.State = ScheduleRunning

	return nil
}

func (p *Postgres) AddSource(source *Source) (*Source, error) {
	source.Id = 0 // force primary key to be empty
	if err := source.Validate(); err != nil {
		return nil, err
	}

	if err := p.db.Create(source).Error; err != nil {
		return nil, errors.Wrap(err, "could not create new source")
	}

	return source, nil
}

func (p *Postgres) UpdateSource(source *Source) (*Source, error) {
	if err := source.Validate(); err != nil {
		return nil, err
	} else if source.Id <= 0 {
		return nil, errors.New("source id must be specified")
	}

	err := p.db.
		Model(source).
		Where("id = ?", source.Id).
		Update(*source).
		Error
	if err != nil {
		return nil, errors.Wrap(err, "could not update source")
	}

	return source, nil
}

func (p *Postgres) RemoveSource(id int) error {
	if id <= 0 {
		return errors.New("source id must be specified")
	}

	if err := p.db.First(&Source{}, id).Error; err != nil {
		return errors.Errorf("could not find any sources with id '%d'", id)
	}

	if err := p.db.Delete(&Source{Id: id}).Error; err != nil {
		return errors.Wrapf(err, "error removing source with id '%d'", id)
	}

	return nil
}

type GetSourceOption struct {
	ScheduledToRun bool
}

func (p *Postgres) GetSources(options *GetSourceOption) ([]*Source, error) {
	var sources []*Source
	if options == nil {
		options = &GetSourceOption{}
	}

	query := p.db
	if options.ScheduledToRun {
		query = query.Where("state = ? AND next_time <= ?", ScheduleNoop, time.Now().UTC())
	}

	if err := query.Find(&sources).Error; err != nil {
		return nil, errors.Wrap(err, "error getting sources")
	}

	return sources, nil
}
