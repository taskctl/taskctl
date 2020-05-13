package main

import (
	"fmt"
	"github.com/taskctl/taskctl/internal/config"
	"github.com/urfave/cli/v2"
)

func newValidateCommand() *cli.Command {
	cmd := &cli.Command{
		Name:      "validate",
		Usage:     "validates config file",
		ArgsUsage: "some-tasks-file.yaml",
		Before: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("please provide file to validate")
			}

			return nil
		},
		Action: func(c *cli.Context) error {
			cl := config.NewConfigLoader()

			_, err := cl.Load(c.Args().First())
			if err != nil {
				fmt.Println(err.Error())
				return nil
			}

			fmt.Println("file is valid")
			return nil
		},
	}

	return cmd
}
