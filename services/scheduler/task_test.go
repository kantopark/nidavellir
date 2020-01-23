package scheduler_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"nidavellir/config"
	"nidavellir/libs"
	. "nidavellir/services/scheduler"
)

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
