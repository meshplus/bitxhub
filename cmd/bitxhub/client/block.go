package client

import (
	"fmt"

	"github.com/spf13/cast"
	"github.com/urfave/cli"
)

func blockCMD() cli.Command {
	return cli.Command{
		Name:   "block",
		Usage:  "Query block",
		Action: getBlock,
	}
}

func getBlock(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("please input block height or block hash")
	}

	input := ctx.Args().Get(0)

	height, err := cast.ToUint64E(input)
	if err != nil {
		return getBlockByHash(ctx, input)
	}

	if err := getBlockByHeight(ctx, height); err != nil {
		return err
	}

	return nil
}

func getBlockByHeight(ctx *cli.Context, height uint64) error {
	url, err := getURL(ctx, fmt.Sprintf("block?type=0&value=%d", height))
	if err != nil {
		return err
	}
	data, err := httpGet(ctx, url)
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}

func getBlockByHash(ctx *cli.Context, hash string) error {
	url, err := getURL(ctx, fmt.Sprintf("block?type=1&value=%s", hash))
	if err != nil {
		return err
	}
	data, err := httpGet(ctx, url)
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}
