package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/fatih/color"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/governance"
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
				Name:  "register",
				Usage: "register appchain",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "method",
						Usage:    "Specific did sub method name(like appchain)",
						Required: true,
					},
					cli.StringFlag{
						Name:     "doc-addr",
						Usage:    "Specify the addr of did document",
						Required: true,
					},
					cli.StringFlag{
						Name:     "doc-hash",
						Usage:    "Specify the hash of did document",
						Required: true,
					},
					cli.StringFlag{
						Name:     "name",
						Usage:    "Specific appchain name",
						Required: true,
					},
					cli.StringFlag{
						Name:     "type",
						Usage:    "Specific appchain type",
						Required: true,
					},
					cli.StringFlag{
						Name:     "desc",
						Usage:    "Specific appchain description",
						Required: true,
					},
					cli.StringFlag{
						Name:     "version",
						Usage:    "Specific appchain version",
						Required: true,
					},
					cli.StringFlag{
						Name:     "validators",
						Usage:    "Specific appchain validators path",
						Required: true,
					},
					cli.StringFlag{
						Name:     "consensus",
						Usage:    "Specific appchain consensus type",
						Required: true,
					},
					cli.StringFlag{
						Name:     "pubkey",
						Usage:    "Specify appchain pubkey",
						Required: true,
					},
				},
				Action: registerAppchain,
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

	receipt, err := invokeBVMContractBySendView(ctx, constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(id))
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

func registerAppchain(ctx *cli.Context) error {
	method := ctx.String("method")
	didDocAddr := ctx.String("doc-addr")
	didDocHash := ctx.String("doc-hash")
	name := ctx.String("name")
	typ := ctx.String("type")
	desc := ctx.String("desc")
	version := ctx.String("version")
	validatorsPath := ctx.String("validators")
	consensus := ctx.String("consensus")
	pubkey := ctx.String("pubkey")
	validatorData, err := ioutil.ReadFile(validatorsPath)
	if err != nil {
		return fmt.Errorf("read validators file: %w", err)
	}

	receipt, err := invokeBVMContract(ctx, constant.AppchainMgrContractAddr.String(), "Register",
		pb.String(method),
		pb.String(didDocAddr),
		pb.String(didDocHash),
		pb.String(string(validatorData)),
		pb.String(consensus),
		pb.String(typ),
		pb.String(name),
		pb.String(desc),
		pb.String(version),
		pb.String(pubkey),
	)
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		ret := &governance.GovernanceResult{}
		if err := json.Unmarshal(receipt.Ret, ret); err != nil {
			return err
		}
		color.Green("proposal id is %s, chain id is %s", ret.ProposalID, ret.Extra)
	} else {
		color.Red("register appchain error: %s\n", string(receipt.Ret))
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
