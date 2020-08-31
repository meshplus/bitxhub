package client

import (
	"fmt"
	"github.com/urfave/cli"
)

func IBTPCMD() cli.Command {
	return cli.Command{
		Name:   "ibtp",
		Usage:  "Query ibtp by ID",
		Action: getIBTPByID,
	}
}

func getIBTPByID(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("please input ibtp id")
	}

	url, err := getURL(ctx, "iptp/"+ctx.Args().Get(0))
	if err != nil {
		return err
	}

	data, err := httpGet(url)
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}
