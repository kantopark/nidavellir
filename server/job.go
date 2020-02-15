package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"nidavellir/libs"
	"nidavellir/services/store"
)

type IJobStore interface {
	GetJob(id int) (*store.Job, error)
	GetJobs(options *store.ListJobOption) ([]*store.Job, error)
}

type JobInfo struct {
	*store.Job
	Logs        string   `json:"logs"`
	ImageLogs   string   `json:"image_logs"`
	OutputFiles []string `json:"output_files"`
}

type JobHandler struct {
	DB    IJobStore
	Files IFileHandler
}

func (j *JobHandler) GetJobs() http.HandlerFunc {
	checkInvalidStates := func(states []string) string {
		var invalidStates []string
		for _, state := range states {
			if libs.IsIn(state, []string{store.JobRunning, store.JobQueued, store.JobFailure, store.JobSuccess}) {
				invalidStates = append(invalidStates, state)
			}
		}
		if len(invalidStates) == 0 {
			return ""
		}
		return fmt.Sprintf("Invalid states: %s", strings.Join(invalidStates, ", "))
	}

	getStates := func(r *http.Request) []string {
		states := strings.Split(r.URL.Query().Get("state"), ",")
		if len(states) == 0 {
			states = append(states, store.JobQueued, store.JobRunning)
		}
		return states
	}

	return func(w http.ResponseWriter, r *http.Request) {
		states := getStates(r)
		if msg := checkInvalidStates(states); msg != "" {
			http.Error(w, msg, 400)
			return
		}

		jobs, err := j.DB.GetJobs(&store.ListJobOption{
			State: states,
		})
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		toJson(w, jobs)
	}
}

func (j *JobHandler) GetJobInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, errors.Wrapf(err, "invalid job id '%d'", id).Error(), 400)
			return
		}

		job, err := j.DB.GetJob(id)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		logs, imageLogs, outputFiles, err := j.Files.GetAll(job.SourceId, job.Id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		info := &JobInfo{
			Job:         job,
			Logs:        logs,
			ImageLogs:   imageLogs,
			OutputFiles: outputFiles,
		}

		toJson(w, info)
	}
}
