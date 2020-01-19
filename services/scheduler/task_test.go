package scheduler_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	"nidavellir/config"
	"nidavellir/libs"
	. "nidavellir/services/scheduler"
)

var (
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

func TestTask_Execute(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	conf, err := config.New()
	assert.NoError(err)

	// clone a test repo and get path
	repo, err := newPythonRepo()
	assert.NoError(err)

	jobId := uniqueJobId()
	outputDir := conf.WorkDir.OutputDir(jobId)
	fileName := "cars.csv"

	task, err := NewTask(
		"TestTask_Execute",
		repo.Image,
		"test-nida-python-execute",
		"extract_a.py",
		outputDir,
		repo.WorkDir,
		map[string]string{
			"file_name": fileName,
		},
	)
	assert.NoError(err)

	logs, err := task.Execute()
	assert.NoError(err)
	assert.NotEmpty(logs)

	assert.True(libs.PathExists(filepath.Join(outputDir, fileName)))
}
