package docker

import "strings"

func splitOutput(output []byte) []string {
	var results []string
	raw := strings.TrimSpace(string(output))
	if len(raw) == 0 {
		return nil
	}

	for _, r := range strings.Split(raw, "\n") {
		results = append(results, strings.TrimSpace(r))
	}

	return results
}
