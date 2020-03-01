package store

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	"github.com/pkg/errors"

	"nidavellir/libs"
)

func (p *Postgres) Migrate() error {
	driver, err := postgres.WithInstance(p.db.DB(), &postgres.Config{})
	if err != nil {
		return errors.Wrap(err, "could not create database driver")
	}

	m, err := migrate.NewWithDatabaseInstance(migrationSource(), "postgres", driver)
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

func migrationSource() string {
	// returns a non-empty sourceUrl and no errors if path exists
	sourceUrlFromPath := func(elem ...string) string {
		dir := filepath.Join(elem...)
		if libs.PathExists(dir) {
			sourceUrl := "file://" + dir

			// Replace windows full path with current directory as go-migrate uses net/url to parse path and that
			// package doesn't parse windows path
			if runtime.GOOS == "windows" {
				cwd, err := os.Getwd()
				if err != nil {
					return ""
				}

				sourceUrl = strings.Replace(strings.Replace(sourceUrl, cwd, ".", 1), `\`, "/", -1)
			}
			return sourceUrl
		}
		return ""
	}

	_, file, _, _ := runtime.Caller(0)
	root, _ := os.Executable()
	for _, elems := range [][]string{
		{filepath.Dir(file), "migration"},
		{filepath.Dir(file), "migrations"},
		{filepath.Dir(root), "migration"},
		{filepath.Dir(root), "migrations"},
	} {
		if sourceUrl := sourceUrlFromPath(elems...); sourceUrl != "" {
			return sourceUrl
		}
	}

	// use gitlab url by default
	username := os.Getenv("GITHUB_USERNAME")
	publicRepoReadonlyToken := os.Getenv("GITHUB_TOKEN")
	repoPath := "kantopark/nidavellir/services/store/migration"
	return fmt.Sprintf("github://%s:%s@%s", username, publicRepoReadonlyToken, repoPath)
}
