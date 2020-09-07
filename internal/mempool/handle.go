package mempool

import "github.com/meshplus/bitxhub-model/pb"

func (mp *CoreMemPool) processTx(tx *pb.Transaction, currentSeqNo uint64) {
	// judge whether this tx will be in btreeIndex or parkingIndex
	if uint64(tx.Nonce) != currentSeqNo+1 {
		// if this tx is not ready to commit, put it into parkingIndex
		mp.transactionStore.parkingLotIndex.insert([]*pb.Transaction{tx})
		return
	}

	// if this tx is ready to commit: follow the steps:
	// 1. find all sequential allTxs in parkingIndex;
	// 2. insert these allTxs into priorityIndex.

	// search for related sequential allTxs in parkingIndex
	// and add these tx into readyIndex as well
	dirtyAcount := mp.transactionStore.allTxs[tx.From.Hex()]
	readyTxs, nonReadyTxs, _ := dirtyAcount.filterReady(makeKey(tx))

	mp.transactionStore.priorityIndex.insert(readyTxs)
	mp.transactionStore.parkingLotIndex.insert(nonReadyTxs)
}
