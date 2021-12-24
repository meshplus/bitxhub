package client

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/urfave/cli"
)

func didMgrCMD() cli.Command {
	return cli.Command{
		Name:  "did",
		Usage: "did manage command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "audit",
				Usage: "audit did method apply",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "method",
						Usage:    "Specify method name",
						Required: true,
					},
					cli.StringFlag{
						Name:     "status",
						Usage:    "audit status, approve or reject",
						Required: true,
					},
				},
				Action: auditMethod,
			},
			cli.Command{
				Name:   "init",
				Usage:  "init did method registry",
				Action: initRegistry,
			},
		},
	}
}

func auditMethod(ctx *cli.Context) error {
	method := ctx.String("method")
	status := ctx.String("status")

	methodDID := "did:bitxhub:" + method + ":."

	var res int32

	if status == "approve" {
		res = 1
	} else if status == "reject" {
		res = -1
	} else {
		return fmt.Errorf("status should be \" approve \" or \" reject \"")
	}

	caller, err := getCaller(ctx)
	if err != nil {
		return fmt.Errorf("construct caller failed: %w", err)
	}

	receipt, err := invokeBVMContract(ctx, constant.MethodRegistryContractAddr.String(), "AuditApply", pb.String(caller), pb.String(methodDID), pb.Int32(res), pb.Bytes([]byte("")))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when audit apply for %s: %w", method, err)
	}
	if receipt.IsSuccess() {
		color.Green("audit apply for %s successfully", method)
	} else {
		color.Red("audit apply for %s failed: %s\n", method, string(receipt.Ret))
	}
	return nil
}

func initRegistry(ctx *cli.Context) error {
	caller, err := getCaller(ctx)
	if err != nil {
		return fmt.Errorf("construct caller failed: %w", err)
	}
	receipt, err := invokeBVMContract(ctx, constant.MethodRegistryContractAddr.String(), "Init", pb.String(caller))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when init did method registry: %w", err)
	}
	if receipt.IsSuccess() {
		color.Green("init did method registry successfully")
	} else {
		color.Red("init did method registry failed: %s", string(receipt.Ret))
	}
	return nil
}

func getCaller(ctx *cli.Context) (string, error) {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return "", fmt.Errorf("pathRootWithDefault error: %w", err)
	}

	keyPath := repo.GetKeyPath(repoRoot)
	key, err := repo.LoadKey(keyPath)
	if err != nil {
		return "", fmt.Errorf("wrong key: %w", err)
	}

	caller := "did:bitxhub:relayroot:" + key.Address

	return caller, nil

}
