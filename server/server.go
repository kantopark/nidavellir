package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/cors"
)

func New(port int) (*http.Server, error) {
	r := chi.NewRouter()
	attachMiddleware(r)
	attachRoutes(r)

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

func attachRoutes(r *chi.Mux) {
	r.Get("/healthcheck", HealthCheck)
}

func HealthCheck(w http.ResponseWriter, _ *http.Request) {
	ok(w)
}
