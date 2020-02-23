package store

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"nidavellir/libs"
)

const (
	ScheduleQueued  = "QUEUED"
	ScheduleRunning = "RUNNING"
	ScheduleNoop    = "NOOP"
	SEP             = "::"
)

var weekDayMap = map[string]int{"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6}

type Source struct {
	Id         int       `json:"id"`
	Name       string    `json:"name"`
	UniqueName string    `json:"-"`
	RepoUrl    string    `json:"repoUrl"`
	Days       string    `json:"days"`
	Times      string    `json:"times"`
	State      string    `json:"state"`
	NextTime   time.Time `json:"nextTime"`
	Secrets    []Secret  `json:"secrets"`
}

func NewSource(name, repoUrl string, startTime time.Time, days []string, times []string) (s *Source, err error) {
	name = strings.TrimSpace(name)

	// time check is done here instead as it is used to set the value later
	reg := regexp.MustCompile(`\d{1,2}:\d{1,2}`)
	for i, t := range times {
		if !reg.MatchString(t) {
			return nil, errors.New("time must be in format hh:mm")
		}

		times[i], err = validateTime(t)
		if err != nil {
			return nil, err
		}
	}

	s = &Source{
		Name:       name,
		UniqueName: libs.LowerTrimReplaceSpace(name),
		RepoUrl:    repoUrl,
		Days:       libs.UpperTrim(strings.Join(days, SEP)),
		Times:      strings.Join(times, SEP),
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

	days, times := s.DaysAndTime()
	if len(days) == 0 || len(times) == 0 {
		return errors.New("days or times schedule cannot be empty")
	}

	for _, day := range days {
		if _, exists := weekDayMap[day]; !exists {
			var validDays []string
			for k := range weekDayMap {
				validDays = append(days, k)
			}
			return errors.Errorf("'%s' is not a supported day. use one of %v", day, validDays)
		}
	}

	if len(times) != len(days) {
		return errors.Errorf("days and times array must match!")
	}

	return nil
}

func validateTime(t string) (string, error) {
	timeParts := strings.Split(strings.TrimSpace(t), ":")
	h, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return "", errors.Wrapf(err, "invalid time '%s'", t)
	} else if h < 0 || h >= 24 {
		return "", errors.Errorf("invalid time '%s'", t)
	}
	m, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return "", errors.Wrapf(err, "invalid time '%s'", t)
	} else if m < 0 || m >= 60 {
		return "", errors.Errorf("invalid time '%s'", t)
	}
	return fmt.Sprintf("%02d:%02d", h, m), nil
}

// sets the source state to Running
func (s *Source) ToRunning() *Source {
	s.State = ScheduleRunning
	return s
}

// Sets the job's state to completed and calculates the next runtime
func (s *Source) ToCompleted() *Source {
	s.NextTime = s.DeriveNextTime()
	s.State = ScheduleNoop
	return s
}

// Derives the next run time for the Source
func (s *Source) DeriveNextTime() (nextTime time.Time) {
	now := time.Now().UTC()
	dow := now.Weekday() // day of week

	// loop through all the date and times in schedule
	// form time that fits the datetime schdule but greater than now
	// take the earliest time
	days, times := s.DaysAndTime()
	for i, d := range days {
		timeParts := strings.Split(times[i], ":")
		hour, _ := strconv.Atoi(timeParts[0])
		minute, _ := strconv.Atoi(timeParts[1])

		// the following calculates the next run date for the schedule
		dayDiff := (weekDayMap[d] - int(dow) + 7) % 7
		t := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC).AddDate(0, 0, dayDiff)
		if t.Before(now) {
			// in the event that this schedule is on the same day but the scheduled time is currently past now, we
			// schedule it for next week
			t = t.AddDate(0, 0, 7)
		}

		if nextTime.IsZero() || t.Before(nextTime) {
			nextTime = t
		}
	}

	return
}

func (s *Source) DaysAndTime() ([]string, []string) {
	days := strings.Split(s.Days, SEP)
	times := strings.Split(s.Times, SEP)

	return days, times
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

type GetSourceOption struct {
	ScheduledToRun bool
	MaskSecrets    bool
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
