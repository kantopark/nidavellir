package scheduler_test

import (
	"log"
	"path/filepath"
	"sync"

	"nidavellir/services/repo"
)

var (
	pythonRepo  *repo.Repo
	longOpsRepo *repo.Repo
)

// clones all repos concurrently
func initRepos() {
	errCh := make(chan error)
	done := make(chan bool, 1)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		pythonRepo = pullRepo("https://github.com/kantopark/python-test-repo", errCh)
	}()

	go func() {
		defer wg.Done()
		longOpsRepo = pullRepo("https://github.com/kantopark/python-test-long-ops-repo", errCh)
	}()

	go func() {
		wg.Wait()
		close(done)
	}()

	for {
		select {
		case err := <-errCh:
			log.Fatal(err) // fails immediately instead of waiting for all
		case <-done:
			return
		}
	}
}

func pullRepo(source string, errCh chan<- error) *repo.Repo {
	name := filepath.Base(source)

	rp, err := repo.NewRepo(source, name)
	if err != nil {
		errCh <- err
		return nil
	}

	if !rp.Exists() {
		err := rp.Clone()
		if err != nil {
			errCh <- err
			return nil
		}
	}

	if exists, err := rp.HasImage(); err != nil {
		errCh <- err
		return nil
	} else if !exists {
		_, err := rp.PullImage()
		if err != nil {
			errCh <- err
			return nil
		}
	}

	return rp
}
