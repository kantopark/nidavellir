package scheduler_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	"nidavellir/services/repo"
	. "nidavellir/services/scheduler"
)

// Tests that environment variables read from repo into tasks group are read correctly
// Repo environment variable can be seen from the runtime config in
// https://github.com/kantopark/python-test-repo/blob/master/runtime.yaml
func TestTaskGroup_TaskEnvVarProcessedCorrectly(t *testing.T) {
	assert := require.New(t)
	jobId := uniqueJobId()

	tg, err := NewTaskGroup(pythonRepo, context.Background(), 0, jobId, time.Now(), appDir)
	assert.NoError(err)
	assert.Len(tg.StepGroups, 3)

	assert.True(reflect.DeepEqual(tg.StepGroups[0].Tasks[0].Env, map[string]string{
		"key1":      "step key1",
		"key2":      "key2",
		"key3":      "key3",
		"key4":      "key4",
		"file_name": "cars.csv",
	}))

	assert.True(reflect.DeepEqual(tg.StepGroups[0].Tasks[1].Env, map[string]string{
		"key1": "function key1",
		"key2": "function key2",
		"key3": "key3",
		"key4": "key4",
	}))

	assert.True(reflect.DeepEqual(tg.StepGroups[1].Tasks[0].Env, map[string]string{
		"key1": "key1",
		"key2": "key2",
		"key3": "key3",
	}))

	assert.True(reflect.DeepEqual(tg.StepGroups[1].Tasks[0].Env, map[string]string{
		"key1": "key1",
		"key2": "key2",
		"key3": "key3",
	}))
}

func TestTaskGroup_AddEnvVar(t *testing.T) {
	assert := require.New(t)
	jobId := uniqueJobId()

	tg, err := NewTaskGroup(pythonRepo, context.Background(), 0, jobId, time.Now(), appDir)
	assert.NoError(err)
	assert.Len(tg.StepGroups, 3)

	key1 := "Priority Key1"
	tg.AddEnvVar(map[string]string{
		"key1":   key1,
		"secret": "value",
	})

	assert.True(reflect.DeepEqual(tg.StepGroups[0].Tasks[0].Env, map[string]string{
		"key1":      key1,
		"secret":    "value",
		"key2":      "key2",
		"key3":      "key3",
		"key4":      "key4",
		"file_name": "cars.csv",
	}))

	assert.True(reflect.DeepEqual(tg.StepGroups[0].Tasks[1].Env, map[string]string{
		"key1":   key1,
		"secret": "value",
		"key2":   "function key2",
		"key3":   "key3",
		"key4":   "key4",
	}))

	assert.True(reflect.DeepEqual(tg.StepGroups[1].Tasks[0].Env, map[string]string{
		"key1":   key1,
		"secret": "value",
		"key2":   "key2",
		"key3":   "key3",
	}))

	assert.True(reflect.DeepEqual(tg.StepGroups[1].Tasks[0].Env, map[string]string{
		"key1":   key1,
		"secret": "value",
		"key2":   "key2",
		"key3":   "key3",
	}))
}

func TestTaskGroup_Execute(t *testing.T) {
	assert := require.New(t)

	tg, err := newTaskGroup(pythonRepo)
	assert.NoError(err)
	assert.Len(tg.StepGroups, 3)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		_, port, err := info.FirstPort()
		assert.NoError(err)

		envs := make(map[string]string)
		for key, value := range postgresEnv {
			envs[key] = value
		}
		envs["POSTGRES_HOST"] = "172.17.0.1"
		envs["POSTGRES_PORT"] = port

		tg.AddEnvVar(envs).SetMaxDuration(5 * time.Minute)

		r, err := tg.Execute()
		assert.NoError(err)
		assert.NotEmpty(r.Logs)
		assert.True(r.Completed)
	})
}

func TestTaskGroup_LongRunningTasksCancelledCorrectly(t *testing.T) {
	assert := require.New(t)

	tests := []struct {
		Duration time.Duration
		HasError bool
	}{
		{2 * time.Second, true}, // max duration only 2 seconds, thus errors
		{10 * time.Second, false},
	}

	for _, test := range tests {
		tg, err := newTaskGroup(longOpsRepo)
		assert.NoError(err)

		tg.SetMaxDuration(test.Duration)
		r, err := tg.Execute()
		if test.HasError {
			assert.Error(err)
		} else {
			assert.NoError(err)
			assert.IsType(&ExecutionResult{}, r)
		}
	}
}

func TestTaskGroup_ExitsWithNonZeroFailureCodes(t *testing.T) {
	assert := require.New(t)

	tg, err := newTaskGroup(failureRepo)
	assert.NoError(err)

	r, err := tg.Execute()
	assert.Error(err)
	assert.IsType(&ExecutionResult{}, r)
}

func TestTaskGroup_BranchExitCode(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	for _, test := range []struct {
		Env      map[string]string
		NumSteps int
	}{
		{nil, 2},
		{map[string]string{"exit_code_1_2": "0"}, 3},
	} {
		tg, err := newTaskGroup(exitCodeRepo)
		assert.NoError(err)
		tg.AddEnvVar(test.Env)

		r, err := tg.Execute()
		assert.NoError(err)
		assert.IsType(&ExecutionResult{}, r)
		assert.Len(r.Steps, test.NumSteps)
	}
}

func newTaskGroup(rp *repo.Repo) (*TaskGroup, error) {
	return NewTaskGroup(rp, context.Background(), 0, uniqueJobId(), time.Now(), appDir)
}
