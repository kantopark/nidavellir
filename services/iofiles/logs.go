package iofiles

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"nidavellir/libs"
)

// Opens a new LogFile instance for general logging purposes.
func NewLogFile(appFolder string, sourceId, jobId int, readonly bool) (*LogFile, error) {
	folder, err := createFolder(appFolder, "jobs", strconv.Itoa(sourceId), strconv.Itoa(jobId))
	if err != nil {
		return nil, err
	}
	file, err := openFile(readonly, folder, "logs.txt")
	if err != nil {
		return nil, err
	}

	return &LogFile{file: file, readonly: readonly}, nil
}

// Opens a new LogFile instance for docker image build logging purposes.
func NewImageLogFile(appFolder string, sourceId, jobId int, readonly bool) (*LogFile, error) {
	folder, err := createFolder(appFolder, "jobs", strconv.Itoa(sourceId), strconv.Itoa(jobId))
	if err != nil {
		return nil, err
	}
	file, err := openFile(readonly, folder, "image.txt")
	if err != nil {
		return nil, err
	}

	return &LogFile{file: file, readonly: readonly}, nil
}

// LogFile helper instance for reading and writing log data
type LogFile struct {
	file     *os.File
	readonly bool
}

// Writes the error or logs into the log file and into the standard output
func (l *LogFile) Write(content interface{}) error {
	if l.readonly {
		return errors.New("cannot append content when file is readonly")
	}
	mw := io.MultiWriter(os.Stdout, l.file)
	logger := log.New()
	logger.SetOutput(mw)
	logger.Printf("%s", content)

	return nil
}

// Closes the LogFile, rendering it unusable anymore
func (l *LogFile) Close() {
	_ = l.file.Close()
}

// Reads all the file content
func (l *LogFile) Read() (string, error) {
	content, err := ioutil.ReadAll(l.file)
	if err != nil {
		return "", errors.Wrap(err, "could not read log file content")
	}
	return string(content), nil
}

// Creates the folder from the path element if it does not exist
func createFolder(elem ...string) (string, error) {
	folder := filepath.Join(elem...)
	if !libs.PathExists(folder) {
		err := os.MkdirAll(folder, 0777)
		if err != nil {
			return "", errors.Wrap(err, "could not create folder path to store logs")
		}
	}
	return folder, nil
}

func openFile(readonly bool, pathElem ...string) (*os.File, error) {
	path := filepath.Join(pathElem...)

	var flags int
	if readonly {
		flags = os.O_RDONLY
	} else {
		flags = os.O_CREATE | os.O_RDWR | os.O_APPEND
	}

	file, err := os.OpenFile(path, flags, 0666)
	if err != nil {
		return nil, err
	}
	return file, nil
}
