package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ReverseProxySettings struct {
	Port                   string
	EnableRateLimiting     bool
	RateLimitAfter         int
	RateLimitWindowSeconds int
}

type LocalDNS struct {
	Records map[string]string
}

func main() {
	settings, err := ReadConfigJSON("4tress.config.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration error:", err)
		return
	}

	records, err := ReadRecords("4tress.records.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "route configuration error:", err)
		return
	}

	logFile, logger, err := CreateLogger("log.txt")
	if err != nil {
		fmt.Fprintln(os.Stderr, "logger error:", err)
		return
	}
	defer logFile.Close()

	app, err := NewApplication(settings, records.Records, logger)
	if err != nil {
		logger.Println("startup error:", err)
		fmt.Fprintln(os.Stderr, "startup error:", err)
		return
	}
	defer app.Close()

	server := &http.Server{
		Addr:              ":" + settings.Port,
		Handler:           app,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Printf("4tress listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case signal := <-stop:
		logger.Printf("received %s; shutting down", signal)
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			logger.Println("graceful shutdown failed:", err)
		}
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Println("server error:", err)
			fmt.Fprintln(os.Stderr, "server error:", err)
		}
	}
}
