package scheduler

import (
	"strings"

	"nidavellir/config"
)

func repoDir(dirname string) string {
	conf, _ := config.New()
	return conf.WorkDir.RepoPath(dirname)
}

func logFilePath(dirname, runDate string) string {
	conf, _ := config.New()
	return conf.WorkDir.LogFilePath(dirname, runDate)
}

func imageLogPath(imageName string) string {
	parts := strings.Split(imageName, ":")

	conf, _ := config.New()
	return conf.WorkDir.ImageBuildLogPath(parts[0], parts[1])
}

func outputDir(jobId int) string {
	conf, _ := config.New()
	return conf.WorkDir.OutputDir(jobId)
}
