package client

import (
	"encoding/json"
	"fmt"
	"strconv"

	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"

	"github.com/fatih/color"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli"
)

func nodeMgrCND() cli.Command {
	return cli.Command{
		Name:  "node",
		Usage: "node manage command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "status",
				Usage: "query node status by node pid",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "pid",
						Usage:    "node pid",
						Required: true,
					},
				},
				Action: getNodeStatusByPid,
			},
			cli.Command{
				Name:  "register",
				Usage: "register node",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "pid",
						Usage:    "node pid",
						Required: true,
					},
					cli.Uint64Flag{
						Name:     "id",
						Usage:    "vp node id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "account",
						Usage:    "node account",
						Required: true,
					},
					cli.StringFlag{
						Name:     "type",
						Usage:    "node type (vpNode or nvpNode), currently only VPNode is supported",
						Value:    "vpNode",
						Required: false,
					},
				},
				Action: registerNode,
			},
			cli.Command{
				Name:  "logout",
				Usage: "logout node by node pid",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "pid",
						Usage:    "node pid",
						Required: true,
					},
				},
				Action: logoutNode,
			},
			cli.Command{
				Name:  "all",
				Usage: "query all nodes info",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "type",
						Usage:    "node type",
						Value:    string(node_mgr.VPNode),
						Required: false,
					},
				},
				Action: allNode,
			},
		},
	}
}

func getNodeStatusByPid(ctx *cli.Context) error {
	pid := ctx.String("pid")

	receipt, err := invokeBVMContractBySendView(ctx, constant.NodeManagerContractAddr.String(), "GetNode", pb.String(pid))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		node := &node_mgr.Node{}
		if err := json.Unmarshal(receipt.Ret, node); err != nil {
			return fmt.Errorf("unmarshal receipt error: %w", err)
		}
		color.Green("node %d is %s\n", node.Pid, string(node.Status))
	} else {
		color.Red("get node status error: %s\n", string(receipt.Ret))
	}
	return nil
}

func registerNode(ctx *cli.Context) error {
	pid := ctx.String("pid")
	vpNodeId := ctx.Uint64("id")
	account := ctx.String("account")
	typ := ctx.String("type")

	receipt, err := invokeBVMContract(ctx, constant.NodeManagerContractAddr.String(), "RegisterNode", pb.String(pid), pb.Uint64(vpNodeId), pb.String(account), pb.String(typ))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
		color.Green("proposal id is %s\n", proposalId)
	} else {
		color.Red("register node error: %s\n", string(receipt.Ret))
	}
	return nil
}

func logoutNode(ctx *cli.Context) error {
	pid := ctx.String("pid")

	receipt, err := invokeBVMContract(ctx, constant.NodeManagerContractAddr.String(), "LogoutNode", pb.String(pid))
	if err != nil {
		return err
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
	typ := ctx.String("type")

	receipt, err := invokeBVMContractBySendView(ctx, constant.NodeManagerContractAddr.String(), "Nodes", pb.String(typ))
	if err != nil {
		return err
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
	table = append(table, []string{"NodePid", "type", "VpNodeId", "Account", "Status"})

	for _, n := range nodes {
		table = append(table, []string{
			strconv.Itoa(int(n.VPNodeId)),
			string(n.NodeType),
			n.Pid,
			n.Account,
			string(n.Status),
		})
	}

	PrintTable(table, true)
}
