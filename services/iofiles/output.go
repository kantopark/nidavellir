package iofiles

import (
	"path/filepath"
	"strconv"
)

// Creates the folder to store the output from the tasks
func GetOutputDir(appFolder string, sourceId, jobId int) (string, error) {
	folder, err := createFolder(appFolder, "jobs", strconv.Itoa(sourceId), strconv.Itoa(jobId), "output")
	if err != nil {
		return "", err
	}
	return folder, nil
}

// Gets the meta file path
func GetMetaFilePath(appFolder string, sourceId, jobId int) string {
	return filepath.Join(appFolder, "jobs", strconv.Itoa(sourceId), strconv.Itoa(jobId), "meta.json")
}
