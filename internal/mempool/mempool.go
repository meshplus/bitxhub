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

func (mp *CoreMemPool) RecvTransaction(tx *pb.Transaction) error {
	// check if this tx signature is valid first
	ok, _ := asym.Verify(crypto.Secp256k1, tx.Signature, tx.SignHash().Bytes(), tx.From)
	if !ok {
		return fmt.Errorf("invalid signature")
	}

	// check the sequence number of tx
	currentSeqNo := mp.transactionStore.findLatestNonce(tx)
	if uint64(tx.Nonce) <= currentSeqNo {
		return fmt.Errorf("current sequence number is %d, required %d", tx.Nonce, currentSeqNo+1)
	}

	// check the existence of hash of this tx
	if ok := mp.transactionStore.hashOccurred(tx); ok {

	}
	// send tx to mempool store
	go mp.processTx(tx, currentSeqNo)
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
