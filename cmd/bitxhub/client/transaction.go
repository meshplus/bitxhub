package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/urfave/cli"
)

func txCMD() cli.Command {
	return cli.Command{
		Name:  "tx",
		Usage: "Transaction manipulation",
		Subcommands: []cli.Command{
			{
				Name:   "get",
				Usage:  "Query transaction",
				Action: getTransaction,
			},
			{
				Name:  "send",
				Usage: "Send transaction",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "key",
						Usage: "Private key path",
					},
					cli.StringFlag{
						Name:  "to",
						Usage: "Target address",
					},
					cli.Uint64Flag{
						Name:  "amount",
						Usage: "Transfer amount",
					},
					cli.Uint64Flag{
						Name:  "type",
						Usage: "Transaction type",
					},
				},
				Action: sendTransaction,
			},
		},
	}

}

func getTransaction(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("please input transaction hash")
	}

	hash := ctx.Args().Get(0)

	url, err := getURL(ctx, "transaction/"+hash)
	if err != nil {
		return err
	}

	data, err := httpGet(url)
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}

func sendTransaction(ctx *cli.Context) error {
	toString := ctx.String("to")
	amount := ctx.Uint64("amount")
	txType := ctx.Uint64("type")
	keyPath := ctx.String("key")

	if keyPath == "" {
		repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
		if err != nil {
			return err
		}

		keyPath = repo.GetKeyPath(repoRoot)
	}

	key, err := repo.LoadKey(keyPath)
	if err != nil {
		return fmt.Errorf("wrong key: %w", err)
	}

	from, err := key.PrivKey.PublicKey().Address()
	if err != nil {
		return fmt.Errorf("wrong private key: %w", err)
	}

	to := types.String2Address(toString)

	req := pb.SendTransactionRequest{
		From:      from,
		To:        to,
		Timestamp: time.Now().UnixNano(),
		Data: &pb.TransactionData{
			Type:   pb.TransactionData_Type(txType),
			Amount: amount,
		},
		Signature: nil,
	}

	tx := &pb.Transaction{
		From:      from,
		To:        to,
		Timestamp: req.Timestamp,
		Nonce:     uint64(req.Nonce),
		Data:      req.Data,
	}

	if err := tx.Sign(key.PrivKey); err != nil {
		return err
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url, err := getURL(ctx, "transaction")
	if err != nil {
		return err
	}

	resp, err := httpPost(url, reqData)
	if err != nil {
		return err
	}

	fmt.Println(string(resp))

	return nil
}
