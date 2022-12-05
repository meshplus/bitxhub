package client

import (
	"fmt"

	"github.com/urfave/cli"
)

func receiptCMD() cli.Command {
	return cli.Command{
		Name:   "receipt",
		Usage:  "Query transaction receipt by transaction hash",
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

	fmt.Println(prettyJson(string(data)))

	return nil
}

func getTxReceipt(ctx *cli.Context, hash string) ([]byte, error) {
	url := getURL(ctx, "receipt/"+hash)

	data, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}

	return data, nil
}
