package server_test

import (
	"github.com/pkg/errors"

	"nidavellir/services/store"
)

type MockJobStore struct {
	db map[int]*store.Job
}

func (m *MockJobStore) GetJob(id int) (*store.Job, error) {
	if job, exist := m.db[id]; !exist {
		return nil, errors.Errorf("no job with id '%d'", id)
	} else {
		return job, nil
	}
}

func (m *MockJobStore) GetJobs(_ *store.ListJobOption) ([]*store.Job, error) {
	var jobs []*store.Job
	for _, job := range m.db {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

type MockFileHandler struct{}

func (m *MockFileHandler) GetAll(sourceId, jobId int) (logs, imageLogs string, files []string, err error) {
	logs, _ = m.GetLogContent(sourceId, jobId)
	imageLogs, _ = m.GetImageLogs(sourceId, jobId)
	files, _ = m.GetOutputFileList(sourceId, jobId)
	return
}

func (m *MockFileHandler) GetImageLogs(_, _ int) (string, error) {
	return "Some Image logs", nil
}

func (m *MockFileHandler) GetLogContent(_, _ int) (string, error) {
	return "Some Job Logs", nil
}

func (m *MockFileHandler) GetOutputFileList(_, _ int) ([]string, error) {
	return []string{"file1", "file2"}, nil
}

type MockJobScheduler struct{}

func (m *MockJobScheduler) AddJob(sourceId int, _ string) error {
	if sourceId == 0 {
		return errors.New("mock error")
	}
	return nil
}

func (m *MockJobScheduler) Start() {
}

func (m *MockJobScheduler) Close() {
}
