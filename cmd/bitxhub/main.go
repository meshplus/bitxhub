package main

import (
	"fmt"
	"os"
	"time"

	"github.com/meshplus/bitxhub/cmd/bitxhub/client"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "BitXHub"
	app.Usage = "A leading inter-blockchain platform"
	app.Compiled = time.Now()

	cli.VersionPrinter = func(c *cli.Context) {
		printVersion()
	}

	// global flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "repo",
			Usage: "BitXHub storage repo path",
		},
	}

	app.Commands = []cli.Command{
		configCMD(),
		initCMD(),
		startCMD(),
		keyCMD(),
		versionCMD(),
		certCMD,
		client.LoadClientCMD(),
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
