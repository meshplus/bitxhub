package main

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "Axiom"
	app.Usage = "A leading inter-blockchain platform"
	app.Compiled = time.Now()

	cli.VersionPrinter = func(c *cli.Context) {
		printVersion()
	}

	// global flags
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "repo",
			Usage: "Axiom storage repo path",
		},
	}

	app.Commands = []*cli.Command{
		configCMD,
		startCMD,
		accountCMD,
		{
			Name:   "start",
			Usage:  "Start a long-running daemon process",
			Action: start,
		},
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Axiom version",
			Action: func(ctx *cli.Context) error {
				printVersion()
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
