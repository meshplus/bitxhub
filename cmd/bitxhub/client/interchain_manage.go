package client

import (
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/utils"
	"github.com/urfave/cli"
)

func interchainMgrCMD() cli.Command {
	return cli.Command{
		Name:  "interchain",
		Usage: "Interchain manage command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "counter",
				Usage: "Query interchain counter by full ibtp id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify full ibtp id",
						Required: true,
					},
				},
				Action: getInterchainCounter,
			},
			cli.Command{
				Name:  "tx",
				Usage: "Query tx hash by full ibtp id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify full ibtp id",
						Required: true,
					},
					cli.BoolFlag{
						Name:     "is_req",
						Usage:    "Specify interchain or receipt",
						Required: false,
					},
				},
				Action: getIbtpTxHash,
			},
			cli.Command{
				Name:  "status",
				Usage: "Query tx status by full ibtp id",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify full ibtp id",
						Required: true,
					},
				},
				Action: getIbtpStatus,
			},
		},
	}
}

func getInterchainCounter(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContractBySendView(ctx, constant.InterchainContractAddr.String(), "GetInterchain", pb.String(id))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when get interchainCounter by id %s: %w", id, err)
	}

	if receipt.IsSuccess() {
		interchain := &pb.Interchain{}
		if err := interchain.Unmarshal(receipt.Ret); err != nil {
			return fmt.Errorf("unmarshal receipt error: %w", err)
		}
		utils.PrettyPrint(interchain)
	} else {
		color.Red("get interchain counter error: %s\n", string(receipt.Ret))
	}
	return nil
}

func getIbtpTxHash(ctx *cli.Context) error {
	id := ctx.String("id")
	isReq := ctx.Bool("is_req")

	receipt, err := invokeBVMContractBySendView(ctx, constant.InterchainContractAddr.String(), "GetIBTPByID", pb.String(id), pb.Bool(isReq))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when get ibtp tx hash by id %s and is_req %v: %w", id, isReq, err)
	}

	if receipt.IsSuccess() {
		hash := &types.Hash{}
		hash.SetBytes(receipt.Ret)
		fmt.Println(hash.String())
	} else {
		color.Red("get interchain counter error: %s\n", string(receipt.Ret))
	}
	return nil
}

func getIbtpStatus(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContractBySendView(ctx, constant.TransactionMgrContractAddr.String(), "GetStatus", pb.String(id))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when get ibtp status by id %s: %w", id, err)
	}

	if receipt.IsSuccess() {
		status := string(receipt.Ret)
		res, err := strconv.Atoi(status)
		if err != nil {
			return err
		}
		fmt.Println("status is:", pb.TransactionStatus(res))
	} else {
		color.Red("get interchain counter error: %s\n", string(receipt.Ret))
	}
	return nil
}
