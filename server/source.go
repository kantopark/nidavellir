package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"nidavellir/services/store"
)

type ISourceStore interface {
	AddSource(source *store.Source) (*store.Source, error)
	GetSource(id int) (*store.Source, error)
	GetSources(options *store.GetSourceOption) ([]*store.Source, error)
	UpdateSource(source *store.Source) (*store.Source, error)
	RemoveSource(id int) error

	GetSecrets(sourceId int) ([]*store.Secret, error)
	UpdateSecret(secret *store.Secret) (*store.Secret, error)
	RemoveSecret(id int) error
	AddSecret(secret *store.Secret) (*store.Secret, error)

	AddSchedule(schedule *store.Schedule) (*store.Schedule, error)
	UpdateSchedule(schedule *store.Schedule) (*store.Schedule, error)
	RemoveSchedule(id int) error
}

type SourceHandler struct {
	DB ISourceStore
}

func (s *SourceHandler) GetSources() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sources, err := s.DB.GetSources(&store.GetSourceOption{
			ScheduledToRun: false,
			MaskSecrets:    true,
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
	return s.generateCreateUpdateSourceHandlerFunc(true)
}

func (s *SourceHandler) UpdateSource() http.HandlerFunc {
	return s.generateCreateUpdateSourceHandlerFunc(false)
}

func (s *SourceHandler) generateCreateUpdateSourceHandlerFunc(isCreate bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var source *store.Source
		err := readJson(r, &source)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if isCreate {
			source, err = s.DB.AddSource(source)
		} else {
			source, err = s.DB.UpdateSource(source)
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		toJson(w, source)
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

func (s *SourceHandler) AddSchedule() http.HandlerFunc {
	return s.createAddUpdateScheduleHandler(true)
}

func (s *SourceHandler) UpdateSchedule() http.HandlerFunc {
	return s.createAddUpdateScheduleHandler(false)
}

func (s *SourceHandler) createAddUpdateScheduleHandler(isCreate bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sourceId, err := strconv.Atoi(chi.URLParam(r, "sourceId"))
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid source id").Error(), 400)
			return
		}

		var schedule *store.Schedule
		err = readJson(r, &schedule)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		schedule.SourceId = sourceId

		if isCreate {
			schedule, err = s.DB.AddSchedule(schedule)
		} else {
			schedule, err = s.DB.UpdateSchedule(schedule)
		}

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		toJson(w, schedule)
	}
}

func (s *SourceHandler) DeleteSchedule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := strconv.Atoi(chi.URLParam(r, "sourceId"))
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid source id").Error(), 400)
			return
		}

		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid schedule id").Error(), 400)
			return
		}

		err = s.DB.RemoveSchedule(id)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		ok(w)
	}
}
