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

	"nidavellir/config"
	"nidavellir/services/iofiles"
)

var (
	appDir      string
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
	appConf config.AppConfig
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
	appDir = dir

	// creates a finite number of jobIds to mock jobs ids in database
	size := 1000
	jobIds = make(chan int, size)
	for i := 0; i < size; i++ {
		jobIds <- i
	}
	initRepos()

	githubToken := os.Getenv("GITHUB_TOKEN")
	provider := "github"
	if githubToken == "" {
		provider = ""
	}
	appConf = config.AppConfig{
		WorkDir: dir,
		PAT: config.PAT{
			Provider: provider,
			Token:    githubToken,
		},
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

// Gets a unique job id
func uniqueJobId() int {
	return <-jobIds
}

// Gets output directory for test job
func outputDir(sourceId, jobId int) (string, error) {
	return iofiles.GetOutputDir(appDir, sourceId, jobId)
}
