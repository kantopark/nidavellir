package scheduler_test

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhui/dktest"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"nidavellir/config"
	"nidavellir/services/repo"
)

var (
	jobIds      chan int
	user        = "user"
	password    = "password"
	dbName      = "db"
	imageName   = "postgres:12-alpine"
	postgresEnv = map[string]string{
		"POSTGRES_USER":     user,
		"POSTGRES_PASSWORD": password,
		"POSTGRES_DB":       dbName,
	}
	postgresImageOptions = dktest.Options{
		ReadyFunc:    dbReady,
		PortRequired: true,
		ReadyTimeout: 5 * time.Minute,
		Env:          postgresEnv,
	}
)

func init() {
	fileInfo, err := ioutil.ReadDir(os.TempDir())
	if err != nil {
		log.Fatalln(err)
	}

	// Remove past nida folders
	for _, f := range fileInfo {
		if strings.HasPrefix(f.Name(), "nida-") {
			err := os.RemoveAll(filepath.Join(os.TempDir(), f.Name()))
			if err != nil {
				log.Println(errors.Wrap(err, "could not clear past test folders"))
			}
		}
	}

	// creates a temporary directory in the user's temp folder which would be used to store
	// the repositories, outputs, etc.
	dir, err := ioutil.TempDir("", "nida-")
	if err != nil {
		log.Fatalln(err)
	}
	viper.Set("app.workdir", dir)

	// creates a finite number of jobIds to mock jobs ids in database
	size := 1000
	jobIds = make(chan int, size)
	for i := 0; i < size; i++ {
		jobIds <- i
	}

	// init repos
	_, err = newPythonRepo()
	if err != nil {
		log.Fatalln(err)
	}
}

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

// Clones a python test repository which would be used by all tests. Each tests should not
// mutate the contents in this repository
func newPythonRepo() (*repo.Repo, error) {
	pythonSource := "https://github.com/kantopark/python-test-repo"
	rp, err := repo.NewRepo(pythonSource, "python-test-repo")
	if err != nil {
		return nil, err
	}

	if !rp.Exists() {
		err := rp.Clone()
		if err != nil {
			return nil, err
		}
	}

	if exists, err := rp.HasImage(); err != nil {
		return nil, err
	} else if !exists {
		_, err := rp.PullImage()
		if err != nil {
			return nil, err
		}
	}

	return rp, nil
}

// Gets a unique job id
func uniqueJobId() int {
	return <-jobIds
}

// Gets output directory for test job
func outputDir(jobId int) (string, error) {
	conf, err := config.New()
	if err != nil {
		return "", err
	}
	return conf.App.OutputDir(jobId), nil
}
