package client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli"
)

func nodeMgrCMD() cli.Command {
	return cli.Command{
		Name:  "node",
		Usage: "Node manage command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:   "all",
				Usage:  "Query all nodes info",
				Action: allNode,
			},
			cli.Command{
				Name:  "status",
				Usage: "Query node status by node account",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "account",
						Usage:    "Specify node account",
						Required: true,
					},
				},
				Action: getNodeStatusByAccount,
			},
			cli.Command{
				Name:  "register",
				Usage: "Register node",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "account",
						Usage:    "Specify node account",
						Required: true,
					},
					cli.StringFlag{
						Name:     "type",
						Usage:    "Specify node type (vpNode or nvpNode)",
						Value:    "vpNode",
						Required: false,
					},
					cli.StringFlag{
						Name:     "pid",
						Usage:    "Specify vp node pid, only useful for vpNode",
						Required: false,
					},
					cli.Uint64Flag{
						Name:     "id",
						Usage:    "Specify vp node id, only useful for vpNode",
						Required: false,
					},
					cli.StringFlag{
						Name:     "name",
						Usage:    "Specify nvp node name, only useful for nvpNode",
						Required: false,
					},
					cli.StringFlag{
						Name:     "permission",
						Usage:    "Specify nvp node permission, only useful for nvpNode, multiple appchain addresses are separated by commas",
						Required: false,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify register reason",
						Required: false,
					},
				},
				Action: registerNode,
			},
			cli.Command{
				Name:  "update",
				Usage: "Update node info",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "account",
						Usage:    "Specify node account",
						Required: true,
					},
					cli.StringFlag{
						Name:     "name",
						Usage:    "Specify nvp node name, only useful for nvpNode",
						Required: false,
					},
					cli.StringFlag{
						Name:     "permission",
						Usage:    "Specify nvp node permission, only useful for nvpNode, multiple appchain addresses are separated by commas",
						Required: false,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify update reason",
						Required: false,
					},
				},
				Action: updateNode,
			},
			cli.Command{
				Name:  "logout",
				Usage: "Logout node by node account",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "account",
						Usage:    "Specify node account",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify logout reason",
						Required: false,
					},
				},
				Action: logoutNode,
			},
		},
	}
}

func getNodeStatusByAccount(ctx *cli.Context) error {
	account := ctx.String("account")

	receipt, err := invokeBVMContractBySendView(ctx, constant.NodeManagerContractAddr.Address().String(), "GetNode", pb.String(account))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when get node status by account %s: %w", account, err)
	}

	if receipt.IsSuccess() {
		node := &node_mgr.Node{}
		if err := json.Unmarshal(receipt.Ret, node); err != nil {
			return fmt.Errorf("unmarshal receipt error: %w", err)
		}
		color.Green("node %s is %s\n", node.Pid, string(node.Status))
	} else {
		color.Red("get node status error: %s\n", string(receipt.Ret))
	}
	return nil
}

func registerNode(ctx *cli.Context) error {
	account := ctx.String("account")
	typ := ctx.String("type")
	pid := ctx.String("pid")
	vpNodeId := ctx.Uint64("id")
	name := ctx.String("name")
	permisssion := ctx.String("permission")
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.NodeManagerContractAddr.Address().String(), "RegisterNode",
		pb.String(account),
		pb.String(typ),
		pb.String(pid),
		pb.Uint64(vpNodeId),
		pb.String(name),
		pb.String(permisssion),
		pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when register node \" account=%s,typ=%s,pid=%s,vpNodeId=%d,name=%s,permission=%s,reason=%s \": %w",
			account, typ, pid, vpNodeId, name, permisssion, reason, err)
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("register node error: %s\n", string(receipt.Ret))
	}
	return nil
}

func updateNode(ctx *cli.Context) error {
	account := ctx.String("account")
	name := ctx.String("name")
	permisssion := ctx.String("permission")
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.NodeManagerContractAddr.Address().String(), "UpdateNode",
		pb.String(account),
		pb.String(name),
		pb.String(permisssion),
		pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when update node \" account=%s,name=%s, permission=%s,reason=%s \": %w",
			account, name, permisssion, reason, err)
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("update node error: %s\n", string(receipt.Ret))
	}
	return nil
}

func logoutNode(ctx *cli.Context) error {
	account := ctx.String("account")
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.NodeManagerContractAddr.Address().String(), "LogoutNode", pb.String(account), pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when logout node by account %s for %s: %w", account, reason, err)
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("logout node error: %s\n", string(receipt.Ret))
	}
	return nil
}

func allNode(ctx *cli.Context) error {
	receipt, err := invokeBVMContractBySendView(ctx, constant.NodeManagerContractAddr.Address().String(), "Nodes")
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when get all node info: %w", err)
	}

	if receipt.IsSuccess() {
		nodes := make([]*node_mgr.Node, 0)
		if receipt.Ret != nil {
			if err := json.Unmarshal(receipt.Ret, &nodes); err != nil {
				return fmt.Errorf("unmarshal nodes error: %w", err)
			}
		}
		printNode(nodes)
	} else {
		color.Red("query node info error: %s\n", string(receipt.Ret))
	}
	return nil
}

func printNode(nodes []*node_mgr.Node) {
	var table [][]string
	table = append(table, []string{"Account", "Type", "Pid", "VpNodeId", "Name", "Permission", "Status", "AuditAdminAddr"})

	for _, n := range nodes {
		permits := []string{}
		for addr, _ := range n.Permissions {
			permits = append(permits, addr)
		}
		permitStr := strings.Join(permits, ",")
		table = append(table, []string{
			n.Account,
			string(n.NodeType),
			n.Pid,
			strconv.Itoa(int(n.VPNodeId)),
			n.Name,
			permitStr,
			string(n.Status),
			n.AuditAdminAddr,
		})
	}

	PrintTable(table, true)
}
