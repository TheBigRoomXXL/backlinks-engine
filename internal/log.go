package internal

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var logger *log.Logger
var counterRequest = make(chan struct{})
var counterError = make(chan error)

func initLogger(s *Settings) {
	// Start the MetricLogger in a goroutine
	logFile, err := os.Create(s.LOG_PATH)
	if err != nil {
		log.Fatal(err)
	}
	logger = log.New(logFile, "", log.LUTC)
	go MetricLogger()
}

func MetricLogger() {
	ticker := time.NewTicker(10 * time.Second)
	start := time.Now()
	requests := 0
	errors := 0
	timeouts := 0
	fmt.Println("┌───────────────┬───────────────┬───────────────┬───────────────┐")
	fmt.Println("│     Time      │   requests    │    errors     │   timeouts    │")
	fmt.Println("├───────────────┼───────────────┼───────────────┼───────────────┤")
	for {
		for {
			select {
			case <-ticker.C:
				time := time.Since(start).Round(time.Second)
				fmt.Printf(
					"│ %13s │ %13d │ %13d │ %13d │\n",
					time, requests, errors, timeouts,
				)
			case <-counterRequest:
				requests++
			case e := <-counterError:
				errors++
				logger.Println(e.Error())
				if strings.Contains(strings.ToLower(e.Error()), "timeout") {
					timeouts++
				}
			}
		}
	}
}
