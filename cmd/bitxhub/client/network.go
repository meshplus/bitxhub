package client

import (
	"fmt"

	"github.com/urfave/cli"
)

func networkCMD() cli.Command {
	return cli.Command{
		Name:   "network",
		Usage:  "Query network info from node",
		Action: network,
	}
}

func network(ctx *cli.Context) error {
	url := getURL(ctx, "info?type=1")

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
