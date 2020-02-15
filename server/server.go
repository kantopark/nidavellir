package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	"github.com/rs/cors"

	"nidavellir/config"
)

type IStore interface {
	ISourceStore
	IJobStore
}

func New(port int, store IStore, conf *config.Config) (*http.Server, error) {
	r := chi.NewRouter()
	attachMiddleware(r)
	err := attachHandlers(r, store, conf)
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

func attachHandlers(r *chi.Mux, store IStore, conf *config.Config) error {
	fileHandler, err := newFileHandler(conf.App.WorkDir)
	if err != nil {
		return err
	}

	r.Get("/health-check", HealthCheck)

	r.Route("/api", func(r chi.Router) {
		r.Use(Authenticator)
		r.Route("/source", func(r chi.Router) {
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
			handler := JobHandler{DB: store, Files: fileHandler}
			r.Get("/", handler.GetJobs())
			r.Get("/{id}", handler.GetJobInfo())
		})
	})

	return nil
}

func Authenticator(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		username, password, ok := r.BasicAuth()
		if !ok {
			log.Println("Could not get authentication credentials")
		} else {
			// TODO add database call here
			context.WithValue(ctx, "username", username)
			context.WithValue(ctx, "password", password)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func HealthCheck(w http.ResponseWriter, _ *http.Request) {
	ok(w)
}
