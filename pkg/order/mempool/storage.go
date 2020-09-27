package mempool

import (
	"fmt"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

// batchStore persists batch into DB, which
func (mpi *mempoolImpl) batchStore(txList []*pb.Transaction) {
	batch := mpi.storage.NewBatch()
	for _, tx := range txList {
		txKey := compositeKey(tx.TransactionHash.Bytes())
		txData, _ := tx.Marshal()
		batch.Put(txKey, txData)
	}
	batch.Commit()
}

// batchDelete batch delete txs
func (mpi *mempoolImpl) batchDelete(hashes []types.Hash) {
	batch := mpi.storage.NewBatch()
	for _, hash := range hashes {
		txKey := compositeKey(hash.Bytes())
		batch.Delete(txKey)
	}
	batch.Commit()
}

func (mpi *mempoolImpl) store(tx *pb.Transaction) {
	txKey := compositeKey(tx.TransactionHash.Bytes())
	txData, _ := tx.Marshal()
	mpi.storage.Put(txKey, txData)
}

func (mpi *mempoolImpl) load(hash types.Hash) (*pb.Transaction, bool) {
	txKey := compositeKey(hash.Bytes())
	txData := mpi.storage.Get(txKey)
	if txData == nil {
		return nil, false
	}
	var tx pb.Transaction
	if err := tx.Unmarshal(txData); err != nil {
		mpi.logger.Error(err)
		return nil, false
	}
	return &tx, true
}

func compositeKey(value interface{}) []byte {
	var prefix = []byte("tx-")
	return append(prefix, []byte(fmt.Sprintf("%v", value))...)
}
