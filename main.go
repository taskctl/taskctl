package main

import (
	"github.com/taskctl/taskctl/cmd"
	"log/slog"
	"os"
)

var version = "dev"

func main() {
	err := cmd.Run(version)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
