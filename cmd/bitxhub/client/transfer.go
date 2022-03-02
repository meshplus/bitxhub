package client

import (
	"fmt"
	"math/big"

	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/urfave/cli"
)

func transferCMD() cli.Command {
	return cli.Command{
		Name:  "transfer",
		Usage: "Transfer balance from address to address",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "key",
				Usage:    "Specify transfer account private key path",
				Required: true,
			},
			cli.StringFlag{
				Name:     "to",
				Usage:    "Specify transfer target address",
				Required: true,
			},
			cli.StringFlag{
				Name:     "amount",
				Usage:    "Specify transfer amount",
				Value:    "1000000000000000000",
				Required: false,
			},
		},
		Action: transferBalance,
	}
}

func transferBalance(ctx *cli.Context) error {
	toString := ctx.String("to")
	amountStr := ctx.String("amount")
	keyPath := ctx.String("key")
	var txType uint64 = 0

	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return fmt.Errorf("invalid amount")
	}

	if keyPath == "" {
		repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
		if err != nil {
			return fmt.Errorf("pathRootWithDefault error: %w", err)
		}

		keyPath = repo.GetKeyPath(repoRoot)
	}

	resp, err := sendTxOrView(ctx, sendTx, toString, amount, txType, keyPath, 0, "")
	if err != nil {
		return fmt.Errorf("send transaction: %w", err)
	}

	fmt.Println(string(resp))
	return nil
}
