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
	url, err := getURL(ctx, "account_balance/"+ctx.Args().Get(0))
	if err != nil {
		return err
	}
	data, err := httpGet(url)
	if err != nil {
		return err
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
