package client

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli"
)

func appchainMgrCMD() cli.Command {
	return cli.Command{
		Name:  "chain",
		Usage: "appchain manage command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "status",
				Usage: "query chain status by chain id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "chain id",
						Required: true,
					},
				},
				Action: getChainStatusById,
			},
			cli.Command{
				Name:  "freeze",
				Usage: "freeze appchain by chain id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "chain id",
						Required: true,
					},
				},
				Action: freezeAppchain,
			},
			cli.Command{
				Name:  "activate",
				Usage: "activate chain by chain id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "chain id",
						Required: true,
					},
				},
				Action: activateAppchain,
			},
		},
	}
}

func getChainStatusById(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContract(ctx, constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		chain := &appchainMgr.Appchain{}
		if err := json.Unmarshal(receipt.Ret, chain); err != nil {
			return fmt.Errorf("unmarshal receipt error: %w", err)
		}
		color.Green("appchain %s is %s", chain.ID, string(chain.Status))
	} else {
		color.Red("get chain status error: %s\n", string(receipt.Ret))
	}
	return nil
}

func freezeAppchain(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContract(ctx, constant.AppchainMgrContractAddr.String(), "FreezeAppchain", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s", proposalId)
	} else {
		color.Red("freeze appchain error: %s\n", string(receipt.Ret))
	}
	return nil
}

func activateAppchain(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContract(ctx, constant.AppchainMgrContractAddr.String(), "ActivateAppchain", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s", proposalId)
	} else {
		color.Red("activate appchain error: %s\n", string(receipt.Ret))
	}
	return nil
}
