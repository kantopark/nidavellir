package image

import (
	"github.com/pkg/errors"

	"nidavellir/services/runtime"
)

func prepareDockerfile(workDir string) (string, error) {
	r, err := runtime.FromDir(workDir)
	if err != nil {
		return "", err
	}

	switch r.Runtime {
	case "dockerfile":
		return "FilePath", nil
	case "python", "r":
		file, err := newDockerfile(r.Runtime, workDir)
		if err != nil {
			return "", err
		}
		if err := file.fetchFile(); err != nil {
			return "", err
		}

		if err := file.writeRequirements(); err != nil {
			return "", err
		}

		if file.HasChanges {
			if err := file.createDockerfile(); err != nil {
				return "", err
			}
			return file.FilePath, nil
		}

		return "", nil
	default:
		return "", errors.Errorf("unsupported runtime '%s'", r.Runtime)
	}
}
