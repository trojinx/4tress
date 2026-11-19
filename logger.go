package main

import (
	"log"
	"os"
)

func CreateLogger(path string) (*os.File, *log.Logger, error) {
	logFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	return logFile, log.New(logFile, "", log.Ldate|log.Ltime|log.LUTC), nil
}
