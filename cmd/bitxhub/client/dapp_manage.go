package client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli"
)

func dappMgrCMD() cli.Command {
	return cli.Command{
		Name:  "dapp",
		Usage: "dapp manage command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "status",
				Usage: "query dapp status by dapp id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify dapp id",
						Required: true,
					},
				},
				Action: getDappStatusById,
			},
			cli.Command{
				Name:  "myDapps",
				Usage: "query dapps by owner addr",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "addr",
						Usage:    "Specify user addr",
						Required: true,
					},
				},
				Action: getDappByOwnerAddr,
			},
			cli.Command{
				Name:  "register",
				Usage: "register dapp",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify dapp contract addr",
						Required: true,
					},
					cli.StringFlag{
						Name:     "name",
						Usage:    "Specify dapp name",
						Required: true,
					},
					cli.StringFlag{
						Name:     "type",
						Usage:    "Specify dapp type",
						Required: true,
					},
					cli.StringFlag{
						Name:     "desc",
						Usage:    "Specify dapp description",
						Required: true,
					},
					cli.StringFlag{
						Name:     "permission",
						Usage:    "Specify users which are not allowed to see the dapp",
						Required: false,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify register reason",
						Required: false,
					},
				},
				Action: registerDapp,
			},
			cli.Command{
				Name:  "freeze",
				Usage: "freeze dapp by dapp id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify dapp id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify freeze reason",
						Required: false,
					},
				},
				Action: freezeDapp,
			},
			cli.Command{
				Name:  "activate",
				Usage: "activate dapp by dapp id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify dapp id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify activate reason",
						Required: false,
					},
				},
				Action: activateDapp,
			},
			cli.Command{
				Name:  "transfer",
				Usage: "transfer dapp to other user",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify dapp id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "addr",
						Usage:    "Specify new owner addr",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify activate reason",
						Required: false,
					},
				},
				Action: transferDapp,
			},
		},
	}
}

func getDappStatusById(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContractBySendView(ctx, constant.DappMgrContractAddr.String(), "GetDapp", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		dapp := &contracts.Dapp{}
		if err := json.Unmarshal(receipt.Ret, dapp); err != nil {
			return fmt.Errorf("unmarshal receipt error: %w", err)
		}
		color.Green("dapp %s is %s", dapp.DappID, string(dapp.Status))
	} else {
		color.Red("get dapp status error: %s\n", string(receipt.Ret))
	}
	return nil
}

func getDappByOwnerAddr(ctx *cli.Context) error {
	addr := ctx.String("addr")

	receipt, err := invokeBVMContractBySendView(ctx, constant.DappMgrContractAddr.String(), "GetDappsByOwner", pb.String(addr))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		var dapps []*contracts.Dapp
		if err := json.Unmarshal(receipt.Ret, &dapps); err != nil {
			return fmt.Errorf("unmarshal receipt error: %w", err)
		}
		printDapp(dapps)
	} else {
		color.Red("get dapp status error: %s\n", string(receipt.Ret))
	}
	return nil
}

func registerDapp(ctx *cli.Context) error {
	id := ctx.String("id")
	name := ctx.String("name")
	typ := ctx.String("type")
	desc := ctx.String("desc")
	reason := ctx.String("reason")
	permissionStr := ctx.String("permission")
	//permissions := strings.Split(permissionStr, ",")

	permissionMap := make(map[string]struct{})
	for _, p := range strings.Split(permissionStr, ",") {
		permissionMap[p] = struct{}{}
	}
	permissionData, err := json.Marshal(permissionMap)
	if err != nil {
		return fmt.Errorf("marshal permission error: %w", err)
	}

	receipt, err := invokeBVMContract(ctx, constant.DappMgrContractAddr.String(), "RegisterDapp",
		pb.String(id),
		pb.String(name),
		pb.String(typ),
		pb.String(desc),
		pb.Bytes(permissionData),
		pb.String(reason),
	)
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		ret := &governance.GovernanceResult{}
		if err := json.Unmarshal(receipt.Ret, ret); err != nil {
			return err
		}
		color.Green("proposal id is %s, dapp id is %s", ret.ProposalID, ret.Extra)
	} else {
		color.Red("register dapp error: %s\n", string(receipt.Ret))
	}
	return nil
}

func freezeDapp(ctx *cli.Context) error {
	id := ctx.String("id")
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.DappMgrContractAddr.String(), "FreezeDapp", pb.String(id), pb.String(reason))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s", proposalId)
	} else {
		color.Red("freeze dapp error: %s\n", string(receipt.Ret))
	}
	return nil
}

func activateDapp(ctx *cli.Context) error {
	id := ctx.String("id")
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.DappMgrContractAddr.String(), "ActivateDapp", pb.String(id), pb.String(reason))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s", proposalId)
	} else {
		color.Red("activate dapp error: %s\n", string(receipt.Ret))
	}
	return nil
}

func transferDapp(ctx *cli.Context) error {
	id := ctx.String("id")
	addr := ctx.String("addr")
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.DappMgrContractAddr.String(), "TransferDapp", pb.String(id), pb.String(addr), pb.String(reason))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s", proposalId)
	} else {
		color.Red("activate dapp error: %s\n", string(receipt.Ret))
	}
	return nil
}

func printDapp(dapps []*contracts.Dapp) {
	var table [][]string
	table = append(table, []string{"Id", "Name", "Type", "Owner", "Createtime", "Score", "Status"})

	for _, dapp := range dapps {
		table = append(table, []string{
			dapp.DappID,
			dapp.Name,
			dapp.Type,
			dapp.OwnerAddr,
			strconv.Itoa(int(dapp.CreateTime)),
			strconv.FormatFloat(dapp.Score, 'g', -1, 64),
			string(dapp.Status),
		})
	}

	fmt.Println("========================================================================================")
	PrintTable(table, true)
}
