package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	crypto2 "github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
)

func GetIBTPSign(ledger *ledger.Ledger, id string, isReq bool, privKey crypto2.PrivateKey) (string, []byte, error) {
	ibtp, err := getIBTP(ledger, id, isReq)
	if err != nil {
		return "", nil, fmt.Errorf("get ibtp %s isReq %v: %w", id, isReq, err)
	}

	txStatus, err := getTxStatus(ledger, id)
	if err != nil {
		return "", nil, fmt.Errorf("get tx status of ibtp %s isReq %v: %w", id, isReq, err)
	}

	hash, err := EncodePackedAndHash(ibtp, txStatus)
	if err != nil {
		return "", nil, fmt.Errorf("encode packed and hash for ibtp %s isReq %v: %w", id, isReq, err)
	}

	sign, err := privKey.Sign(hash)
	if err != nil {
		return "", nil, fmt.Errorf("bitxhub sign ibtp %s isReq %v: %w", id, isReq, err)
	}

	addr, err := privKey.PublicKey().Address()
	if err != nil {
		return "", nil, err
	}

	return addr.String(), sign, nil
}

func getIBTP(ledger *ledger.Ledger, id string, isReq bool) (*pb.IBTP, error) {
	key := contracts.IndexMapKey(id)
	if !isReq {
		key = contracts.IndexReceiptMapKey(id)
	}

	ok, val := ledger.Copy().GetState(constant.InterchainContractAddr.Address(), []byte(key))
	if !ok {
		return nil, fmt.Errorf("cannot get the tx hash which contains the IBTP %s", id)
	}

	var hash types.Hash
	if err := json.Unmarshal(val, &hash); err != nil {
		return nil, err
	}

	tx, err := ledger.GetTransaction(&hash)
	if err != nil {
		return nil, err
	}

	return tx.GetIBTP(), nil
}

// TODO: support global status
func getTxStatus(ledger *ledger.Ledger, id string) (pb.TransactionStatus, error) {
	ok, val := ledger.Copy().GetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.TxInfoKey(id)))
	if !ok {
		return 0, fmt.Errorf("no tx status found for ibtp %s", id)
	}
	var record pb.TransactionRecord
	if err := json.Unmarshal(val, &record); err != nil {
		return 0, err
	}

	return record.Status, nil
}

func EncodePackedAndHash(ibtp *pb.IBTP, txStatus pb.TransactionStatus) ([]byte, error) {
	var (
		data []byte
		pd   pb.Payload
	)

	data = append(data, []byte(ibtp.From)...)
	data = append(data, []byte(ibtp.To)...)
	data = append(data, uint64ToBytesInBigEndian(ibtp.Index)...)
	data = append(data, uint64ToBytesInBigEndian(uint64(ibtp.Type))...)

	if err := pd.Unmarshal(ibtp.Payload); err != nil {
		return nil, err
	}

	data = append(data, pd.Hash...)
	data = append(data, uint64ToBytesInBigEndian(uint64(txStatus))...)

	hash := crypto.Keccak256(data)

	return hash[:], nil
}

func uint64ToBytesInBigEndian(i uint64) []byte {
	bytes := make([]byte, 8)

	binary.BigEndian.PutUint64(bytes, i)

	return bytes
}