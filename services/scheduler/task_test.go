package scheduler_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"nidavellir/libs"
	. "nidavellir/services/scheduler"
)

func TestNewTask(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	tests := []struct {
		Name      string
		Image     string
		Tag       string
		Cmd       string
		OutputDir string
		WorkDir   string
		Env       map[string]string
		HasError  bool
	}{
		{"Name", "Image", "Tag", "Cmd", "Output", "Work", map[string]string{"key": "value"}, false},
		{"Name", "Image", "Tag", "Cmd", "Output", "Work", nil, false},
		{"Name", "Image", "Tag", "Cmd", "Output", "", nil, false},
		{"Name", "Image", "Tag", "Cmd", "", "Work", nil, true},
		{"Name", "Image", "Tag", "", "Output", "Work", nil, true},
		{"Name", "Image", "", "Cmd", "Output", "Work", nil, true},
		{"Name", "", "Tag", "Cmd", "Output", "Work", nil, true},
		{"", "Image", "Tag", "Cmd", "Output", "Work", nil, true},
	}

	for _, t := range tests {
		_, err := NewTask(t.Name, t.Image, t.Tag, t.Cmd, t.OutputDir, t.WorkDir, t.Env)
		if t.HasError {
			assert.Error(err)
		} else {
			assert.NoError(err)
		}
	}
}

func TestTask_Execute(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	repo := pythonRepo

	jobId := uniqueJobId()
	outputDir, err := outputDir(jobId)
	assert.NoError(err)
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
