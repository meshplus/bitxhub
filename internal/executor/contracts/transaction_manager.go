package contracts

import (
	"fmt"

	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
)

type TransactionStatus string

const (
	PREFIX                          = "tx-"
	StatusBegin   TransactionStatus = "BEGIN"
	StatusSuccess TransactionStatus = "SUCCESS"
	StatusFail    TransactionStatus = "FAIL"
)

type TransactionManager struct {
	boltvm.Stub
}

type TransactionInfo struct {
	globalState TransactionStatus
	childTxInfo map[string]TransactionStatus
}

func (t *TransactionManager) BeginMultiTXs(globalId string, childTxIds ...string) *boltvm.Response {
	if t.Has(t.txInfoKey(globalId)) {
		return boltvm.Error("Transaction id already exists")
	}

	txInfo := &TransactionInfo{
		globalState: StatusBegin,
		childTxInfo: make(map[string]TransactionStatus),
	}

	for _, childTxId := range childTxIds {
		txInfo.childTxInfo[childTxId] = StatusBegin
		t.Set(childTxId, []byte(globalId))
	}

	t.SetObject(t.txInfoKey(globalId), txInfo)

	return boltvm.Success(nil)
}

func (t *TransactionManager) Begin(txId string) *boltvm.Response {
	if t.Has(t.txInfoKey(txId)) {
		return boltvm.Error("Transaction id already exists")
	}

	t.Set(t.txInfoKey(txId), []byte(StatusBegin))

	return boltvm.Success(nil)
}

func (t *TransactionManager) Report(txId string, result int32) *boltvm.Response {
	ok, val := t.Get(t.txInfoKey(txId))
	if ok {
		status := TransactionStatus(val)
		if status != StatusBegin {
			return boltvm.Error(fmt.Sprintf("transaction with Id %s is finished", txId))
		}

		if result == 0 {
			t.Set(t.txInfoKey(txId), []byte(StatusSuccess))
		} else {
			t.Set(t.txInfoKey(txId), []byte(StatusFail))
		}
	} else {
		ok, val = t.Get(txId)
		if !ok {
			return boltvm.Error(fmt.Sprintf("cannot get global id of child tx id %s", txId))
		}

		globalId := string(val)
		txInfo := &TransactionInfo{}
		if !t.GetObject(t.txInfoKey(globalId), &txInfo) {
			return boltvm.Error(fmt.Sprintf("transaction global id %s does not exist", globalId))
		}

		if txInfo.globalState != StatusBegin {
			return boltvm.Error(fmt.Sprintf("transaction with global Id %s is finished", globalId))
		}

		status, ok := txInfo.childTxInfo[txId]
		if !ok {
			return boltvm.Error(fmt.Sprintf("%s is not in transaction %s", txId, globalId))
		}

		if status != StatusBegin {
			return boltvm.Error(fmt.Sprintf("%s has already reported result", txId))
		}

		if result == 0 {
			txInfo.childTxInfo[txId] = StatusSuccess
			count := 0
			for _, res := range txInfo.childTxInfo {
				if res != StatusSuccess {
					break
				}
				count++
			}

			if count == len(txInfo.childTxInfo) {
				txInfo.globalState = StatusSuccess
			}
		} else {
			txInfo.childTxInfo[txId] = StatusFail
			txInfo.globalState = StatusFail
		}

		t.SetObject(t.txInfoKey(globalId), txInfo)
	}

	return boltvm.Success(nil)
}

func (t *TransactionManager) txInfoKey(id string) string {
	return fmt.Sprintf("%s-%s", PREFIX, id)
}
