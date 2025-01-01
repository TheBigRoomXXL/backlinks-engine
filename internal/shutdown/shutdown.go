package shutdown

import (
	"os"
	"os/signal"
	"syscall"
)

var signalChan = make(chan os.Signal, 1)
var subscribers = make([]chan struct{}, 0)

func Subscribe(pubsub chan struct{}) {
	subscribers = append(subscribers, pubsub)
}

func Shutdown() {
	// First we publish the shutdown to every subscriber
	for _, sub := range subscribers {
		sub <- struct{}{}
	}
	// Then we want for there "done" message
	for _, pub := range subscribers {
		<-pub
	}
	os.Exit(0)
}

func init() {
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		Shutdown()
	}()
}
