package main

import (
	"github.com/taskctl/taskctl/cmd"
	"log/slog"
	"os"
)

func main() {
	err := cmd.Run()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
