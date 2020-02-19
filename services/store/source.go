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
	Id         int    `json:"id"`
	Name       string `json:"name"`
	UniqueName string `json:"-"`
	RepoUrl    string `json:"repoUrl"`
	// time interval between job and next job in seconds
	Interval int       `json:"interval"`
	State    string    `json:"state"`
	NextTime time.Time `json:"nextTime"`
	Secrets  []Secret  `json:"secrets"`
}

func NewSource(name, repoUrl string, startTime time.Time, interval int) (*Source, error) {
	name = strings.TrimSpace(name)

	s := &Source{
		Name:       name,
		UniqueName: libs.LowerTrimReplaceSpace(name),
		RepoUrl:    repoUrl,
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

	if s.Interval < 30 {
		return errors.Errorf("interval must be >= 30 (seconds)")
	}

	return nil
}

// sets the source state to Running
func (s *Source) ToRunning() *Source {
	s.State = ScheduleRunning
	return s
}

// Sets the job's state to completed and calculates the next runtime
func (s *Source) ToCompleted() *Source {
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

// Gets the source with the specified id
func (p *Postgres) GetSource(id int) (*Source, error) {
	var source Source
	if err := p.db.First(&source, "id = ?", id).Error; err != nil {
		return nil, errors.Wrapf(err, "could not find source with id '%d'", id)
	}

	return &source, nil
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

	if options.MaskSecrets {
		for _, source := range sources {
			source.maskSecrets()
		}
	}

	return sources, nil
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
	MaskSecrets    bool
}

// Masks all secret values
func (s *Source) maskSecrets() {
	if len(s.Secrets) == 0 {
		return
	}

	var secrets []Secret
	for _, secret := range s.Secrets {
		secret.Value = strings.Repeat("*", len(secret.Value))
		secrets = append(secrets, secret)
	}

	s.Secrets = secrets
}

func (s *Source) SecretMap() map[string]string {
	secrets := make(map[string]string, len(s.Secrets))

	for _, secret := range s.Secrets {
		secrets[secret.Key] = secret.Value
	}

	return secrets
}
