package client

import (
	"fmt"
	"io/ioutil"

	"github.com/tidwall/gjson"

	"github.com/meshplus/bitxhub/internal/constant"
	rpcx "github.com/meshplus/go-bitxhub-client"
	"github.com/urfave/cli"
)

func ruleCMD() cli.Command {
	return cli.Command{
		Name:  "rule",
		Usage: "Command about rule",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "key",
				Usage:    "Specific key.json path",
				Required: true,
			},
		},
		Subcommands: cli.Commands{
			{
				Name:  "deploy",
				Usage: "Deploy validation rule",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "path",
						Usage:    "Specific rule path",
						Required: true,
					},
				},
				Action: deployRule,
			},
		},
	}
}

func deployRule(ctx *cli.Context) error {
	keyPath := ctx.GlobalString("key")
	grpcAddr := ctx.GlobalString("grpc")
	rulePath := ctx.String("path")

	contract, err := ioutil.ReadFile(rulePath)
	if err != nil {
		return err
	}

	client, err := loadClient(keyPath, grpcAddr)
	if err != nil {
		return fmt.Errorf("load client: %w", err)
	}

	data, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return err
	}

	address := gjson.Get(string(data), "address")

	contractAddr, err := client.DeployContract(contract)
	if err != nil {
		return fmt.Errorf("deploy rule: %w", err)
	}

	_, err = client.InvokeBVMContract(
		constant.RuleManagerContractAddr.Address(),
		"RegisterRule",
		rpcx.String(address.String()),
		rpcx.String(contractAddr.String()))
	if err != nil {
		return fmt.Errorf("register rule")
	}

	fmt.Println("Deploy rule to bitxhub successfully")

	return nil
}
