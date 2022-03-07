package client

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/urfave/cli"
)

const (
	sendTx   = "transaction"
	sendView = "view"
)

func txCMD() cli.Command {
	return cli.Command{
		Name:   "tx",
		Usage:  "Query transaction by transaction hash",
		Action: getTransaction,
	}

}

func getTransaction(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("please input transaction hash")
	}

	hash := ctx.Args().Get(0)

	url := getURL(ctx, "transaction/"+hash)

	data, err := httpGet(ctx, url)
	if err != nil {
		return fmt.Errorf("get transaction %s failed: %w", hash, err)
	}

	fmt.Println(string(data))

	return nil
}

func sendTxOrView(ctx *cli.Context, sendType, toString string, amount *big.Int, txType uint64, keyPath string, vmType uint64, method string, args ...*pb.Arg) ([]byte, error) {

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
		return nil, fmt.Errorf("marshal invoke payload error: %w", err)
	}

	data := &pb.TransactionData{
		Type:    pb.TransactionData_Type(txType),
		Amount:  amount.String(),
		VmType:  pb.TransactionData_VMType(vmType),
		Payload: invokePayloadData,
	}
	payload, err := data.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal transaction data error: %w", err)
	}

	getNonceUrl := getURL(ctx, fmt.Sprintf("pendingNonce/%s", from.String()))

	encodedNonce, err := httpGet(ctx, getNonceUrl)
	if err != nil {
		return nil, fmt.Errorf("httpGet from url %s failed: %w", getNonceUrl, err)
	}

	ret, err := parseResponse(encodedNonce)
	if err != nil {
		return nil, fmt.Errorf("wrong response: %w", err)
	}

	nonce, err := strconv.ParseUint(ret, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse pending nonce :%w", err)
	}

	tx := &pb.BxhTransaction{
		From:      from,
		To:        to,
		Timestamp: time.Now().UnixNano(),
		Nonce:     nonce,
		Payload:   payload,
	}

	if err := tx.Sign(key.PrivKey); err != nil {
		return nil, fmt.Errorf("sign tx error: %s", err)
	}

	reqData, err := json.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("marshal tx error: %w", err)
	}

	url := getURL(ctx, sendType)

	resp, err := httpPost(ctx, url, reqData)
	if err != nil {
		return nil, fmt.Errorf("httpPost %s to url %s failed: %w", reqData, url, err)
	}

	return resp, nil
}
