package scheduler_test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"nidavellir/services/repo"
)

var jobIds chan int

func init() {
	fileInfo, err := ioutil.ReadDir(os.TempDir())
	if err != nil {
		log.Fatalln(err)
	}

	for _, f := range fileInfo {
		if strings.HasPrefix(f.Name(), "nida-") {
			err := os.RemoveAll(filepath.Join(os.TempDir(), f.Name()))
			if err != nil {
				log.Println(errors.Wrap(err, "could not clear past test folders"))
			}
		}
	}

	dir, err := ioutil.TempDir("", "nida-")
	if err != nil {
		log.Fatalln(err)
	}

	jobIds = make(chan int, 100)
	for i := 0; i < 100; i++ {
		jobIds <- i
	}

	viper.Set("workdir.path", dir)
}

func newPythonRepo() (*repo.Repo, error) {
	pythonSource := "https://github.com/kantopark/python-test-repo"
	rp, err := repo.NewRepo(pythonSource, "python-test-repo")
	if err != nil {
		return nil, err
	}

	if !rp.Exists() {
		err := rp.Clone()
		if err != nil {
			return nil, err
		}
	}

	if exists, err := rp.HasImage(); err != nil {
		return nil, err
	} else if !exists {
		_, err := rp.PullImage()
		if err != nil {
			return nil, err
		}
	}

	return rp, nil
}

func uniqueJobId() int {
	return <-jobIds
}
