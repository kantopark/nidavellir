package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/kantopark/cronexpr"
	"github.com/pkg/errors"

	"nidavellir/services/store"
)

type ISourceStore interface {
	AddSource(source *store.Source) (*store.Source, error)
	GetSource(id int) (*store.Source, error)
	GetSources(options *store.GetSourceOption) ([]*store.Source, error)
	GetSourceByName(name string) (*store.Source, error)
	UpdateSource(source *store.Source) (*store.Source, error)
	RemoveSource(id int) error

	GetSecrets(sourceId int) ([]*store.Secret, error)
	UpdateSecret(secret *store.Secret) (*store.Secret, error)
	RemoveSecret(id int) error
	AddSecret(secret *store.Secret) (*store.Secret, error)
}

type SourceHandler struct {
	DB ISourceStore
}

func (s *SourceHandler) GetSources() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sources, err := s.DB.GetSources(&store.GetSourceOption{
			ScheduledToRun: false,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		toJson(w, sources)
	}
}

func (s *SourceHandler) GetSource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid source id").Error(), 400)
			return
		}

		source, err := s.DB.GetSource(id)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		toJson(w, source)
	}
}

func (s *SourceHandler) CreateSource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var source *store.Source
		err := readJson(r, &source)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		source, err = s.DB.AddSource(source)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		toJson(w, source)
	}
}

func (s *SourceHandler) UpdateSource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var source *store.Source
		err := readJson(r, &source)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// to prevent concurrent updates when there is a job that is still updating
		// as that may screw up secret injection
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ticker.C:
				curr, err := s.DB.GetSource(source.Id)
				if err != nil {
					http.Error(w, err.Error(), 400)
					return
				}
				if curr.State != store.ScheduleNoop {
					continue
				}

				source, err = s.DB.UpdateSource(source)
				if err != nil {
					http.Error(w, err.Error(), 400)
					return
				}

				toJson(w, source)
				return

			case <-time.After(1 * time.Minute):
				http.Error(w, "Source has a job that is still executing. "+
					"Please update it later when the job is done", 400)
				return
			}
		}
	}
}

func (s *SourceHandler) DeleteSource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid source id").Error(), 400)
			return
		}
		err = s.DB.RemoveSource(id)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		ok(w)
	}
}

func (s *SourceHandler) GetSecrets() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sourceId, err := strconv.Atoi(chi.URLParam(r, "sourceId"))
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid source id").Error(), 400)
			return
		}

		secrets, err := s.DB.GetSecrets(sourceId)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		toJson(w, secrets)
	}
}

func (s *SourceHandler) AddSecret() http.HandlerFunc {
	return s.createAddUpdateSecretHandler(true)
}

func (s *SourceHandler) UpdateSecret() http.HandlerFunc {
	return s.createAddUpdateSecretHandler(false)
}

func (s *SourceHandler) createAddUpdateSecretHandler(isCreate bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sourceId, err := strconv.Atoi(chi.URLParam(r, "sourceId"))
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid source id").Error(), 400)
			return
		}

		var secret *store.Secret
		err = readJson(r, &secret)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		secret.SourceId = sourceId

		if isCreate {
			secret, err = s.DB.AddSecret(secret)
		} else {
			secret, err = s.DB.UpdateSecret(secret)
		}

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		toJson(w, secret)
	}
}

func (s *SourceHandler) DeleteSecret() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := strconv.Atoi(chi.URLParam(r, "sourceId"))
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid source id").Error(), 400)
			return
		}

		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid secret id").Error(), 400)
			return
		}

		err = s.DB.RemoveSecret(id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		ok(w)
	}
}

func (s *SourceHandler) ValidateCron() http.HandlerFunc {
	type CronInput struct {
		Expression string `json:"expression"`
	}

	type NextRun struct {
		Time  time.Time `json:"time"`
		Delta float64   `json:"delta"`
	}

	type CronOutput struct {
		Errors   []string  `json:"errors"`
		NextRuns []NextRun `json:"nextRuns"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var input *CronInput
		err := readJson(r, &input)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		output := &CronOutput{Errors: []string{}}

		cron, err := cronexpr.Parse(input.Expression)
		if err != nil {
			output.Errors = append(output.Errors, errors.Wrap(err, "invalid cron expression").Error())
		} else {
			nextTimes := cron.NextN(time.Now(), 100)
			output.NextRuns = append(output.NextRuns, NextRun{Time: nextTimes[0], Delta: 0})

			for i, t := range nextTimes[1:] {
				delta := t.Sub(nextTimes[i]).Minutes()
				output.NextRuns = append(output.NextRuns, NextRun{Time: t, Delta: delta})

				if delta < 5 {
					output.Errors = append(output.Errors, "Any cron interval specified must be > 5 minutes")
					break
				}
			}
		}

		toJson(w, output)
	}
}

func (s *SourceHandler) ValidateSourceName() http.HandlerFunc {
	type Response struct {
		Exists bool `json:"exists"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		source, _ := s.DB.GetSourceByName(name)
		toJson(w, &Response{Exists: source != nil})
	}
}
