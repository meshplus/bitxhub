package client

import (
	"fmt"

	"github.com/urfave/cli"
)

func receiptCMD() cli.Command {
	return cli.Command{
		Name:   "receipt",
		Usage:  "Query receipt",
		Action: getReceipt,
	}
}

func getReceipt(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("please input transaction hash")
	}

	url, err := getURL(ctx, "receipt/"+ctx.Args().Get(0))
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
