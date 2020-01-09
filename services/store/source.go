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
	Id         int       `json:"id"`
	Name       string    `json:"name"`
	UniqueName string    `json:"-"`
	RepoUrl    string    `json:"repo_url"`
	CommitTag  string    `json:"commit_tag"`
	Interval   int       `json:"interval"`
	State      string    `json:"state"`
	NextTime   time.Time `json:"next_time"`
	Secrets    []Secret  `json:"secrets"`
}

func NewSource(name, repoUrl, commitTag string, startTime time.Time, interval int) (*Source, error) {
	name = strings.TrimSpace(name)

	s := &Source{
		Name:       name,
		UniqueName: strings.ToLower(strings.Replace(name, " ", "-", -1)),
		RepoUrl:    repoUrl,
		CommitTag:  strings.TrimSpace(commitTag),
		Interval:   interval,
		State:      ScheduleNoop,
		NextTime:   startTime,
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

// sets the source state to Running
func (s *Source) Running() *Source {
	s.State = ScheduleRunning
	return s
}

// Sets the job's state to completed and calculates the next runtime
func (s *Source) Completed() *Source {
	s.NextTime = s.NextTime.Add(time.Duration(s.Interval) * time.Second)
	s.State = ScheduleNoop
	return s
}

// Adds a new job source
func (p *Postgres) AddSource(source Source) (*Source, error) {
	source.Id = 0 // force primary key to be empty
	if err := source.Validate(); err != nil {
		return nil, err
	}

	if err := p.db.Create(&source).Error; err != nil {
		return nil, errors.Wrap(err, "could not create new source")
	}

	return &source, nil
}

// Updates a job source
func (p *Postgres) UpdateSource(source Source) (*Source, error) {
	if err := source.Validate(); err != nil {
		return nil, err
	} else if source.Id <= 0 {
		return nil, errors.New("source id must be specified")
	}

	err := p.db.
		Model(&source).
		Where("id = ?", source.Id).
		Update(source).
		Error
	if err != nil {
		return nil, errors.Wrap(err, "could not update source")
	}

	return &source, nil
}

// Removes a job source
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

// Gets a list of jobs sources specified by the option. If nil, lists all job
// sources
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
