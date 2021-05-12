package client

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/urfave/cli"
)

const relaychainDIDPrefix = "did:bitxhub:relayroot:"

func didCMD() cli.Command {
	return cli.Command{
		Name:  "did",
		Usage: "did command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:   "init",
				Usage:  "init method-registry contract",
				Action: initAdminDID,
			},
			cli.Command{
				Name:  "addAdmin",
				Usage: "add admin role for method-registry contract",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name: "addr",
						Usage: "Specify node address derived from node public key " +
							"to add method-registry contract",
						Required: true,
					},
				},
				Action: addAdmin,
			},
		},
	}
}

func initAdminDID(ctx *cli.Context) error {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return err
	}
	keyPath := repo.GetKeyPath(repoRoot)
	key, err := repo.LoadKey(keyPath)
	if err != nil {
		return fmt.Errorf("wrong key: %w", err)
	}

	from, err := key.PrivKey.PublicKey().Address()
	if err != nil {
		return fmt.Errorf("wrong private key: %w", err)
	}
	relayAdminDID := genRelaychainDID(from.String())
	// init method registry with this admin key
	receipt, err := invokeBVMContract(ctx,
		constant.MethodRegistryContractAddr.String(),
		"Init", pb.String(relayAdminDID),
	)
	if err != nil {
		return fmt.Errorf("invoke bvm contract: %w", err)
	}
	if !receipt.IsSuccess() {
		return fmt.Errorf("method registery init faild: %s", string(receipt.Ret))
	}
	// init did registry with this admin key
	receipt, err = invokeBVMContract(ctx,
		constant.DIDRegistryContractAddr.String(),
		"Init", pb.String(relayAdminDID),
	)
	if err != nil {
		return fmt.Errorf("invoke bvm contract: %w", err)
	}
	if !receipt.IsSuccess() {
		return fmt.Errorf("did registery init faild: %s", string(receipt.Ret))
	}
	return nil
}

func addAdmin(ctx *cli.Context) error {
	addr := ctx.String("addr")

	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return err
	}
	keyPath := repo.GetKeyPath(repoRoot)
	key, err := repo.LoadKey(keyPath)
	if err != nil {
		return fmt.Errorf("wrong key: %w", err)
	}

	from, err := key.PrivKey.PublicKey().Address()
	if err != nil {
		return fmt.Errorf("wrong private key: %w", err)
	}

	adminDID := genRelaychainDID(from.String())
	toAddAdminDID := genRelaychainDID(addr)
	receipt, err := invokeBVMContract(ctx, constant.MethodRegistryContractAddr.String(), "AddAdmin",
		pb.String(adminDID), pb.String(toAddAdminDID))
	if err != nil {
		return err
	}

	if receipt.IsSuccess() {
		color.Green("Add admin for method-registry successfully!\n")
	} else {
		color.Red("Add admin for method-registry error: %s\n", string(receipt.Ret))
	}
	return nil
}

func genRelaychainDID(addr string) string {
	return relaychainDIDPrefix + addr
}
