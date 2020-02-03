package main

import (
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors:    false,
		DisableTimestamp: true,
	})
	listenSignals()

	if err := Execute(); err != nil {
		log.Fatalf("Command execution error: %s", err)
	}
}

func listenSignals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			select {
			case sig := <-sigs:
				var exit int
				switch sig {
				case syscall.SIGINT:
					exit = 130
				case syscall.SIGTERM:
					exit = 143
				}
				Abort()
				os.Exit(exit)
			}
		}
	}()
}
