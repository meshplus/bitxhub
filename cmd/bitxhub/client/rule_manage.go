package client

import (
	"encoding/json"
	"fmt"

	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"

	"github.com/fatih/color"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/urfave/cli"
)

func ruleMgrCMD() cli.Command {
	return cli.Command{
		Name:  "rule",
		Usage: "rule manage command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "all",
				Usage: "query all rules info of one chain",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "chain id",
						Required: true,
					},
				},
				Action: getRulesList,
			},
			cli.Command{
				Name:  "available",
				Usage: "query available rule address of a chain",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "chain id",
						Required: true,
					},
				},
				Action: getAvailableRuleAddress,
			},
			cli.Command{
				Name:  "status",
				Usage: "query rule status by rule address and chain id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "chain id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "addr",
						Usage:    "rule addr",
						Required: true,
					},
				},
				Action: getRuleStatus,
			},
			cli.Command{
				Name:  "bind",
				Usage: "bind rule with chain id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "chain id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "addr",
						Usage:    "rule address",
						Required: true,
					},
				},
				Action: bindRule,
			},
			cli.Command{
				Name:  "unbind",
				Usage: "unbind rule with chain id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "chain id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "addr",
						Usage:    "rule address",
						Required: true,
					},
				},
				Action: unbindRule,
			},
			cli.Command{
				Name:  "freeze",
				Usage: "freeze rule by chain id and rule address",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "chain id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "addr",
						Usage:    "rule address",
						Required: true,
					},
				},
				Action: freezeRule,
			},
			cli.Command{
				Name:  "activate",
				Usage: "activate rule by chain id and rule address",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "chain id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "addr",
						Usage:    "rule address",
						Required: true,
					},
				},
				Action: activateRule,
			},
		},
	}
}

func getRulesList(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContract(ctx, constant.RuleManagerContractAddr.String(), "Rules", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		rules := make([]*ruleMgr.Rule, 0)
		if receipt.Ret != nil {
			if err := json.Unmarshal(receipt.Ret, &rules); err != nil {
				return fmt.Errorf("unmarshal rules error: %w", err)
			}
		}
		printRule(rules)
	} else {
		color.Red("get rules error: %s\n", string(receipt.Ret))
	}
	return nil
}

func getAvailableRuleAddress(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContract(ctx, constant.RuleManagerContractAddr.String(), "GetAvailableRuleAddr", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		color.Green("available rule address is %s", string(receipt.Ret))
	} else {
		color.Red("get available rule address error: %s\n", string(receipt.Ret))
	}
	return nil
}

func getRuleStatus(ctx *cli.Context) error {
	chainId := ctx.String("id")
	ruleAddr := ctx.String("addr")

	receipt, err := invokeBVMContract(ctx, constant.RuleManagerContractAddr.String(), "GetRuleByAddr", pb.String(chainId), pb.String(ruleAddr))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		rule := &ruleMgr.Rule{}
		if err := json.Unmarshal(receipt.Ret, rule); err != nil {
			return fmt.Errorf("unmarshal receipt error: %w", err)
		}
		color.Green("the rule %s is %s", ruleAddr, string(rule.Status))
	} else {
		color.Red("get rule status error: %s\n", string(receipt.Ret))
	}
	return nil
}

func bindRule(ctx *cli.Context) error {
	id := ctx.String("id")
	addr := ctx.String("addr")

	receipt, err := invokeBVMContract(ctx, constant.RuleManagerContractAddr.String(), "BindRule", pb.String(id), pb.String(addr))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		color.Green("proposal id is %s", string(receipt.Ret))
	} else {
		color.Red("bind rule error: %s\n", string(receipt.Ret))
	}
	return nil
}

func unbindRule(ctx *cli.Context) error {
	id := ctx.String("id")
	addr := ctx.String("addr")

	receipt, err := invokeBVMContract(ctx, constant.RuleManagerContractAddr.String(), "UnbindRule", pb.String(id), pb.String(addr))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		color.Green("proposal id is %s", string(receipt.Ret))
	} else {
		color.Red("unbind rule error: %s\n", string(receipt.Ret))
	}
	return nil
}

func freezeRule(ctx *cli.Context) error {
	id := ctx.String("id")
	addr := ctx.String("addr")

	receipt, err := invokeBVMContract(ctx, constant.RuleManagerContractAddr.String(), "FreezeRule", pb.String(id), pb.String(addr))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		color.Green("proposal id is %s", string(receipt.Ret))
	} else {
		color.Red("freeze rule error: %s\n", string(receipt.Ret))
	}
	return nil
}

func activateRule(ctx *cli.Context) error {
	id := ctx.String("id")
	addr := ctx.String("addr")

	receipt, err := invokeBVMContract(ctx, constant.RuleManagerContractAddr.String(), "ActivateRule", pb.String(id), pb.String(addr))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		color.Green("proposal id is %s", string(receipt.Ret))
	} else {
		color.Red("activate rule error: %s\n", string(receipt.Ret))
	}
	return nil
}

func printRule(rules []*ruleMgr.Rule) {
	var table [][]string
	table = append(table, []string{"ChainId", "RuleAddress", "Status"})

	for _, r := range rules {
		table = append(table, []string{
			r.ChainId,
			r.Address,
			string(r.Status),
		})
	}

	PrintTable(table, true)
}
