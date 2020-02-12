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

func startDb() *store.DbOption {
	name := "nida-db"
	option := &store.DbOption{
		Host:     "localhost",
		Port:     8432,
		User:     "user",
		Password: "password",
		DbName:   "db",
	}

	if logs, err := container.Stop(&container.StopOptions{Name: name, Port: option.Port}); err != nil {
		log.Fatalln(err)
	} else {
		log.Println(fmt.Sprintf("Stopped container: %s", strings.TrimSpace(logs)))
	}

	if logs, err := container.Run(&container.RunOptions{
		Image: "postgres",
		Tag:   "12-alpine",
		Name:  name,
		Env: map[string]string{
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "db",
		},
		Ports: map[int]int{option.Port: 5432},
		Volumes: map[string]string{
			name: "/var/lib/postgresql/data",
		},
		Daemon: true,
	}); err != nil {
		log.Fatalln(err)
	} else {
		log.Println(fmt.Sprintf("Started postgres database container: %s", strings.TrimSpace(logs)))
	}

	return option
}
