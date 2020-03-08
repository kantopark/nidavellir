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
	dbOption, cleanUp := startDb()
	defer cleanUp()

	db, err := store.New(dbOption)
	if err != nil {
		log.Fatalln(err)
	}

	sch, err := scheduler.NewScheduler(db, conf.App.WorkDir)
	if err != nil {
		log.Fatalln(err)
	}

	srv, err := server.New(conf.App.Port, db, sch, conf)
	if err != nil {
		log.Fatalln(err)
	}

	app, err := application.New(srv, db, sch, conf)
	if err != nil {
		log.Fatalln(err)
	}
	app.Run()
}
