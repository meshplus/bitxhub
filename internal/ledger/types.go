package ledger

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

//go:generate mockgen -destination mock_ledger/mock_ledger.go -package mock_ledger -source types.go
type Ledger interface {
	BlockchainLedger
	StateAccessor

	AccountCache() *AccountCache

	// PersistBlockData
	PersistBlockData(blockData *BlockData)

	// AddEvent
	AddEvent(*pb.Event)

	// Events
	Events(txHash string) []*pb.Event

	// Rollback
	Rollback(height uint64) error

	// RemoveJournalsBeforeBlock
	RemoveJournalsBeforeBlock(height uint64) error

	// Close release resource
	Close()
}

// StateAccessor manipulates the state data
type StateAccessor interface {
	// GetOrCreateAccount
	GetOrCreateAccount(*types.Address) *Account

	// GetAccount
	GetAccount(*types.Address) *Account

	// GetBalance
	GetBalance(*types.Address) uint64

	// SetBalance
	SetBalance(*types.Address, uint64)

	// GetState
	GetState(*types.Address, []byte) (bool, []byte)

	// SetState
	SetState(*types.Address, []byte, []byte)

	// AddState
	AddState(*types.Address, []byte, []byte)

	// SetCode
	SetCode(*types.Address, []byte)

	// GetCode
	GetCode(*types.Address) []byte

	// SetNonce
	SetNonce(*types.Address, uint64)

	// GetNonce
	GetNonce(*types.Address) uint64

	// QueryByPrefix
	QueryByPrefix(address *types.Address, prefix string) (bool, [][]byte)

	// Commit commits the state data
	Commit(height uint64, accounts map[string]*Account, blockJournal *BlockJournal) error

	// FlushDirtyDataAndComputeJournal flushes the dirty data and computes block journal
	FlushDirtyDataAndComputeJournal() (map[string]*Account, *BlockJournal)

	// Version
	Version() uint64

	// Clear
	Clear()
}

// BlockchainLedger handles block, transaction and receipt data.
type BlockchainLedger interface {
	// PutBlock put block into store
	PutBlock(height uint64, block *pb.Block) error

	// GetBlock get block with height
	GetBlock(height uint64) (*pb.Block, error)

	// GetBlockSign get the signature of block
	GetBlockSign(height uint64) ([]byte, error)

	// GetBlockByHash get the block using block hash
	GetBlockByHash(hash *types.Hash) (*pb.Block, error)

	// GetTransaction get the transaction using transaction hash
	GetTransaction(hash *types.Hash) (*pb.Transaction, error)

	// GetTransactionMeta get the transaction meta data
	GetTransactionMeta(hash *types.Hash) (*pb.TransactionMeta, error)

	// GetReceipt get the transaction receipt
	GetReceipt(hash *types.Hash) (*pb.Receipt, error)

	// GetInterchainMeta get interchain meta data
	GetInterchainMeta(height uint64) (*pb.InterchainMeta, error)

	// PersistExecutionResult persist the execution result
	PersistExecutionResult(block *pb.Block, receipts []*pb.Receipt, meta *pb.InterchainMeta) error

	// GetChainMeta get chain meta data
	GetChainMeta() *pb.ChainMeta

	// UpdateChainMeta update the chain meta data
	UpdateChainMeta(*pb.ChainMeta)

	// GetTxCountInBlock get the transaction count in a block
	GetTransactionCount(height uint64) (uint64, error)
}
