package main

import (
	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"

	"nidavellir/services/docker"
	"nidavellir/services/image"
	"nidavellir/services/repo"
)

func systemCheck() {
	var errs error
	if err := docker.SystemCheck(); err != nil {
		errs = multierror.Append(errs, err)
	}

	if err := repo.SystemCheck(); err != nil {
		errs = multierror.Append(errs, err)
	}

	if err := image.SystemCheck(); err != nil {
		errs = multierror.Append(errs, err)
	}

	if errs != nil {
		log.Fatalln(multierror.Flatten(errs))
	}
}

func startDb() {
	name := "nida"
	port := 5432
	c := docker.NewContainer()

	if logs, err := c.Stop(&docker.ContainerStopOptions{Name: name, Port: port}); err != nil {
		log.Fatalln(err)
	} else {
		log.Println(logs)
	}

	if logs, err := c.Run(&docker.ContainerRunOptions{
		Image: "postgres",
		Tag:   "12-alpine",
		Name:  name,
		Env: map[string]string{
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       name,
		},
		Ports: map[int]int{port: port},
		Volumes: map[string]string{
			name: "/var/lib/postgresql/data",
		},
		Daemon: true,
	}); err != nil {
		log.Fatalln(err)
	} else {
		log.Println(logs)
	}
}
