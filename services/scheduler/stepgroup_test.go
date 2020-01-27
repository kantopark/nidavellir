package scheduler_test

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"

	. "nidavellir/services/scheduler"
)

func TestStepGroup_ExecuteTasksSequential(t *testing.T) {
	assert := require.New(t)
	repo := pythonRepo

	step := repo.Steps[0]
	assert.NotNil(step)

	var tasks []*Task
	jobId := uniqueJobId()

	for _, ti := range step.TaskInfoList {
		outputDir, err := outputDir(jobId)
		assert.NoError(err)

		task, err := NewTask(
			ti.Name,
			ti.Image,
			ti.Tag,
			ti.Cmd,
			outputDir,
			ti.WorkDir,
			ti.Env,
		)
		assert.NoError(err)
		tasks = append(tasks, task)
	}

	sg, err := NewStepGroup(step.Name, tasks)
	assert.NoError(err)

	// test both sequential and parallel runs, when i == 1, StepGroup runs sequentially
	for i := 1; i <= 2; i++ {
		ctx := context.Background()
		sem := semaphore.NewWeighted(int64(i))
		logs, err := sg.ExecuteTasks(ctx, sem)
		assert.NoError(err)
		assert.NotEmpty(logs)

		dir, err := outputDir(jobId)
		assert.NoError(err)
		files, err := ioutil.ReadDir(dir)
		assert.NoError(err)
		assert.Len(files, 2) // for this particular repo, should have 2 files
	}
}
