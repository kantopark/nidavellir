package store

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	"github.com/pkg/errors"
)

func (p *Postgres) Migrate() error {
	driver, err := postgres.WithInstance(p.db.DB(), &postgres.Config{})
	if err != nil {
		return errors.Wrap(err, "could not create database driver")
	}

	username := "danielbok"
	publicRepoReadonlyToken := ""
	repoPath := "kantopark/nidavellir/services/store/migration"
	sourceUrl := fmt.Sprintf("github://%s:%s@%s", username, publicRepoReadonlyToken, repoPath)

	m, err := migrate.NewWithDatabaseInstance(sourceUrl, "postgres", driver)
	if err != nil {
		return errors.Wrap(err, "could not create migration instance")
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, "could not apply migrations")
	}
	return nil
}
