package scheduler_test

import (
	"log"
	"path/filepath"
	"sync"

	"nidavellir/services/repo"
)

var (
	pythonRepo   *repo.Repo
	longOpsRepo  *repo.Repo
	failureRepo  *repo.Repo
	exitCodeRepo *repo.Repo
)

// clones all repos concurrently
func initRepos() {
	errCh := make(chan error)
	done := make(chan bool, 1)
	var wg sync.WaitGroup

	sourceDetails := []struct {
		Repo **repo.Repo
		Url  string
	}{
		{&pythonRepo, "https://github.com/kantopark/python-test-repo"},
		{&longOpsRepo, "https://github.com/kantopark/python-test-long-ops-repo"},
		{&failureRepo, "https://github.com/kantopark/python-test-failure-repo"},
		{&exitCodeRepo, "https://github.com/kantopark/python-test-exit-code"},
	}

	wg.Add(len(sourceDetails))

	for _, source := range sourceDetails {
		go func(r **repo.Repo, url string, _errCh chan error) {
			defer wg.Done()
			*r = pullRepo(url, _errCh)
		}(source.Repo, source.Url, errCh)
	}

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

	pat := appConf.PAT
	rp, err := repo.NewRepo(source, name, appDir, pat.Provider, pat.Token)
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
