package telemetry

import (
	"log"
	"os"
	"strings"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/settings"
)

var (
	ErrorChan = make(chan error)
	logger    *log.Logger
)

func init() {
	s, err := settings.New()
	if err != nil {
		log.Fatal("failed to init telemetry: failed to get settings: ", err)
	}
	// Start the MetricLogger in a goroutine
	logFile, err := os.Create(s.LOG_PATH)
	if err != nil {
		log.Fatal("failed to init telemetry: failed to create log file: ", err)
	}
	logger = log.New(logFile, "", log.LstdFlags)
	go ErrorHandler()
}

func ErrorHandler() {
	done := make(chan bool)
	defer func() {
		done <- true
	}()

	for {
		select {
		case <-done:
			close(ErrorChan)
			return
		case err := <-ErrorChan:
			Errors.Add(1)
			if strings.Contains(err.Error(), "i/o timeout") {
				TCPTimeout.Add(1)
			}
			logger.Println(err.Error())
		}
	}
}
