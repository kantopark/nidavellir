package scheduler_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"

	rp "nidavellir/services/repo"
	. "nidavellir/services/scheduler"
)

// Using the PythonRepo, tests that tasks are executed correctly
func TestStepGroup_ExecuteTasks(t *testing.T) {
	assert := require.New(t)

	jobId := uniqueJobId()
	sg, err := FormTestStepGroup(pythonRepo, jobId)
	assert.NoError(err)

	// test both sequential and parallel runs, when i == 1, StepGroup runs sequentially
	for i := 1; i <= 2; i++ {
		ctx := context.Background()
		sem := semaphore.NewWeighted(int64(i))
		logs, err := sg.ExecuteTasks(ctx, sem)
		assert.NoError(err)
		assert.NotEmpty(logs)

		dir, err := outputDir(1, jobId)
		assert.NoError(err)
		files, err := ioutil.ReadDir(dir)
		assert.NoError(err)
		assert.Len(files, 2) // for this particular repo, should have 2 files
	}
}

// Creates a mock StepGroup. This StepGroup will be formed with the first step from the
// repo's Steps
func FormTestStepGroup(repo *rp.Repo, jobId int) (*StepGroup, error) {
	var tasks []*Task

	for _, ti := range repo.Steps[0].TaskInfoList {
		outputDir, err := outputDir(1, jobId)
		if err != nil {
			return nil, err
		}

		task, err := NewTask(
			ti.Name,
			ti.Image,
			fmt.Sprintf("%s__%d", ti.Tag, jobId),
			ti.Cmd,
			outputDir,
			ti.WorkDir,
			ti.Env,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return NewStepGroup(repo.Steps[0].Name, tasks)
}
