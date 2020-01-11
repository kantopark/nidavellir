package scheduler

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

func logOutput(file *os.File, content interface{}) {
	mw := io.MultiWriter(os.Stdout, file)
	logger := log.New()
	logger.SetOutput(mw)
	logger.Println(content)
}
