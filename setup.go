package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"

	"nidavellir/services/docker"
	container "nidavellir/services/docker/dkcontainer"
	"nidavellir/services/repo"
	"nidavellir/services/store"
)

func systemCheck() {
	var errs error
	if err := docker.SystemCheck(); err != nil {
		errs = multierror.Append(errs, err)
	}

	if err := repo.SystemCheck(); err != nil {
		errs = multierror.Append(errs, err)
	}

	if errs != nil {
		log.Fatalln(errs)
	}
}

func startDb() (option *store.DbOption, cleanUp func()) {
	name := "nida-db"
	port := 8432
	option = &store.DbOption{
		Host:     "localhost",
		Port:     port,
		User:     "user",
		Password: "password",
		DbName:   "db",
	}
	// cleans up DB by stopping it
	cleanUp = func() {
		logs, err := container.Stop(&container.StopOptions{Name: name, Port: port, IgnoreNotFoundError: true})
		if err != nil {
			log.Fatalln(err)
		} else {
			if len(logs) > 0 {
				log.Println(fmt.Sprintf("Stopped database container: %s", strings.TrimSpace(logs)))
			}
		}
	}

	cleanUp()
	if logs, err := container.Run(&container.RunOptions{
		Image: "postgres",
		Tag:   "12-alpine",
		Name:  name,
		Env: map[string]string{
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "db",
		},
		Ports: map[int]int{port: 5432},
		Volumes: map[string]string{
			name: "/var/lib/postgresql/data",
		},
		Daemon: true,
	}); err != nil {
		log.Fatalln(err)
	} else {
		log.Println(fmt.Sprintf("Started postgres database container: %s", strings.TrimSpace(logs)))
	}

	return option, cleanUp
}
