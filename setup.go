package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"nidavellir/config"
	"nidavellir/libs"
	"nidavellir/services/docker"
	container "nidavellir/services/docker/dkcontainer"
	"nidavellir/services/repo"
	"nidavellir/services/store"
)

type System struct {
	DatabaseName string
	DatabasePort int
	WorkDir      string
}

func NewSystem(conf *config.Config) *System {
	// Hard-Coded values as Nidavellir is envisioned to own the single machine
	return &System{
		DatabaseName: "nida-db",
		DatabasePort: 7432,
		WorkDir:      conf.App.WorkDir,
	}
}

func (s *System) SystemCheck() {
	var errs error

	// check that docker exists
	if err := docker.SystemCheck(); err != nil {
		errs = multierror.Append(errs, err)
	}

	// Check that the repo service (clone, pull, build image) has all the tools needed
	if err := repo.SystemCheck(); err != nil {
		errs = multierror.Append(errs, err)
	}

	// check that the database port is not taken
	if err := s.checkDbPort(); err != nil {
		// attempt to stop previous running instance db
		if err2 := s.stopDb(); err2 != nil {
			errs = multierror.Append(errs, err, err2)
		}
	}

	if errs != nil {
		log.Fatalln(errs)
	}
}

// Initialize the environment for the application. This comprises the following tasks:
// 1) starting the database.
func (s *System) Initialize() (option *store.DbOption) {
	log.Print("Initializing system setup")
	option = &store.DbOption{
		Host:     "localhost",
		Port:     s.DatabasePort,
		User:     "user",
		Password: "password",
		DbName:   "db",
	}

	if err := s.startDb(option); err != nil {
		log.Fatal(err)
	}

	if !libs.PathExists(s.WorkDir) {
		if err := os.MkdirAll(s.WorkDir, os.ModePerm); err != nil {
			log.Fatal(err)
		} else {
			log.Printf("Working Directory: %s", s.WorkDir)
		}
	}

	return option
}

func (s *System) startDb(option *store.DbOption) error {
	if logs, err := container.Run(&container.RunOptions{
		Image: "postgres",
		Tag:   "12-alpine",
		Name:  s.DatabaseName,
		Env: map[string]string{
			"POSTGRES_USER":     option.User,
			"POSTGRES_PASSWORD": option.Password,
			"POSTGRES_DB":       option.DbName,
		},
		Ports: map[int]int{s.DatabasePort: 5432},
		Volumes: map[string]string{
			s.DatabaseName: "/var/lib/postgresql/data",
		},
		Daemon: true,
	}); err != nil {
		return err
	} else {
		log.Println(fmt.Sprintf("Started postgres database container: %s", strings.TrimSpace(logs)))
	}
	return nil
}

// Cleans up any of the environment processes that were previously initialized by System
func (s *System) CleanUp() {
	log.Print("Cleaning up system setup services")
	if err := s.stopDb(); err != nil {
		log.Fatal(err)
	}
}

func (s *System) stopDb() error {
	logs, err := container.Stop(&container.StopOptions{Name: s.DatabaseName, Port: s.DatabasePort, IgnoreNotFoundError: true})
	if err != nil {
		return err
	} else if len(logs) > 0 {
		log.Println(fmt.Sprintf("Stopped database container: %s", strings.TrimSpace(logs)))
	}

	return nil
}

func (s *System) checkDbPort() error {
	server, err := net.Listen("tcp", fmt.Sprintf(":%d", s.DatabasePort))
	if err != nil {
		return errors.Errorf("port %d which is used for Nidavellir's database is already taken", s.DatabasePort)
	}
	defer func() { _ = server.Close() }()

	return nil
}
