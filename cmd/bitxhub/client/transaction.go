package client

import (
	"encoding/json"
	"fmt"
	"strconv"
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

	data, err := httpGet(ctx, url)
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

	resp, err := sendTx(ctx, toString, amount, txType, keyPath, 0, "")
	if err != nil {
		return fmt.Errorf("send transaction: %w", err)
	}

	fmt.Println(string(resp))
	return nil
}

func sendTx(ctx *cli.Context, toString string, amount uint64, txType uint64, keyPath string, vmType uint64, method string, args ...*pb.Arg) ([]byte, error) {

	key, err := repo.LoadKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("wrong key: %w", err)
	}

	from, err := key.PrivKey.PublicKey().Address()
	if err != nil {
		return nil, fmt.Errorf("wrong private key: %w", err)
	}

	to := types.NewAddressByStr(toString)

	invokePayload := &pb.InvokePayload{
		Method: method,
		Args:   args,
	}
	invokePayloadData, err := invokePayload.Marshal()
	if err != nil {
		return nil, err
	}

	data := &pb.TransactionData{
		Type:    pb.TransactionData_Type(txType),
		Amount:  amount,
		VmType:  pb.TransactionData_VMType(vmType),
		Payload: invokePayloadData,
	}
	payload, err := data.Marshal()
	if err != nil {
		return nil, err
	}

	getNonceUrl, err := getURL(ctx, fmt.Sprintf("pendingNonce/%s", from.String()))
	if err != nil {
		return nil, err
	}

	encodedNonce, err := httpGet(ctx, getNonceUrl)
	if err != nil {
		return nil, err
	}

	ret, err := parseResponse(encodedNonce)
	if err != nil {
		return nil, err
	}

	nonce, err := strconv.ParseUint(ret, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse pending nonce :%w", err)
	}

	tx := &pb.Transaction{
		From:      from,
		To:        to,
		Timestamp: time.Now().UnixNano(),
		Nonce:     nonce,
		Payload:   payload,
	}

	if err := tx.Sign(key.PrivKey); err != nil {
		return nil, err
	}

	reqData, err := json.Marshal(tx)
	if err != nil {
		return nil, err
	}

	url, err := getURL(ctx, "transaction")
	if err != nil {
		return nil, err
	}

	resp, err := httpPost(ctx, url, reqData)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
