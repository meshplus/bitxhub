package mempool

import "github.com/meshplus/bitxhub-model/pb"

//go:generate mockgen -destination mock_mempool/mock_mempool.go -package mock_mempool -source types.go
type MemPool interface {
	// Recv transactions from clients
	RecvTransactions(txs []*pb.Transaction) error

	//  Fetch ordered block of transactions
	GetBlock() []*pb.TransactionHash

	// Remove committed transactions from PendingPool
	CommitTransactions(hashes []*pb.TransactionHash)

	//  Fetch missing transactions
	FetchMissingTransactions(MissingTransactionsHashList []*pb.Transaction)

	// Receive the transactions
	RecvMissingTransactions(MissingTransactionsList []*pb.Transaction)
}
