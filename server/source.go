package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"nidavellir/services/store"
)

type ISourceStore interface {
	AddSource(source store.Source) (*store.Source, error)
	GetSource(id int) (*store.Source, error)
	GetSources(options *store.GetSourceOption) ([]*store.Source, error)
	UpdateSource(source store.Source) (*store.Source, error)
	RemoveSource(id int) error
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
			http.Error(w, errors.Wrap(err, "invalid id provided").Error(), 400)
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

		source, err = s.DB.AddSource(*source)
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

		source, err = s.DB.UpdateSource(*source)
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
			http.Error(w, errors.Wrap(err, "invalid id provided").Error(), 400)
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
