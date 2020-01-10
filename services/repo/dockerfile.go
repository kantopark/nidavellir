package repo

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"nidavellir/libs"
)

type dockerfile struct {
	Lang       string
	WorkDir    string
	Content    string
	HasChanges bool
	FilePath   string
}

func newDockerfile(lang, workDir string) (*dockerfile, error) {
	if !libs.IsIn(lang, []string{"python"}) {
		return nil, errors.Errorf("unsupported dockerfile language: %s", lang)
	}

	return &dockerfile{
		Lang:     lang,
		WorkDir:  workDir,
		FilePath: "build.FilePath",
	}, nil
}

func (d *dockerfile) fetchFile() error {
	var url string
	switch d.Lang {
	case "python":
		url = "https://raw.githubusercontent.com/kantopark/nidavellir/master/dockerfiles/python.FilePath"
	}

	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrapf(err, "could not get '%s' dockerfile", d.Lang)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "could not retrieve dockerfile body content")
	}

	d.Content = string(body)
	return nil
}

func (d *dockerfile) writeRequirements() error {
	req := "requirements.txt"
	if !libs.PathExists(filepath.Join(d.WorkDir, req)) {
		return nil
	}

	switch d.Lang {
	case "python":
		d.HasChanges = true
		d.Content = strings.Replace(d.Content, "# TEMPLATE LINE OVERWRITE", strings.TrimSpace(fmt.Sprintf(`
COPY %s %s
RUN pip install -f %s
`, req, req, req)), 1)
	}

	return nil
}

func (d *dockerfile) createDockerfile() error {
	err := ioutil.WriteFile(d.FilePath, []byte(d.Content), 0777)
	if err != nil {
		return errors.Wrapf(err, "could not create '%s'", d.FilePath)
	}

	return nil
}
