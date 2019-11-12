package main

import (
	"github.com/sirupsen/logrus"
	"github.com/trntv/wilson/cmd"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	listenSignals()

	if err := cmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
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
				cmd.Abort()
				os.Exit(exit)
			}
		}
	}()
}
