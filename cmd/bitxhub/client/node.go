package client

import (
	"encoding/json"
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
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

	data, err := httpGet(ctx, url)
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
		Name:  "delVPNode",
		Usage: "delete a vp node",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "pid",
				Usage: "pid of vp node",
			},
		},
		Action: delVPNode,
	}
}

func delVPNode(ctx *cli.Context) error {
	pid := ctx.String("pid")
	if pid == "" {
		return fmt.Errorf("please input pid")
	}

	url, err := getURL(ctx, "delvpnode")
	if err != nil {
		return err
	}

	p := pb.DelVPNodeRequest{Pid: pid}
	reqData, err := json.Marshal(p)
	data, err := httpPost(ctx, url, reqData)
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}
