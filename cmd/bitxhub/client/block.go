package client

import (
	"fmt"

	"github.com/spf13/cast"
	"github.com/urfave/cli"
)

func blockCMD() cli.Command {
	return cli.Command{
		Name:   "block",
		Usage:  "Query block by block hash or block height",
		Action: getBlock,
	}
}

func getBlock(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return getLatestBlock(ctx)
	}

	input := ctx.Args().Get(0)

	height, err := cast.ToUint64E(input)
	if err != nil {
		return getBlockByHash(ctx, input)
	}

	if err := getBlockByHeight(ctx, height); err != nil {
		return fmt.Errorf("get block with height %d failed: %w", height, err)
	}

	return nil
}

func getBlockByHeight(ctx *cli.Context, height uint64) error {
	url := getURL(ctx, fmt.Sprintf("block?type=0&value=%d", height))
	data, err := httpGet(ctx, url)
	if err != nil {
		return fmt.Errorf("httpGet from url %s failed: %w", url, err)
	}

	fmt.Println(prettyJson(string(data)))

	return nil
}

func getBlockByHash(ctx *cli.Context, hash string) error {
	url := getURL(ctx, fmt.Sprintf("block?type=1&value=%s", hash))
	data, err := httpGet(ctx, url)
	if err != nil {
		return fmt.Errorf("httpGet from url %s failed: %w", url, err)
	}

	fmt.Println(prettyJson(string(data)))
	return nil
}

func getLatestBlock(ctx *cli.Context) error {
	url := getURL(ctx, "block?type=2")
	data, err := httpGet(ctx, url)
	if err != nil {
		return fmt.Errorf("httpGet from url %s failed: %w", url, err)
	}

	fmt.Println(prettyJson(string(data)))
	return nil
}
