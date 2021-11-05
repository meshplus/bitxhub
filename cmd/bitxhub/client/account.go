package client

import (
	"fmt"

	"github.com/urfave/cli"
)

func accountCMD() cli.Command {
	return cli.Command{
		Name:   "account",
		Usage:  "Query account information",
		Action: getAccount,
	}
}

func getAccount(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("lack of account address")
	}

	// get block by height
	url := getURL(ctx, "account_balance/"+ctx.Args().Get(0))
	data, err := httpGet(ctx, url)
	if err != nil {
		return fmt.Errorf("httpGet from url %s failed: %w", url, err)
	}

	ret, err := parseResponse(data)
	if err != nil {
		return fmt.Errorf("wrong response: %w", err)
	}

	retJson, err := prettyJson(ret)
	if err != nil {
		return fmt.Errorf("wrong response: %w", err)
	}

	fmt.Println(retJson)

	return nil
}
