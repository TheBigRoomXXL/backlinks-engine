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
	s, _ := settings.New()
	// Start the MetricLogger in a goroutine
	// TODO: append instead of tunc once tests app is stable
	logFile, err := os.OpenFile(s.LOG_PATH, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
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
