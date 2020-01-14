package dkcontainer

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"nidavellir/services/docker/dkutils"
)

type SearchOptions struct {
	Name string
	Port int
}

func Search(options *SearchOptions) ([]string, error) {
	sep := "::"
	cmd := exec.Command("docker", "container", "list", "-a", "--format", "{{.Names}}"+sep+"{{.Ports}}"+sep+"{{.ID}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "could not get list of containers")
	}

	idMap := make(map[string]int)
	for _, namesPorts := range dkutils.SplitOutput(output) {
		parts := strings.Split(namesPorts, "::")
		id := parts[2]

		for _, name := range strings.Split(parts[0], ",") {
			if options.Name != "" && name == options.Name {
				idMap[id] = 0
			}
		}

		for _, fullAddress := range strings.Split(parts[1], ",") {
			if addresses := strings.Split(fullAddress, "->"); len(addresses) == 2 {
				if _parts := strings.Split(addresses[0], ":"); len(_parts) == 2 {
					if port, err := strconv.Atoi(_parts[1]); err == nil && options.Port == port {
						idMap[id] = 0
					}
				}
			}
		}
	}

	var ids []string
	for k := range idMap {
		ids = append(ids, k)
	}

	return ids, nil
}
