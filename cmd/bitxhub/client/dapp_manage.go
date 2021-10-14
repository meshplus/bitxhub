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
				Name:   "all",
				Usage:  "query all dapps",
				Action: getAllDapps,
			},
			cli.Command{
				Name:  "permission",
				Usage: "query permission dapps",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "caller",
						Usage:    "Specify caller addr",
						Required: true,
					},
				},
				Action: getPermissionDapps,
			},
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
						Name:     "name",
						Usage:    "Specify dapp name",
						Required: true,
					},
					cli.StringFlag{
						Name:     "type",
						Usage:    "Specify dapp type, one of tool, application, game and others",
						Required: true,
					},
					cli.StringFlag{
						Name:     "desc",
						Usage:    "Specify dapp description",
						Required: true,
					},
					cli.StringFlag{
						Name:     "url",
						Usage:    "Specify dapp url",
						Required: true,
					},
					cli.StringFlag{
						Name:     "contractAddrs",
						Usage:    "Specify dapp contract addr. If there are multiple contract addresses, separate them with ','",
						Required: true,
					},
					cli.StringFlag{
						Name:     "permission",
						Usage:    "Specify the addr of users which are not allowed to see the dapp. If there are multiple contract addresses, separate them with ','",
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
				Name:  "update",
				Usage: "update dapp",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify dapp id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "name",
						Usage:    "Specify dapp name",
						Required: false,
					},
					cli.StringFlag{
						Name:     "desc",
						Usage:    "Specify dapp description",
						Required: false,
					},
					cli.StringFlag{
						Name:     "url",
						Usage:    "Specify dapp url",
						Required: false,
					},
					cli.StringFlag{
						Name:     "contractAddrs",
						Usage:    "Specify dapp contract addr. If there are multiple contract addresses, separate them with ','",
						Required: false,
					},
					cli.StringFlag{
						Name:     "permission",
						Usage:    "Specify the addr of users which are not allowed to see the dapp. If there are multiple contract addresses, separate them with ','",
						Required: false,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify register reason",
						Required: false,
					},
				},
				Action: updateDapp,
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
						Usage:    "Specify freeze reason",
						Required: false,
					},
				},
				Action: transferDapp,
			},
			cli.Command{
				Name:  "confirm",
				Usage: "confirm dapp transfer",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify dapp id",
						Required: true,
					},
				},
				Action: confirmDapp,
			},
			cli.Command{
				Name:  "evaluate",
				Usage: "evaluate dapp",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify dapp id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "desc",
						Usage:    "Specify evaluate desc",
						Required: true,
					},
					cli.Float64Flag{
						Name:     "score",
						Usage:    "Specify evaluate score, [0,5]",
						Required: false,
					},
				},
				Action: evaluateDapp,
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

func getAllDapps(ctx *cli.Context) error {
	receipt, err := invokeBVMContractBySendView(ctx, constant.DappMgrContractAddr.String(), "GetAllDapps")
	if err != nil {
		return fmt.Errorf("GetAllDapps error: %v", err)
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

func getPermissionDapps(ctx *cli.Context) error {
	caller := ctx.String("caller")
	receipt, err := invokeBVMContractBySendView(ctx, constant.DappMgrContractAddr.String(), "GetPermissionDapps", pb.String(caller))
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
	name := ctx.String("name")
	typ := ctx.String("type")
	desc := ctx.String("desc")
	url := ctx.String("url")
	contractAddrs := strings.TrimSpace(ctx.String("contractAddrs"))
	reason := ctx.String("reason")
	permissionStr := strings.TrimSpace(ctx.String("permission"))

	receipt, err := invokeBVMContract(ctx, constant.DappMgrContractAddr.String(), "RegisterDapp",
		pb.String(name),
		pb.String(typ),
		pb.String(desc),
		pb.String(url),
		pb.String(contractAddrs),
		pb.String(permissionStr),
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

func updateDapp(ctx *cli.Context) error {
	id := ctx.String("id")
	name := ctx.String("name")
	desc := ctx.String("desc")
	url := ctx.String("url")
	contractAddrs := strings.TrimSpace(ctx.String("contractAddrs"))
	reason := ctx.String("reason")
	permissionStr := strings.TrimSpace(ctx.String("permission"))

	receipt, err := invokeBVMContractBySendView(ctx, constant.DappMgrContractAddr.String(), "GetDapp",
		pb.String(id),
	)
	if err != nil {
		return err
	}
	if receipt.IsSuccess() {
		dapp := &contracts.Dapp{}
		if err := json.Unmarshal(receipt.Ret, dapp); err != nil {
			return fmt.Errorf("unmarshal receipt error: %w", err)
		}
		if name == "" {
			name = dapp.Name
		}
		if desc == "" {
			desc = dapp.Desc
		}
		if url == "" {
			url = dapp.Url
		}
		if contractAddrs == "" {
			for k, _ := range dapp.ContractAddr {
				if contractAddrs == "" {
					contractAddrs = k
				} else {
					contractAddrs = fmt.Sprintf("%s,%s", contractAddrs, k)
				}
			}

		}
		if permissionStr == "" {
			for k, _ := range dapp.Permission {
				if permissionStr == "" {
					permissionStr = k
				} else {
					permissionStr = fmt.Sprintf("%s,%s", permissionStr, k)
				}
			}

		}
	} else {
		color.Red("get dapp info error: %s\n", string(receipt.Ret))
	}

	receipt, err = invokeBVMContract(ctx, constant.DappMgrContractAddr.String(), "UpdateDapp",
		pb.String(id),
		pb.String(name),
		pb.String(desc),
		pb.String(url),
		pb.String(contractAddrs),
		pb.String(permissionStr),
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
		if ret.ProposalID != "" {
			color.Green("proposal id is %s", ret.ProposalID)
		} else {
			color.Green("update dapp success")
		}
	} else {
		color.Red("update dapp error: %s\n", string(receipt.Ret))
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
		color.Red("transfer dapp error: %s\n", string(receipt.Ret))
	}
	return nil
}

func confirmDapp(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContract(ctx, constant.DappMgrContractAddr.String(), "ConfirmTransfer", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		color.Green("confirm dapp transfer success")
	} else {
		color.Red("confirm dapp transfer error: %s\n", string(receipt.Ret))
	}
	return nil
}

func evaluateDapp(ctx *cli.Context) error {
	id := ctx.String("id")
	desc := ctx.String("addr")
	score := ctx.Float64("score")

	receipt, err := invokeBVMContract(ctx, constant.DappMgrContractAddr.String(), "EvaluateDapp", pb.String(id), pb.String(desc), pb.Float64(score))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		color.Green("evaluate dapp success")
	} else {
		color.Red("evaluate dapp error: %s\n", string(receipt.Ret))
	}
	return nil
}

func printDapp(dapps []*contracts.Dapp) {
	var table [][]string
	table = append(table, []string{"Id", "Name", "Type", "Owner", "Createtime", "Score", "Status", "TranRec"})

	for _, dapp := range dapps {
		transStr := ""
		for _, t := range dapp.TransferRecords {
			transStr = fmt.Sprintf("%s,%s-%s-%d", transStr, t.From, t.To, t.CreateTime)
		}
		table = append(table, []string{
			dapp.DappID,
			dapp.Name,
			string(dapp.Type),
			dapp.OwnerAddr,
			strconv.Itoa(int(dapp.CreateTime)),
			strconv.FormatFloat(dapp.Score, 'g', -1, 64),
			string(dapp.Status),
			transStr,
		})
	}

	fmt.Println("========================================================================================")
	PrintTable(table, true)
}
