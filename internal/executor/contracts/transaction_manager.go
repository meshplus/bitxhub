package contracts

import (
	"fmt"
	"strconv"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/pb"
)

const PREFIX = "tx-"

type TransactionManager struct {
	boltvm.Stub
}

type TransactionInfo struct {
	GlobalState pb.TransactionStatus
	ChildTxInfo map[string]pb.TransactionStatus
}

func (t *TransactionManager) BeginMultiTXs(globalId string, childTxIds ...string) *boltvm.Response {
	if t.Has(t.txInfoKey(globalId)) {
		return boltvm.Error("Transaction id already exists")
	}

	txInfo := TransactionInfo{
		GlobalState: pb.TransactionStatus_BEGIN,
		ChildTxInfo: make(map[string]pb.TransactionStatus),
	}

	for _, childTxId := range childTxIds {
		txInfo.ChildTxInfo[childTxId] = pb.TransactionStatus_BEGIN
		t.Set(childTxId, []byte(globalId))
	}

	t.SetObject(t.globalTxInfoKey(globalId), txInfo)

	return boltvm.Success(nil)
}

func (t *TransactionManager) Begin(txId string) *boltvm.Response {
	t.AddObject(t.txInfoKey(txId), pb.TransactionStatus_BEGIN)

	return boltvm.Success(nil)
}

func (t *TransactionManager) Report(txId string, result int32) *boltvm.Response {
	var status pb.TransactionStatus
	ok := t.GetObject(t.txInfoKey(txId), &status)
	if ok {
		if status != pb.TransactionStatus_BEGIN {
			return boltvm.Error(fmt.Sprintf("transaction with Id %s is finished", txId))
		}

		if result == 0 {
			t.SetObject(t.txInfoKey(txId), pb.TransactionStatus_SUCCESS)
		} else {
			t.SetObject(t.txInfoKey(txId), pb.TransactionStatus_FAILURE)
		}
	} else {
		ok, val := t.Get(txId)
		if !ok {
			return boltvm.Error(fmt.Sprintf("cannot get global id of child tx id %s", txId))
		}

		globalId := string(val)
		txInfo := TransactionInfo{}
		if !t.GetObject(t.globalTxInfoKey(globalId), &txInfo) {
			return boltvm.Error(fmt.Sprintf("transaction global id %s does not exist", globalId))
		}

		if txInfo.GlobalState != pb.TransactionStatus_BEGIN {
			return boltvm.Error(fmt.Sprintf("transaction with global Id %s is finished", globalId))
		}

		status, ok := txInfo.ChildTxInfo[txId]
		if !ok {
			return boltvm.Error(fmt.Sprintf("%s is not in transaction %s, %v", txId, globalId, txInfo))
		}

		if status != pb.TransactionStatus_BEGIN {
			return boltvm.Error(fmt.Sprintf("%s has already reported result", txId))
		}

		if result == 0 {
			txInfo.ChildTxInfo[txId] = pb.TransactionStatus_SUCCESS
			count := 0
			for _, res := range txInfo.ChildTxInfo {
				if res != pb.TransactionStatus_SUCCESS {
					break
				}
				count++
			}

			if count == len(txInfo.ChildTxInfo) {
				txInfo.GlobalState = pb.TransactionStatus_SUCCESS
			}
		} else {
			txInfo.ChildTxInfo[txId] = pb.TransactionStatus_FAILURE
			txInfo.GlobalState = pb.TransactionStatus_FAILURE
		}

		t.SetObject(t.globalTxInfoKey(globalId), txInfo)
	}

	return boltvm.Success(nil)
}

func (t *TransactionManager) GetStatus(txId string) *boltvm.Response {
	var status pb.TransactionStatus
	ok := t.GetObject(t.txInfoKey(txId), &status)
	if ok {
		return boltvm.Success([]byte(strconv.Itoa(int(status))))
	}

	txInfo := TransactionInfo{}
	ok = t.GetObject(t.globalTxInfoKey(txId), &txInfo)
	if ok {
		return boltvm.Success([]byte(strconv.Itoa(int(txInfo.GlobalState))))
	}

	ok, val := t.Get(txId)
	if !ok {
		return boltvm.Error(fmt.Sprintf("cannot get global id of child tx id %s", txId))
	}

	globalId := string(val)
	txInfo = TransactionInfo{}
	if !t.GetObject(t.globalTxInfoKey(globalId), &txInfo) {
		return boltvm.Error(fmt.Sprintf("transaction info for global id %s does not exist", globalId))
	}

	return boltvm.Success([]byte(strconv.Itoa(int(txInfo.GlobalState))))
}

func (t *TransactionManager) txInfoKey(id string) string {
	return fmt.Sprintf("%s-%s", PREFIX, id)
}

func (t *TransactionManager) globalTxInfoKey(id string) string {
	return fmt.Sprintf("global-%s-%s", PREFIX, id)
}
