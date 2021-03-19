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

	data, err := getTxReceipt(ctx, ctx.Args().Get(0))
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}

func getTxReceipt(ctx *cli.Context, hash string) ([]byte, error) {
	url, err := getURL(ctx, "receipt/"+hash)
	if err != nil {
		return nil, err
	}

	data, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}

	return data, nil
}
