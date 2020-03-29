package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	rpcx "github.com/meshplus/go-bitxhub-client"
	"github.com/urfave/cli"
)

func appchainCMD() cli.Command {
	return cli.Command{
		Name:  "appchain",
		Usage: "Command about appchain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "key",
				Usage:    "Specific key.json path",
				Required: true,
			},
		},
		Subcommands: []cli.Command{
			{
				Name:  "register",
				Usage: "Register appchain in bitxhub",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "name",
						Usage:    "Specific appchain name",
						Required: true,
					},
					cli.StringFlag{
						Name:     "type",
						Usage:    "Specific appchain type",
						Required: true,
					},
					cli.StringFlag{
						Name:     "desc",
						Usage:    "Specific appchain description",
						Required: true,
					},
					cli.StringFlag{
						Name:     "version",
						Usage:    "Specific appchain version",
						Required: true,
					},
					cli.StringFlag{
						Name:     "validators",
						Usage:    "Specific appchain validators path",
						Required: true,
					},
				},
				Action: registerAppchain,
			},
			{
				Name:  "audit",
				Usage: "Audit appchain in bitxhub",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specific appchain id",
						Required: true,
					},
				},
				Action: auditAppchain,
			},
			{
				Name:  "get",
				Usage: "Get appchain info",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specific appchain id",
						Required: true,
					},
				},
				Action: getAppchain,
			},
		},
	}
}

func registerAppchain(ctx *cli.Context) error {
	keyPath := ctx.GlobalString("key")
	grpcAddr := ctx.GlobalString("grpc")
	name := ctx.String("name")
	typ := ctx.String("type")
	desc := ctx.String("desc")
	version := ctx.String("version")
	validatorsPath := ctx.String("validators")

	data, err := ioutil.ReadFile(validatorsPath)
	if err != nil {
		return fmt.Errorf("read validators file: %w", err)
	}

	client, err := loadClient(keyPath, grpcAddr)
	if err != nil {
		return fmt.Errorf("load client: %w", err)
	}

	receipt, err := client.InvokeBVMContract(
		rpcx.InterchainContractAddr,
		"Register", rpcx.String(string(data)),
		rpcx.Int32(1),
		rpcx.String(typ),
		rpcx.String(name),
		rpcx.String(desc),
		rpcx.String(version),
	)
	if err != nil {
		return fmt.Errorf("invoke bvm contract: %w", err)
	}

	if !receipt.IsSuccess() {
		return fmt.Errorf("invoke register: %s", receipt.Ret)
	}

	appchain := &rpcx.Appchain{}
	if err := json.Unmarshal(receipt.Ret, appchain); err != nil {
		return err
	}

	fmt.Printf("appchain register successfully, id is %s\n", appchain.ID)

	return nil
}

func auditAppchain(ctx *cli.Context) error {
	keyPath := ctx.GlobalString("key")
	grpcAddr := ctx.GlobalString("grpc")
	id := ctx.String("id")

	client, err := loadClient(keyPath, grpcAddr)
	if err != nil {
		return fmt.Errorf("load client: %w", err)
	}

	receipt, err := client.InvokeBVMContract(
		rpcx.InterchainContractAddr,
		"Audit",
		rpcx.String(id),
		rpcx.Int32(1),
		rpcx.String("Audit passed"),
	)

	if err != nil {
		return err
	}

	if !receipt.IsSuccess() {
		return fmt.Errorf("invoke audit: %s", receipt.Ret)
	}

	fmt.Printf("audit appchain %s successfully\n", id)

	return nil
}

func getAppchain(ctx *cli.Context) error {
	keyPath := ctx.GlobalString("key")
	grpcAddr := ctx.GlobalString("grpc")

	client, err := loadClient(keyPath, grpcAddr)
	if err != nil {
		return fmt.Errorf("load client: %w", err)
	}

	receipt, err := client.InvokeBVMContract(
		rpcx.InterchainContractAddr,
		"Appchain",
	)

	if err != nil {
		return err
	}

	if !receipt.IsSuccess() {
		return fmt.Errorf("get appchain: %s", receipt.Ret)
	}

	fmt.Println(string(receipt.Ret))

	return nil
}
