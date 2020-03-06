package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	. "nidavellir/server"
	"nidavellir/services/store"
)

func NewJobHandler() *JobHandler {
	jobMap := make(map[int]*store.Job)
	startTime := time.Date(2020, 1, 2, 9, 30, 0, 0, time.Local)
	states := []string{store.JobQueued, store.JobQueued, store.JobQueued, store.JobRunning, store.JobSuccess,
		store.JobFailure, store.JobQueued, store.JobRunning, store.JobSuccess}

	for i := 0; i < 20; i++ {
		jobMap[i+1] = &store.Job{
			Id:        i + 1,
			SourceId:  i%3 + 1,
			InitTime:  startTime,
			StartTime: startTime.Add(1 * time.Minute),
			EndTime:   startTime.Add(9 * time.Minute),
			State:     states[i%len(states)],
			Trigger:   store.TriggerSchedule,
		}
		startTime = startTime.Add(time.Duration((i+1)*10) * time.Minute)
	}

	return &JobHandler{
		DB:        &MockJobStore{db: jobMap},
		Files:     &MockFileHandler{},
		Scheduler: &MockJobScheduler{},
	}
}

func TestJobHandler_GetJobs(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewJobHandler()

	for _, test := range []struct {
		States []string
	}{
		{nil},
		{[]string{store.JobQueued}},
		{[]string{store.JobRunning}},
	} {
		w := httptest.NewRecorder()
		r := NewTestRequest("GET", "/", nil, nil)
		if len(test.States) > 0 {
			r.URL.Query().Add("states", strings.Join(test.States, ","))
		}

		handler.GetJobs()(w, r)
		assert.Equal(http.StatusOK, w.Code)

		var jobs []*store.Job
		err := readJson(w, &jobs)
		assert.NoError(err)
		assert.IsType([]*store.Job{}, jobs)
		assert.Condition(func() bool {
			return len(jobs) > 0
		})
	}
}

func TestJobHandler_GetJobInfo(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewJobHandler()

	for _, test := range []struct {
		Id         string
		StatusCode int
	}{
		{"0", http.StatusBadRequest},
		{"1", http.StatusOK},
	} {
		w := httptest.NewRecorder()
		r := NewTestRequest("GET", "/", nil, map[string]string{
			"id": test.Id,
		})

		handler.GetJobInfo()(w, r)
		assert.Equal(test.StatusCode, w.Code)

		if test.StatusCode == http.StatusOK {
			var info *JobInfo
			err := readJson(w, &info)
			assert.NoError(err)
			assert.IsType(&JobInfo{}, info)
		}
	}
}

func TestJobHandler_InsertJob(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewJobHandler()

	w := httptest.NewRecorder()
	r := NewTestRequest("POST", "/trigger", nil, map[string]string{"sourceId": "1"})

	handler.InsertJob()(w, r)
	assert.Equal(http.StatusOK, w.Code)
}
