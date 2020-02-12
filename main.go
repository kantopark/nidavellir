package main

import "C"
import (
	"log"

	"nidavellir/application"
	"nidavellir/config"
	"nidavellir/server"
	"nidavellir/services/scheduler"
	"nidavellir/services/store"
)

func main() {
	conf, err := config.New()
	if err != nil {
		log.Fatalln(err)
	}

	systemCheck()
	dbOption := startDb()

	db, err := store.New(dbOption)
	if err != nil {
		log.Fatalln(err)
	}

	manager := scheduler.NewJobManager(db)
	srv, err := server.New(conf.App.Port)
	if err != nil {
		log.Fatalln(err)
	}

	app, err := application.New(srv, db, manager, conf)
	if err != nil {
		log.Fatalln(err)
	}
	app.Run()
}
