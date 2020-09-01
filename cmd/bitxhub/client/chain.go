package client

import (
	"fmt"

	"github.com/urfave/cli"
)

func chainCMD() cli.Command {
	return cli.Command{
		Name:  "chain",
		Usage: "Query bitxhub chain info",
		Subcommands: []cli.Command{
			{
				Name:   "meta",
				Usage:  "Query bitxhub chain meta",
				Action: getChainMeta,
			},
			{
				Name:   "status",
				Usage:  "Query bitxhub chain status",
				Action: getChainStatus,
			},
		},
	}
}

func getChainMeta(ctx *cli.Context) error {
	url, err := getURL(ctx, "chain_meta")
	if err != nil {
		return err
	}

	data, err := httpGet(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}

	fmt.Println(string(data))

	return nil
}

func getChainStatus(ctx *cli.Context) error {
	url, err := getURL(ctx, "info?type=0")
	if err != nil {
		return err
	}

	data, err := httpGet(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}

	ret, err := parseResponse(data)
	if err != nil {
		return err
	}

	retJson, err := prettyJson(ret)
	if err != nil {
		return fmt.Errorf("wrong response: %w", err)
	}

	fmt.Println(retJson)

	return nil

}
