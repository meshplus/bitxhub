package mempool

import "github.com/meshplus/bitxhub-model/pb"

func (mp *CoreMemPool) processDirtyAccount(txs []*pb.Transaction) {
	for _, tx := range txs {
		account := tx.From.Hex()
		accountTxs := mp.transactionStore.allTxs[account]

		// if this tx is ready to commit: follow the steps:
		// 1. find all sequential txs in parkingLotIndex;
		// 2. insert these txs into priorityIndex.

		// search for related sequential txs in parkingLotIndex
		// and add these txs into priorityIndex
		pendingNonce := mp.transactionStore.getPendingNonce(account)
		readyTxs, nonReadyTxs, nextDemandNonce := accountTxs.filterReady(
			&orderedQueueKey{
				accountAddress: account,
				seqNo:          pendingNonce,
			})
		mp.transactionStore.setPendingNonce(account, nextDemandNonce)

		// inset ready txs into priorityIndex.
		mp.transactionStore.priorityIndex.insert(readyTxs)

		// inset non-ready txs into parkingLotIndex.
		mp.transactionStore.parkingLotIndex.insert(nonReadyTxs)
	}
}
