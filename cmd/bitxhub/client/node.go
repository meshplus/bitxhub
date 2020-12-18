package client

import (
	"fmt"

	"github.com/urfave/cli"
)

func validatorsCMD() cli.Command {
	return cli.Command{
		Name:   "validators",
		Usage:  "Query validator address",
		Action: getValidators,
	}
}

func getValidators(ctx *cli.Context) error {
	url, err := getURL(ctx, "info?type=2")
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

func delVPNodeCMD() cli.Command {
	return cli.Command{
		Name:   "delVPNode",
		Usage:  "delete a vp node",
		Action: delVPNode,
	}
}

func delVPNode(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("please input pid")
	}

	url, err := getURL(ctx, "delvpnode/"+ctx.Args().Get(0))
	if err != nil {
		return err
	}

	// TODO (FBZ): change to httpPost
	data, err := httpGet(url)
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}