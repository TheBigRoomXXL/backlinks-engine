package telemetry

import (
	"log"
	"os"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/settings"
)

var (
	ErrorChan = make(chan error)
	logger    *log.Logger
)

func init() {
	s, err := settings.New()
	if err != nil {
		log.Fatal("failed to init telemetry: failed to get settings: %w", err)
	}
	// Start the MetricLogger in a goroutine
	logFile, err := os.Create(s.LOG_PATH)
	if err != nil {
		log.Fatal("failed to init telemetry: failed to create log file: %w", err)
	}
	logger = log.New(logFile, "", log.LUTC)
	go ErrorLogger()
}

func ErrorLogger() {
	done := make(chan bool)
	defer func() {
		done <- true
	}()

	for {
		select {
		case <-done:
			return
		case err := <-ErrorChan:
			Errors.Add(1)
			logger.Println(err.Error())
		}
	}
}
