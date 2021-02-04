package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("prometheus cleaner \n")
	cleaner, err := newCleaner("./job.yml")
	if err != nil {
		os.Exit(1)
	}
	if err := cleaner.Do(); err != nil {
		panic(err)
	}
	log.Info("success")
}
