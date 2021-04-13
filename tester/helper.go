package tester

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
)

func genBVMContractTransaction(privateKey crypto.PrivateKey, nonce uint64, address *types.Address, method string, args ...*pb.Arg) (*pb.Transaction, error) {
	return genContractTransaction(pb.TransactionData_BVM, privateKey, nonce, address, method, args...)
}

func genXVMContractTransaction(privateKey crypto.PrivateKey, nonce uint64, address *types.Address, method string, args ...*pb.Arg) (*pb.Transaction, error) {
	return genContractTransaction(pb.TransactionData_XVM, privateKey, nonce, address, method, args...)
}

func genIBTPTransaction(privateKey crypto.PrivateKey, ibtp *pb.IBTP, nonce uint64) (*pb.Transaction, error) {
	from, err := privateKey.PublicKey().Address()
	if err != nil {
		return nil, err
	}

	ibtpd, err := ibtp.Marshal()
	if err != nil {
		return nil, err
	}

	pl := &pb.InvokePayload{
		Method: "HandleIBTP",
		Args:   []*pb.Arg{pb.Bytes(ibtpd)},
	}

	data, err := pl.Marshal()
	if err != nil {
		return nil, err
	}

	td := &pb.TransactionData{
		Type:    pb.TransactionData_INVOKE,
		VmType:  pb.TransactionData_BVM,
		Payload: data,
	}

	payload, err := td.Marshal()
	if err != nil {
		return nil, err
	}

	tx := &pb.Transaction{
		From:      from,
		To:        constant.InterchainContractAddr.Address(),
		Payload:   payload,
		Timestamp: time.Now().UnixNano(),
		Nonce:     nonce,
		IBTP:      ibtp,
	}

	if err := tx.Sign(privateKey); err != nil {
		return nil, fmt.Errorf("tx sign: %w", err)
	}

	tx.TransactionHash = tx.Hash()

	return tx, nil
}

func invokeBVMContract(api api.CoreAPI, privateKey crypto.PrivateKey, nonce uint64, address *types.Address, method string, args ...*pb.Arg) (*pb.Receipt, error) {
	tx, err := genBVMContractTransaction(privateKey, nonce, address, method, args...)
	if err != nil {
		return nil, err
	}

	return sendTransactionWithReceipt(api, tx)
}

func sendTransactionWithReceipt(api api.CoreAPI, tx *pb.Transaction) (*pb.Receipt, error) {
	err := api.Broker().HandleTransaction(tx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("get receipt timeout")
		default:
			time.Sleep(200 * time.Millisecond)
			receipt, err := api.Broker().GetReceipt(tx.TransactionHash)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					continue
				}

				return nil, err
			}

			return receipt, nil
		}
	}

}

func genContractTransaction(vmType pb.TransactionData_VMType, privateKey crypto.PrivateKey, nonce uint64, address *types.Address, method string, args ...*pb.Arg) (*pb.Transaction, error) {
	from, err := privateKey.PublicKey().Address()
	if err != nil {
		return nil, err
	}

	pl := &pb.InvokePayload{
		Method: method,
		Args:   args[:],
	}

	data, err := pl.Marshal()
	if err != nil {
		return nil, err
	}

	td := &pb.TransactionData{
		Type:    pb.TransactionData_INVOKE,
		VmType:  vmType,
		Payload: data,
	}

	payload, err := td.Marshal()
	if err != nil {
		return nil, err
	}

	tx := &pb.Transaction{
		From:      from,
		To:        address,
		Payload:   payload,
		Timestamp: time.Now().UnixNano(),
		Nonce:     nonce,
	}

	if err := tx.Sign(privateKey); err != nil {
		return nil, fmt.Errorf("tx sign: %w", err)
	}

	tx.TransactionHash = tx.Hash()

	return tx, nil
}

func deployContract(api api.CoreAPI, privateKey crypto.PrivateKey, nonce uint64, contract []byte) (*types.Address, error) {
	from, err := privateKey.PublicKey().Address()
	if err != nil {
		return nil, err
	}

	td := &pb.TransactionData{
		Type:    pb.TransactionData_INVOKE,
		VmType:  pb.TransactionData_XVM,
		Payload: contract,
	}

	payload, err := td.Marshal()
	if err != nil {
		return nil, err
	}

	tx := &pb.Transaction{
		From:      from,
		Payload:   payload,
		Timestamp: time.Now().UnixNano(),
		Nonce:     nonce,
	}

	tx.TransactionHash = tx.Hash()

	if err := tx.Sign(privateKey); err != nil {
		return nil, fmt.Errorf("tx sign: %w", err)
	}

	receipt, err := sendTransactionWithReceipt(api, tx)
	if err != nil {
		return nil, err
	}

	ret := types.NewAddress(receipt.GetRet())

	return ret, nil
}
