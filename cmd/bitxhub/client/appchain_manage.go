package client

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/fatih/color"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli"
)

func appchainMgrCMD() cli.Command {
	return cli.Command{
		Name:  "appchain",
		Usage: "Appchain manage command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "info",
				Usage: "Query appchain info by appchain name",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "name",
						Usage:    "Specify appchain name",
						Required: true,
					},
				},
				Action: getChainByName,
			},
			cli.Command{
				Name:  "status",
				Usage: "Query chain status by appchain id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify appchain id",
						Required: true,
					},
				},
				Action: getChainStatusById,
			},
			cli.Command{
				Name:  "freeze",
				Usage: "Freeze appchain by appchain id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify appchain id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify freeze reason",
						Required: false,
					},
				},
				Action: freezeAppchain,
			},
			cli.Command{
				Name:  "activate",
				Usage: "Activate appchain by appchain id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify appchain id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify activate reason",
						Required: false,
					},
				},
				Action: activateAppchain,
			},
		},
	}
}

func getChainByName(ctx *cli.Context) error {
	name := ctx.String("name")

	receipt, err := invokeBVMContractBySendView(ctx, constant.AppchainMgrContractAddr.String(), "GetAppchainByName", pb.String(name))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when get appchain by name %s: %w", name, err)
	}

	if receipt.IsSuccess() {
		chain := &appchainMgr.Appchain{}
		if err := json.Unmarshal(receipt.Ret, chain); err != nil {
			return fmt.Errorf("unmarshal receipt error: %w", err)
		}
		printChain(chain)
	} else {
		color.Red("get chain error: %s\n", string(receipt.Ret))
	}
	return nil
}

func printChain(chain *appchainMgr.Appchain) {
	var table [][]string
	table = append(table, []string{"Id", "Name", "Type", "Broker", "Status", "Desc", "Version"})

	table = append(table, []string{
		chain.ID,
		chain.ChainName,
		chain.ChainType,
		string(chain.Broker),
		string(chain.Status),
		chain.Desc,
		strconv.Itoa(int(chain.Version)),
	})
	PrintTable(table, true)
}

func getChainStatusById(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContractBySendView(ctx, constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(id))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when get appchain status by ID %s: %w", id, err)
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
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.AppchainMgrContractAddr.String(), "FreezeAppchain", pb.String(id), pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when freeze appchain %s for %s: %w", id, reason, err)
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
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.AppchainMgrContractAddr.String(), "ActivateAppchain", pb.String(id), pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when activate appchain %s for %s: %w", id, reason, err)
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s", proposalId)
	} else {
		color.Red("activate appchain error: %s\n", string(receipt.Ret))
	}
	return nil
}
