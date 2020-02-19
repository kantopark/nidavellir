package application

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"nidavellir/config"
	"nidavellir/services/scheduler"
	"nidavellir/services/store"
)

type App struct {
	closeCh chan struct{}
	manager *scheduler.JobManager
	server  *http.Server
	conf    *config.Config
}

func New(server *http.Server, store *store.Postgres, manager *scheduler.JobManager, conf *config.Config) (*App, error) {
	setLogger()
	if err := store.Migrate(); err != nil {
		return nil, err
	}

	if err := createAdminAccount(store, conf); err != nil {
		return nil, err
	}

	return &App{
		closeCh: make(chan struct{}),
		manager: manager,
		server:  server,
		conf:    conf,
	}, nil
}

func (a *App) Run() {
	go a.shutdownListener()
	go a.manager.Start()
	a.runServer()
	<-a.closeCh
}

func (a *App) runServer() {
	if conf := a.conf.App; conf.HasCerts() {
		tls := conf.TLS
		log.Info("Running server in HTTPS mode")
		if err := a.server.ListenAndServeTLS(tls.CertFile, tls.KeyFile); err != http.ErrServerClosed {
			log.WithField("cause", err).Error("server error")
		}
	} else {
		log.Info("Running server in HTTP mode")
		if err := a.server.ListenAndServe(); err != http.ErrServerClosed {
			log.WithField("cause", err).Error("server error")
		}
	}
}

func (a *App) shutdownListener() {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	sig := <-sigint

	log.WithField("signal", sig.String()).Info("Shutting down server")

	if err := a.server.Shutdown(context.Background()); err != nil {
		log.WithField("cause", err).Error("error shutting down application server")
	}

	log.Info("Shutting down job manager")
	a.manager.Close()

	close(a.closeCh)
}

func setLogger() {
	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}
