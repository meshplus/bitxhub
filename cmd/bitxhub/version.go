package main

import "github.com/urfave/cli"

func versionCMD() cli.Command {
	return cli.Command{
		Name:   "version",
		Usage:  "BitXHub version",
		Action: version,
	}
}

func version(ctx *cli.Context) error {
	printVersion()

	return nil
}
