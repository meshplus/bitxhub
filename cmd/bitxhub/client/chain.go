package client

import (
	"fmt"

	"github.com/urfave/cli"
)

func chainCMD() cli.Command {
	return cli.Command{
		Name:  "chain",
		Usage: "Query BitXHub chain info",
		Subcommands: []cli.Command{
			{
				Name:   "meta",
				Usage:  "Query BitXHub chain meta",
				Action: getChainMeta,
			},
			{
				Name:   "status",
				Usage:  "Query BitXHub chain status",
				Action: getChainStatus,
			},
		},
	}
}

func getChainMeta(ctx *cli.Context) error {
	url := getURL(ctx, "chain_meta")

	data, err := httpGet(ctx, url)
	if err != nil {
		return fmt.Errorf("httpGet from url %s error: %w", url, err)
	}

	fmt.Println(prettyJson(string(data)))

	return nil
}

func getChainStatus(ctx *cli.Context) error {
	url := getURL(ctx, "info?type=0")

	data, err := httpGet(ctx, url)
	if err != nil {
		return fmt.Errorf("httpGet from url %s failed: %w", url, err)
	}

	ret, err := parseResponse(data)
	if err != nil {
		return fmt.Errorf("wrong response: %w", err)
	}

	fmt.Println(ret)

	return nil

}
