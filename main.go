package main

import (
	"github.com/sirupsen/logrus"
	"github.com/taskctl/taskctl/cmd/taskctl_cmd"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors:   false,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   false,
	})

	err := taskctl_cmd.Run()
	if err != nil {
		logrus.Fatal(err)
	}
}
