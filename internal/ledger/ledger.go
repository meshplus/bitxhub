package ledger

import (
	"math/big"

	"github.com/axiomesh/axiom-kit/types"
	vm "github.com/axiomesh/eth-kit/evm"
)

//go:generate mockgen -destination mock_ledger/mock_ledger.go -package mock_ledger -source ledger.go -typed

// ChainLedgerImpl handles block, transaction and receipt data.
type ChainLedger interface {
	// GetBlock get block with height
	GetBlock(height uint64) (*types.Block, error)

	// GetBlockSign get the signature of block
	GetBlockSign(height uint64) ([]byte, error)

	// GetBlockByHash get the block using block hash
	GetBlockByHash(hash *types.Hash) (*types.Block, error)

	// GetTransaction get the transaction using transaction hash
	GetTransaction(hash *types.Hash) (*types.Transaction, error)

	// GetTransactionMeta get the transaction meta data
	GetTransactionMeta(hash *types.Hash) (*types.TransactionMeta, error)

	// GetReceipt get the transaction receipt
	GetReceipt(hash *types.Hash) (*types.Receipt, error)

	// PersistExecutionResult persist the execution result
	PersistExecutionResult(block *types.Block, receipts []*types.Receipt) error

	// GetChainMeta get chain meta data
	GetChainMeta() *types.ChainMeta

	// UpdateChainMeta update the chain meta data
	UpdateChainMeta(*types.ChainMeta)

	// LoadChainMeta get chain meta data
	LoadChainMeta() (*types.ChainMeta, error)

	// GetTransactionCount get the transaction count in a block
	GetTransactionCount(height uint64) (uint64, error)

	RollbackBlockChain(height uint64) error

	GetBlockHash(height uint64) *types.Hash

	Close()
}

type StateLedger interface {
	StateAccessor

	vm.StateDB

	AddLog(log *types.EvmLog)

	GetLogs(types.Hash, uint64, *types.Hash) []*types.EvmLog

	// Rollback
	RollbackState(height uint64) error

	PrepareBlock(*types.Hash, uint64)

	ClearChangerAndRefund()

	// Close release resource
	Close()

	Finalise()

	// Version
	Version() uint64

	NewView() StateLedger
}

// StateAccessor manipulates the state data
type StateAccessor interface {
	// GetOrCreateAccount
	GetOrCreateAccount(*types.Address) IAccount

	// GetAccount
	GetAccount(*types.Address) IAccount

	// GetBalance
	GetBalance(*types.Address) *big.Int

	// SetBalance
	SetBalance(*types.Address, *big.Int)

	// GetState
	GetState(*types.Address, []byte) (bool, []byte)

	// SetState
	SetState(*types.Address, []byte, []byte)

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
	Commit(height uint64, accounts map[string]IAccount, stateRoot *types.Hash) error

	// FlushDirtyData flushes the dirty data
	FlushDirtyData() (map[string]IAccount, *types.Hash)

	// Set tx context for state db
	SetTxContext(thash *types.Hash, txIndex int)

	// Clear
	Clear()
}

type IAccount interface {
	GetAddress() *types.Address

	GetState(key []byte) (bool, []byte)

	GetCommittedState(key []byte) []byte

	SetState(key []byte, value []byte)

	SetCodeAndHash(code []byte)

	Code() []byte

	CodeHash() []byte

	SetNonce(nonce uint64)

	GetNonce() uint64

	GetBalance() *big.Int

	SetBalance(balance *big.Int)

	SubBalance(amount *big.Int)

	AddBalance(amount *big.Int)

	Query(prefix string) (bool, [][]byte)

	Finalise()

	IsEmpty() bool

	Suicided() bool

	SetSuicided(bool)
}
