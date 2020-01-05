package main

import (
	log "github.com/sirupsen/logrus"

	"nidavellir/services/docker"
)

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

func main() {
	startDb()
}
