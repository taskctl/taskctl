package main

import (
	"context"

	cmd "github.com/Ensono/taskctl/cmd/taskctl"
)

func main() {
	// init loggerHere or in init function
	cmd.Execute(context.Background())
}
