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
	if !libs.IsIn(lang, []string{"dockerfile", "python"}) {
		return nil, errors.Errorf("unsupported dockerfile language: %s", lang)
	}

	return &dockerfile{
		Lang:     lang,
		WorkDir:  workDir,
		FilePath: "build.Dockerfile",
	}, nil
}

// Fetches the template file from the github repo. This is used when the user uses the default
// Dockerfile that is provided
func (d *dockerfile) fetchFile() error {
	var url string
	switch d.Lang {
	case "python":
		url = "https://raw.githubusercontent.com/kantopark/nidavellir/master/dockerfiles/python.Dockerfile"
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

// Loads the template file from the user's working directory. This is used when the user specifies his
// or her own Dockerfile
func (d *dockerfile) loadContent() error {
	fp := filepath.Join(d.WorkDir, "Dockerfile")
	if !libs.PathExists(fp) {
		return errors.New("dockerfile missing")
	}

	content, err := ioutil.ReadFile(fp)
	if err != nil {
		return errors.New("could not read Dockerfile")
	}

	d.Content = strings.TrimSpace(string(content))
	d.HasChanges = true
	return nil
}

// Adds any build arguments to the Dockerfile. This is useful when the server needs to set variables for
// building which the user may not and should not be aware of. Examples include http_proxy and https_proxy
// Instead of arguments, these values are actually injected as environment variables which will be removed
// later
func (d *dockerfile) writeBuildArgs(buildArgs map[string]string) {
	if len(buildArgs) == 0 {
		return
	}

	lines := strings.Split(d.Content, "\n")
	var newLines []string
	// get line number which starts with FROM (which is the start of Dockerfile)
	start := 0
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "FROM") {
			start = i
			newLines = append(newLines, line, "\n")
		}
	}

	// add build env variables
	envs := []string{"ENV"}
	for key, value := range buildArgs {
		envs = append(envs, fmt.Sprintf("%s=%s", key, value))
	}
	newLines = append(newLines, strings.Join(envs, " "))

	// add rest of content. Stops when we encounter ENTRYPOINT, which is when
	// we'll need to unset the env variables we injected before
	newLines = append(newLines, "\n")
	end := len(lines)
	for i, line := range lines[start:] {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "ENTRYPOINT") {
			end = i
			break
		}

		newLines = append(newLines, strings.TrimSpace(line))
	}

	// unset build env
	envs = []string{"ENV"}
	for key := range buildArgs {
		envs = append(envs, fmt.Sprintf("%s=", key))
	}
	newLines = append(newLines, strings.Join(envs, " "))

	// add entrypoint and any other remaining cmd that follow after ENTRYPOINT
	for _, line := range lines[end:] {
		newLines = append(newLines, strings.TrimSpace(line))
	}

	// form content
	d.Content = strings.Join(newLines, "\n")
	d.HasChanges = true
}

// Includes any additional requirements that the user has
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

// creates the new dockerfile in the repo's working directory which will then be used
// by Docker to build the image
func (d *dockerfile) createDockerfile() error {
	err := ioutil.WriteFile(d.FilePath, []byte(d.Content), 0777)
	if err != nil {
		return errors.Wrapf(err, "could not create '%s'", d.FilePath)
	}

	return nil
}
