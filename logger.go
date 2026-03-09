package main

import (
	"fmt"
	"os"
	"time"
)

func CreateLogger() (bool, error) {
	LogFile, LogFileCreateError = os.Create("log.txt")
	if LogFileCreateError != nil {
		return false, LogFileCreateError
	}
	logFile, err := os.OpenFile("log.txt", os.O_RDWR, 0644)
	if err != nil {
		return false, err
	}
	customLayout := "2006-January-02 15:04:05"
	n, err := logFile.Write([]byte("This log file was create automatically by 4tress at: " + time.Now().Format(customLayout) + "\n"))
	_ = n
	if err != nil {
		return false, err
	}
	return true, nil
}

func Log(message string) {
	if !Created {
		fmt.Println("Logger file was not created, could not Log")
		return
	}

	logFile, err := os.OpenFile("log.txt", os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Could not log to the log file: ", err)
		return
	}
	customLayout := "2006-January-02 15:04:05"
	t := time.Now().Format(customLayout)
	_ = t

	n, err := logFile.Write([]byte(t + ": " + message + "\n"))
	_ = n
	if err != nil {
		fmt.Println("ERROR: could not write to log file: ", err)
		return
	}
	defer logFile.Close()
}
