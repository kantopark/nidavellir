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

var (
	Sunday    = "SUNDAY"
	Monday    = "MONDAY"
	Tuesday   = "TUESDAY"
	Wednesday = "WEDNESDAY"
	Thursday  = "THURSDAY"
	Friday    = "FRIDAY"
	Saturday  = "SATURDAY"
	Weekday   = "WEEKDAY"
	Everyday  = "EVERYDAY"

	timeCheckRegex = regexp.MustCompile(`\d{1,2}:\d{1,2}`)
	dayMap         = map[string]int{
		Sunday:    0,
		Monday:    1,
		Tuesday:   2,
		Wednesday: 3,
		Thursday:  4,
		Friday:    5,
		Saturday:  6,
		Weekday:   10,
		Everyday:  11,
	}
)

type Schedule struct {
	Id       int    `json:"id"`
	SourceId int    `json:"source_id"`
	Day      string `json:"day"`
	Time     string `json:"time"`
}

func NewSchedule(sourceId int, day, time string) (*Schedule, error) {
	s := &Schedule{
		SourceId: sourceId,
		Day:      day,
		Time:     time,
	}

	if err := s.Format(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Schedule) Format() error {
	// time check is done here instead as it is used to set the value later
	if !timeCheckRegex.MatchString(s.Time) {
		return errors.New("time must be in format hh:mm")
	}
	h, m, err := s.hourMinute()
	if err != nil {
		return err
	}

	s.Time = fmt.Sprintf("%02d:%02d", h, m)
	s.Day = libs.UpperTrim(s.Day)

	if _, exists := dayMap[s.Day]; !exists {
		var validDays []string
		for k := range dayMap {
			validDays = append(validDays, k)
		}
		return errors.Errorf("'%s' is not a supported day. use one of %v", s.Day, validDays)
	}

	return nil
}

// Gets the next runtime for the Schedule. If it is scheduled to run every Monday at 09:00
// and it is now, Tuesday 17:45, it will get the next Monday 09:30.
func (s *Schedule) NextTime(now time.Time) time.Time {
	hour, minute, _ := s.hourMinute() // should not have errors as it should have been checked previously
	nextTime := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	dowNow := now.Weekday() // day of week

	addIfBefore := func(days int) time.Time {
		if nextTime.After(now) {
			// no need to add date as the next run time is still in the future
			return nextTime
		}
		return nextTime.AddDate(0, 0, days)
	}

	switch s.Day {
	case Everyday:
		// add day if necessary
		return addIfBefore(1)
	case Weekday:
		switch dowNow {
		case 5:
			// Friday, next run date is Monday (add 3 days)
			return addIfBefore(3)
		case 6:
			// Saturday, force next run date is Monday  (add 2 days)
			return nextTime.AddDate(0, 0, 2)
		case 0:
			// Sunday, force next run date is Monday  (add 1 days)
			return nextTime.AddDate(0, 0, 1)
		default:
			// Monday to Thursday, add 1 day if necessary so its Monday to Friday
			return addIfBefore(1)
		}

	default: // Weekday specified exactly
		dowTarget := dayMap[s.Day]
		if dowTarget == int(dowNow) && nextTime.After(now) {
			// same day and next run time is in the future so no changes.
			// example, dow is Monday 1, target is Monday 1, now is 09:00, next run time is 10:00
			// so just return the next run time
			return nextTime
		}
		// need to add the date difference
		// the difference is mod ((target DOW - current DOW) + 7)
		// example, current DOW is Monday 1, target Thursday 4: (4 - 1 + 7) % 7 = add 3 days
		// example, current DOW is Thursday 4, target Tuesday 2: (2 - 4 + 7) % 7 = add 5 days
		// example, current DOW is Thursday 4, target Thursday 4: (4 - 4 + 7) % 7 + 7 days = add 7 days, edge case
		days := (dayMap[s.Day] - int(dowNow) + 7) % 7
		if days == 0 {
			days = 7 // edge case for same day
		}

		return nextTime.AddDate(0, 0, days)
	}
}

func (s *Schedule) hourMinute() (int, int, error) {
	timeParts := strings.Split(s.Time, ":")
	h, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return 0, 0, errors.Wrapf(err, "invalid time '%s'", s.Time)
	} else if h < 0 || h >= 24 {
		return 0, 0, errors.Errorf("invalid time '%s'", s.Time)
	}

	m, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return 0, 0, errors.Wrapf(err, "invalid time '%s'", s.Time)
	} else if m < 0 || m >= 60 {
		return 0, 0, errors.Errorf("invalid time '%s'", s.Time)
	}

	return h, m, nil
}
