package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/dhui/dktest"
	"github.com/pkg/errors"

	. "nidavellir/services/store"
)

var (
	user                 = "user"
	password             = "password"
	dbName               = "db"
	imageName            = "postgres:12-alpine"
	postgresImageOptions = dktest.Options{
		ReadyFunc:    dbReady,
		PortRequired: true,
		ReadyTimeout: 5 * time.Minute,
		Env: map[string]string{
			"POSTGRES_USER":     user,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       dbName,
		},
	}
)

func connectionString(c dktest.ContainerInfo) (string, error) {
	ip, port, err := c.FirstPort()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", ip, port, user, password, dbName), nil
}

func dbReady(ctx context.Context, c dktest.ContainerInfo) bool {
	cs, err := connectionString(c)
	if err != nil {
		return false
	}

	db, err := sql.Open("postgres", cs)
	if err != nil {
		return false
	}
	defer func() { _ = db.Close() }()

	return db.PingContext(ctx) == nil
}

func newTestDb(c dktest.ContainerInfo) (*Postgres, error) {
	ip, strPort, err := c.FirstPort()
	if err != nil {
		return nil, errors.Wrap(err, "could not obtain test postgres db network address")
	}

	port, _ := strconv.Atoi(strPort)
	store, err := New(&DbOption{
		Host:     ip,
		Port:     port,
		User:     user,
		Password: password,
		DbName:   dbName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not create test postgres db")
	}

	if err = store.Migrate(); err != nil {
		return nil, err
	}

	return store, nil
}
