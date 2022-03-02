package client

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli"
)

func roleMgrCMD() cli.Command {
	return cli.Command{
		Name:  "role",
		Usage: "Role manage command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:   "all",
				Usage:  "Query all roles info",
				Action: allRole,
			},
			cli.Command{
				Name:  "status",
				Usage: "Query role status by role id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify role id(address)",
						Required: true,
					},
				},
				Action: getRoleStatusById,
			},
			cli.Command{
				Name:  "register",
				Usage: "Register role",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "address",
						Usage:    "Specify role address(id)",
						Required: true,
					},
					cli.StringFlag{
						Name:  "type",
						Usage: "Specify role type, one of governanceAdmin or auditAdmin",
						Value: string(contracts.GovernanceAdmin),
					},
					cli.StringFlag{
						Name:     "nodeAccount",
						Usage:    "Specify node account for auditAdmin, only useful for auditAdmin",
						Required: false,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify register reason",
						Required: false,
					},
				},
				Action: registerRole,
			},
			cli.Command{
				Name:  "freeze",
				Usage: "Freeze role by role id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify role id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify freeze reason",
						Required: false,
					},
				},
				Action: freezeRole,
			},
			cli.Command{
				Name:  "activate",
				Usage: "Activate role by role id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify role id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify activate reason",
						Required: false,
					},
				},
				Action: activateRole,
			},
			cli.Command{
				Name:  "logout",
				Usage: "Logout role by role id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify role id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify logout reason",
						Required: false,
					},
				},
				Action: logoutRole,
			},
			cli.Command{
				Name:  "bind",
				Usage: "Bind audit role with node",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify role id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "account",
						Usage:    "Specify node account",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify bind reason",
						Required: false,
					},
				},
				Action: bindRole,
			},
		},
	}
}

func getRoleStatusById(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContractBySendView(ctx, constant.RoleContractAddr.Address().String(), "GetRoleInfoById", pb.String(id))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when get role info by ID %s: %w", id, err)
	}

	if receipt.IsSuccess() {
		role := &contracts.Role{}
		if err := json.Unmarshal(receipt.Ret, role); err != nil {
			return fmt.Errorf("unmarshal receipt error: %v", err)
		}
		color.Green("role %s is %s\n", role.ID, string(role.Status))
	} else {
		color.Red("get role status error: %s\n", string(receipt.Ret))
	}
	return nil
}

func registerRole(ctx *cli.Context) error {
	addr := ctx.String("address")
	typ := ctx.String("type")
	nodeAccount := ctx.String("nodeAccount")
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.Address().String(), "RegisterRole", pb.String(addr), pb.String(typ), pb.String(nodeAccount), pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when register role \" addr=%s,typ=%s,nodeAccount=%s,reason=%s \": %w",
			addr, typ, nodeAccount, reason, err)
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
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.Address().String(), "UpdateAuditAdminNode", pb.String(id), pb.String(nodePid), pb.String(reason))
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
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.Address().String(), "FreezeRole", pb.String(id), pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when freeze role %s for %s: %w", id, reason, err)
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
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.Address().String(), "ActivateRole", pb.String(id), pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when activate role %s for %s: %w", id, reason, err)
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
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.Address().String(), "LogoutRole", pb.String(id), pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when logout role %s for %s: %w", id, reason, err)
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("logout role error: %s\n", string(receipt.Ret))
	}
	return nil
}

func bindRole(ctx *cli.Context) error {
	id := ctx.String("id")
	account := ctx.String("account")
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.RoleContractAddr.Address().String(), "BindRole", pb.String(id), pb.String(account), pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when bind role %s with node %s for %s: %w", id, account, reason, err)
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("bind role error: %s\n", string(receipt.Ret))
	}
	return nil
}

func allRole(ctx *cli.Context) error {
	ret, err := invokeBVMContractBySendView(ctx, constant.RoleContractAddr.Address().String(), "GetAllRoles")
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when get all roles: %w", err)
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
	table = append(table, []string{"RoleId", "type", "Status", "NodeAccount", "AppchainID"})

	for _, r := range roles {
		var typ string
		if r.RoleType == contracts.GovernanceAdmin && r.Weight == repo.SuperAdminWeight {
			typ = string(contracts.SuperGovernanceAdmin)
		} else {
			typ = string(r.RoleType)

		}
		table = append(table, []string{
			r.ID,
			typ,
			string(r.Status),
			r.NodeAccount,
			r.AppchainID,
		})
	}

	PrintTable(table, true)
}
