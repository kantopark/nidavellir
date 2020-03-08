package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	"github.com/rs/cors"

	"nidavellir/config"
	"nidavellir/server/authentication"
	"nidavellir/services/scheduler"
)

type IStore interface {
	ISourceStore
	IJobStore
	IAccountStore
}

func New(port int, store IStore, scheduler scheduler.IScheduler, conf *config.Config) (*http.Server, error) {
	r := chi.NewRouter()
	attachMiddleware(r)
	err := attachHandlers(r, store, scheduler, conf)
	if err != nil {
		return nil, errors.Wrap(err, "error attaching route handlers to http.Server")
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}, nil
}

func attachMiddleware(r *chi.Mux) {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler)
}

func attachHandlers(r *chi.Mux, store IStore, scheduler scheduler.IScheduler, conf *config.Config) error {
	fileHandler, err := newFileHandler(conf.App.WorkDir)
	if err != nil {
		return err
	}

	r.Get("/health-check", HealthCheck)

	// Private APIs
	r.Route("/api", func(r chi.Router) {
		// Non-public api protector, auth middleware will drop any unauthorized access
		r.Route("/source", func(r chi.Router) {
			r.Use(authentication.New(store, false, conf.Auth...))
			handler := SourceHandler{DB: store}

			r.Get("/", handler.GetSources())
			r.Get("/{id}", handler.GetSource())
			r.Post("/", handler.CreateSource())
			r.Put("/", handler.UpdateSource())
			r.Delete("/{id}", handler.DeleteSource())

			r.Get("/{sourceId}/secret", handler.GetSecrets())
			r.Post("/{sourceId}/secret", handler.AddSecret())
			r.Put("/{sourceId}/secret", handler.UpdateSecret())
			r.Delete("/{sourceId}/secret/{id}", handler.DeleteSecret())
		})

		r.Route("/job", func(r chi.Router) {
			r.Use(authentication.New(store, false, conf.Auth...))
			handler := JobHandler{DB: store, Files: fileHandler, Scheduler: scheduler}

			r.Get("/", handler.GetJobs())
			r.Get("/{id}", handler.GetJobInfo())
			r.Get("/trigger/{sourceId}", handler.InsertJob())
		})

		r.Route("/account", func(r chi.Router) {
			r.Use(authentication.New(store, false, config.BasicAuth))
			handler := AccountHandler{DB: store}

			r.Put("/", handler.UpdateAccount())
			r.Post("/", handler.AddAccount())
			r.Delete("/{id}", handler.RemoveAccount())
		})
	})

	// Public APIs
	r.Route("/public-api", func(r chi.Router) {
		r.Route("/account", func(r chi.Router) {
			handler := AccountHandler{DB: store}
			r.Post("/", handler.ValidateAccount())
		})
	})

	// TODO add file exposer

	return nil
}

func HealthCheck(w http.ResponseWriter, _ *http.Request) {
	ok(w)
}
