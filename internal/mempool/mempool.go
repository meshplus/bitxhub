package mempool

import (
	"fmt"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/pb"
)

var _ MemPool = (*CoreMemPool)(nil)

const (
	btreeDegree = 10
)

type CoreMemPool struct {
	transactionStore *transactionStore
}

func New() *CoreMemPool {
	return &CoreMemPool{
		transactionStore: newTransactionStore(),
	}
}

func (mp *CoreMemPool) RecvTransactions(txs []*pb.Transaction) error {
	for _, tx := range txs {
		// check if this tx signature is valid first
		ok, _ := asym.Verify(crypto.Secp256k1, tx.Signature, tx.SignHash().Bytes(), tx.From)
		if !ok {
			return fmt.Errorf("invalid signature")
		}

		// check the sequence number of tx
		account := tx.From.Hex()
		currentSeqNo := mp.transactionStore.getPendingNonce(account)
		if uint64(tx.Nonce) <= currentSeqNo {
			return fmt.Errorf("current sequence number is %d, required %d", tx.Nonce, currentSeqNo+1)
		}

		// check the existence of hash of this tx
		if ok := mp.transactionStore.txHashMap[tx.TransactionHash.Hex()]; ok {
			return fmt.Errorf("tx already received")
		}
		mp.transactionStore.txHashMap[tx.TransactionHash.Hex()] = true

		accountTxs, ok := mp.transactionStore.allTxs[account]
		if !ok {
			// if this is new account to send tx, create a new txSortedMap
			accountTxs = newTxSortedMap()
			mp.transactionStore.allTxs[account] = accountTxs
		}
		accountTxs.items[uint64(tx.Nonce)] = tx
		accountTxs.index.insert([]*pb.Transaction{tx})
	}
	// send tx to mempool store
	mp.processDirtyAccount(txs)
	return nil
}

func (mp *CoreMemPool) GetBlock() []*pb.TransactionHash {
	// todo: add implementation
	return nil
}

func (mp *CoreMemPool) CommitTransactions(hashes []*pb.TransactionHash) {
	// todo: add implementation
}

func (mp *CoreMemPool) FetchMissingTransactions(MissingTransactionsHashList []*pb.Transaction) {
	// todo: add implementation
}

func (mp *CoreMemPool) RecvMissingTransactions(MissingTransactionsList []*pb.Transaction) {
	// todo: add implementation
}
