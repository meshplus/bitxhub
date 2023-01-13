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
			{
				Name:  "tps",
				Usage: "Query BitXHub tps",
				Flags: []cli.Flag{
					cli.Uint64Flag{
						Name:     "begin",
						Usage:    "Specify begin block number",
						Required: true,
					},
					cli.Uint64Flag{
						Name:     "end",
						Usage:    "Specify end block num",
						Required: true,
					},
				},
				Action: getTps,
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

func getTps(ctx *cli.Context) error {
	begin := ctx.Uint64("begin")
	end := ctx.Uint64("end")
	url := getURL(ctx, fmt.Sprintf("tps/%d/%d", begin, end))

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
