package iofiles

import (
	"os"
	"path/filepath"
	"strconv"

	"nidavellir/libs"
)

// Creates the folder to store the output from the tasks
func GetOutputDir(dataFolder string, jobId int) (string, error) {
	folder := filepath.Join(dataFolder, "output", strconv.Itoa(jobId))
	if !libs.PathExists(folder) {
		err := os.MkdirAll(folder, 0777)
		if err != nil {
			return folder, err
		}
	}
	return folder, nil
}
