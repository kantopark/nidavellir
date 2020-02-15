package server

import (
	"io/ioutil"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"nidavellir/libs"
	"nidavellir/services/iofiles"
)

type IFileHandler interface {
	GetAll(sourceId, jobId int) (logs, imageLogs string, files []string, err error)
	GetImageLogs(sourceId, jobId int) (string, error)
	GetLogContent(sourceId, jobId int) (string, error)
	GetOutputFileList(sourceId, jobId int) ([]string, error)
}

func newFileHandler(appFolder string) (*FileHandler, error) {
	if !libs.PathExists(appFolder) {
		return nil, errors.New("server error. App folder does not exist!")
	}

	return &FileHandler{appFolder}, nil
}

type FileHandler struct {
	AppFolder string
}

func (f *FileHandler) GetAll(sourceId, jobId int) (string, string, []string, error) {
	var errs error
	logs, err := f.GetLogContent(sourceId, jobId)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	imageLogs, err := f.GetImageLogs(sourceId, jobId)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	files, err := f.GetOutputFileList(sourceId, jobId)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	if errs != nil {
		return "", "", nil, errs
	} else {
		return logs, imageLogs, files, nil
	}
}

func (f *FileHandler) GetImageLogs(sourceId, jobId int) (logs string, err error) {
	logs, err = f.readLog(sourceId, jobId, true)
	return
}

func (f *FileHandler) GetLogContent(sourceId, jobId int) (logs string, err error) {
	logs, err = f.readLog(sourceId, jobId, false)
	return
}

func (f *FileHandler) GetOutputFileList(sourceId, jobId int) (files []string, err error) {
	folder, err := iofiles.GetOutputDir(f.AppFolder, sourceId, jobId)
	if err != nil {
		return
	}

	fileInfos, err := ioutil.ReadDir(folder)
	if err != nil {
		return
	}

	for _, file := range fileInfos {
		if !file.IsDir() {
			files = append(files, file.Name())
		}
	}
	return
}

func (f *FileHandler) readLog(sourceId, jobId int, forImage bool) (logs string, err error) {
	var file *iofiles.LogFile

	if forImage {
		file, err = iofiles.NewImageLogFile(f.AppFolder, sourceId, jobId, true)
	} else {
		file, err = iofiles.NewLogFile(f.AppFolder, sourceId, jobId, true)
	}
	if err != nil {
		return
	}

	logs, err = file.Read()
	if err != nil {
		return "", err
	}

	return
}
