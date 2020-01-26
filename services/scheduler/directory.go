package scheduler

import (
	"strings"

	"nidavellir/config"
)

func logFilePath(dirname, runDate string) string {
	conf, _ := config.New()
	return conf.App.LogFilePath(dirname, runDate)
}

func imageLogPath(imageName string) string {
	parts := strings.Split(imageName, ":")

	conf, _ := config.New()
	return conf.App.ImageBuildLogPath(parts[0], parts[1])
}

func outputDir(jobId int) string {
	conf, _ := config.New()
	return conf.App.OutputDir(jobId)
}
