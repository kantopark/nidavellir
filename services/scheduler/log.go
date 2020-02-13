package scheduler

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"nidavellir/libs"
)

func NewLogFile(dataFolder, logType, imageNameOrTaskName, imageTagOrTaskDate string) (*LogFile, error) {
	if !libs.IsIn(strings.ToLower(logType), []string{"logs", "image-logs"}) {
		return nil, errors.Errorf("unsupported log type: %s", logType)
	}

	folderPath := filepath.Join(dataFolder, logType, imageNameOrTaskName)
	if !libs.PathExists(folderPath) {
		err := os.MkdirAll(folderPath, 0777)
		if err != nil {
			return nil, errors.Wrap(err, "could not create folder path to store logs")
		}
	}

	path := filepath.Join(folderPath, imageTagOrTaskDate)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &LogFile{file: file}, nil
}

type LogFile struct {
	file *os.File
}

func (l *LogFile) AppendContent(content interface{}) {
	mw := io.MultiWriter(os.Stdout, l.file)
	logger := log.New()
	logger.SetOutput(mw)
	logger.Println(content)
}

func (l *LogFile) Close() {
	_ = l.file.Close()
}
