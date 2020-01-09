package store

import (
	"fmt"
	"os"

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

	username := os.Getenv("GITHUB_USERNAME")
	publicRepoReadonlyToken := os.Getenv("GITHUB_TOKEN")
	repoPath := "kantopark/nidavellir/services/store/migration"
	sourceUrl := fmt.Sprintf("github://%s:%s@%s", username, publicRepoReadonlyToken, repoPath)

	m, err := migrate.NewWithDatabaseInstance(sourceUrl, "postgres", driver)
	if err != nil {
		return errors.Wrap(err, "could not create migration instance. "+
			"If error is due to rate limit by github, set your username and token in the environment with "+
			"'GITHUB_USERNAME' and 'GITHUB_TOKEN' respectively")
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, "could not apply migrations")
	}
	return nil
}
