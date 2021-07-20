package client

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli"
)

func roleMgrCND() cli.Command {
	return cli.Command{
		Name:  "role",
		Usage: "role manage command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "status",
				Usage: "query role status by role id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "role id(address)",
						Required: true,
					},
				},
				Action: getRoleStatusById,
			},
			cli.Command{
				Name:  "register",
				Usage: "register role",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "address",
						Usage:    "role address(id)",
						Required: true,
					},
					cli.StringFlag{
						Name:  "type",
						Usage: "role type, one of governanceAdmin or auditAdmin",
						Value: string(contracts.GovernanceAdmin),
					},
					cli.StringFlag{
						Name:     "nodePid",
						Usage:    "node pid for auditAdmin, only useful for auditAdmin",
						Required: false,
					},
				},
				Action: registerRole,
			},
			cli.Command{
				Name:  "update",
				Usage: "update node for auditAdmin",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "auditAdmin id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "nodePid",
						Usage:    "node pid",
						Required: true,
					},
				},
				Action: updateRole,
			},
			cli.Command{
				Name:  "freeze",
				Usage: "freeze role by role id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "role id",
						Required: true,
					},
				},
				Action: freezeRole,
			},
			cli.Command{
				Name:  "activate",
				Usage: "activate role by role id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "role id",
						Required: true,
					},
				},
				Action: activateRole,
			},
			cli.Command{
				Name:  "logout",
				Usage: "logout role by role id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "role pid",
						Required: true,
					},
				},
				Action: logoutRole,
			},
			cli.Command{
				Name:  "all",
				Usage: "query all roles info",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "type",
						Usage:    "role type",
						Value:    string(contracts.GovernanceAdmin),
						Required: false,
					},
				},
				Action: allRole,
			},
		},
	}
}

func getRoleStatusById(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContractBySendView(ctx, constant.RoleContractAddr.String(), "GetRoleById", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		role := &contracts.Role{}
		if err := json.Unmarshal(receipt.Ret, role); err != nil {
			return fmt.Errorf("unmarshal receipt error: %v", err)
		}
		color.Green("role %d is %s\n", role.ID, string(role.Status))
	} else {
		color.Red("get role status error: %s\n", string(receipt.Ret))
	}
	return nil
}

func registerRole(ctx *cli.Context) error {
	addr := ctx.String("address")
	typ := ctx.String("type")
	nodePid := ctx.String("nodePid")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.String(), "RegisterRole", pb.String(addr), pb.String(typ), pb.String(nodePid))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("register role error: %s\n", string(receipt.Ret))
	}
	return nil
}

func updateRole(ctx *cli.Context) error {
	id := ctx.String("id")
	nodePid := ctx.String("nodePid")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.String(), "UpdateAuditAdminNode", pb.String(id), pb.String(nodePid))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("update auditAdmin node error: %s\n", string(receipt.Ret))
	}
	return nil
}

func freezeRole(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.String(), "FreezeRole", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("freeze role error: %s\n", string(receipt.Ret))
	}
	return nil
}

func activateRole(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.String(), "ActivateRole", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("activate role error: %s\n", string(receipt.Ret))
	}
	return nil
}

func logoutRole(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.String(), "LogoutRole", pb.String(id))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("logout role error: %s\n", string(receipt.Ret))
	}
	return nil
}

func allRole(ctx *cli.Context) error {
	typ := ctx.String("type")

	var ret *pb.Receipt
	switch typ {
	case string(contracts.GovernanceAdmin):
		receipt, err := invokeBVMContractBySendView(ctx, constant.RoleContractAddr.String(), "GetAdminRoles")
		if err != nil {
			return err
		}
		ret = receipt
	case string(contracts.AuditAdmin):
		receipt, err := invokeBVMContractBySendView(ctx, constant.RoleContractAddr.String(), "GetAuditAdminRoles")
		if err != nil {
			return err
		}
		ret = receipt
	default:
		return fmt.Errorf("illegal role type")
	}

	if ret.IsSuccess() {
		roles := make([]*contracts.Role, 0)
		if ret.Ret != nil {
			if err := json.Unmarshal(ret.Ret, &roles); err != nil {
				return fmt.Errorf("unmarshal roles error: %v", err)
			}
		}
		printRole(roles)
	} else {
		color.Red("query role info error: %s\n", string(ret.Ret))
	}
	return nil
}

func printRole(roles []*contracts.Role) {
	var table [][]string
	table = append(table, []string{"RoleId", "type", "Weight", "NodePid", "Status"})

	for _, r := range roles {
		table = append(table, []string{
			r.ID,
			string(r.RoleType),
			strconv.Itoa(int(r.Weight)),
			r.NodePid,
			string(r.Status),
		})
	}

	PrintTable(table, true)
}
