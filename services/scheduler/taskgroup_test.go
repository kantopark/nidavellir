package scheduler_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	. "nidavellir/services/scheduler"
)

// Tests that environment variables read from repo into tasks group are read correctly
// Repo environment variable can be seen from the runtime config in
// https://github.com/kantopark/python-test-repo/blob/master/runtime.yaml
func TestTaskGroup_TaskEnvVarProcessedCorrectly(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	repo := pythonRepo
	ctx := context.Background()
	jobId := uniqueJobId()

	tg, err := NewTaskGroup(repo, ctx, 0, jobId, time.Now())
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
	t.Parallel()
	assert := require.New(t)
	repo := pythonRepo
	ctx := context.Background()
	jobId := uniqueJobId()

	tg, err := NewTaskGroup(repo, ctx, 0, jobId, time.Now())
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
