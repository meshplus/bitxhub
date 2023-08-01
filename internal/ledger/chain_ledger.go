package ledger

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/blockfile"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/repo"
	"github.com/axiomesh/eth-kit/ledger"
	"github.com/sirupsen/logrus"
)

var _ ledger.ChainLedger = (*ChainLedger)(nil)

type ChainLedger struct {
	blockchainStore storage.Storage
	bf              *blockfile.BlockFile
	repo            *repo.Repo
	chainMeta       *types.ChainMeta
	chainMutex      sync.RWMutex
	logger          logrus.FieldLogger
}

func NewChainLedgerImpl(blockchainStore storage.Storage, bf *blockfile.BlockFile, repo *repo.Repo, logger logrus.FieldLogger) (*ChainLedger, error) {
	c := &ChainLedger{
		blockchainStore: blockchainStore,
		bf:              bf,
		repo:            repo,
		chainMutex:      sync.RWMutex{},
		logger:          logger,
	}

	chainMeta, err := c.LoadChainMeta()
	if err != nil {
		return nil, fmt.Errorf("load chain meta: %w", err)
	}
	c.chainMeta = chainMeta
	return c, nil
}

// GetBlock get block with height
func (l *ChainLedger) GetBlock(height uint64) (*types.Block, error) {
	data, err := l.bf.Get(blockfile.BlockFileBodiesTable, height)
	if err != nil {
		return nil, fmt.Errorf("get bodies with height %d from blockfile failed: %w", height, err)
	}

	block := &types.Block{}
	if err := block.Unmarshal(data); err != nil {
		return nil, fmt.Errorf("unmarshal block error: %w", err)
	}

	txHashesData := l.blockchainStore.Get(compositeKey(blockTxSetKey, height))
	if txHashesData == nil {
		return nil, fmt.Errorf("cannot get tx hashes of block")
	}
	txHashes := make([]*types.Hash, 0)
	if err := json.Unmarshal(txHashesData, &txHashes); err != nil {
		return nil, fmt.Errorf("unmarshal tx hash data error: %w", err)
	}

	txsBytes, err := l.bf.Get(blockfile.BlockFileTXsTable, height)
	if err != nil {
		return nil, fmt.Errorf("get transactions with height %d from blockfile failed: %w", height, err)
	}
	txs, err := types.UnmarshalTransactions(txsBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshal txs bytes error: %w", err)
	}

	block.Transactions = txs

	return block, nil
}

func (l *ChainLedger) GetBlockHash(height uint64) *types.Hash {
	hash := l.blockchainStore.Get(compositeKey(blockHeightKey, height))
	if hash == nil {
		return &types.Hash{}
	}
	return types.NewHashByStr(string(hash))
}

// GetBlockSign get the signature of block
func (l *ChainLedger) GetBlockSign(height uint64) ([]byte, error) {
	block, err := l.GetBlock(height)
	if err != nil {
		return nil, fmt.Errorf("get block with height %d failed: %w", height, err)
	}

	return block.Signature, nil
}

// GetBlockByHash get the block using block hash
func (l *ChainLedger) GetBlockByHash(hash *types.Hash) (*types.Block, error) {
	data := l.blockchainStore.Get(compositeKey(blockHashKey, hash.String()))
	if data == nil {
		return nil, storage.ErrorNotFound
	}

	height, err := strconv.Atoi(string(data))
	if err != nil {
		return nil, fmt.Errorf("wrong height, %w", err)
	}

	return l.GetBlock(uint64(height))
}

// GetTransaction get the transaction using transaction hash
func (l *ChainLedger) GetTransaction(hash *types.Hash) (*types.Transaction, error) {
	metaBytes := l.blockchainStore.Get(compositeKey(transactionMetaKey, hash.String()))
	if metaBytes == nil {
		return nil, storage.ErrorNotFound
	}
	meta := &types.TransactionMeta{}
	if err := meta.Unmarshal(metaBytes); err != nil {
		return nil, fmt.Errorf("unmarshal transaction meta bytes error: %w", err)
	}
	txsBytes, err := l.bf.Get(blockfile.BlockFileTXsTable, meta.BlockHeight)
	if err != nil {
		return nil, fmt.Errorf("get transactions with height %d from blockfile failed: %w", meta.BlockHeight, err)
	}
	txs, err := types.UnmarshalTransactions(txsBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshal txs bytes error: %w", err)
	}

	return txs[meta.Index], nil
}

func (l *ChainLedger) GetTransactionCount(height uint64) (uint64, error) {
	txHashesData := l.blockchainStore.Get(compositeKey(blockTxSetKey, height))
	if txHashesData == nil {
		return 0, fmt.Errorf("cannot get tx hashes of block")
	}
	txHashes := make([]types.Hash, 0)
	if err := json.Unmarshal(txHashesData, &txHashes); err != nil {
		return 0, fmt.Errorf("unmarshal tx hash data error: %w", err)
	}

	return uint64(len(txHashes)), nil
}

// GetTransactionMeta get the transaction meta data
func (l *ChainLedger) GetTransactionMeta(hash *types.Hash) (*types.TransactionMeta, error) {
	data := l.blockchainStore.Get(compositeKey(transactionMetaKey, hash.String()))
	if data == nil {
		return nil, storage.ErrorNotFound
	}

	meta := &types.TransactionMeta{}
	if err := meta.Unmarshal(data); err != nil {
		return nil, fmt.Errorf("unmarshal transaction meta error: %w", err)
	}

	return meta, nil
}

// GetReceipt get the transaction receipt
func (l *ChainLedger) GetReceipt(hash *types.Hash) (*types.Receipt, error) {
	metaBytes := l.blockchainStore.Get(compositeKey(transactionMetaKey, hash.String()))
	if metaBytes == nil {
		return nil, storage.ErrorNotFound
	}
	meta := &types.TransactionMeta{}
	if err := meta.Unmarshal(metaBytes); err != nil {
		return nil, fmt.Errorf("unmarshal transaction meta bytes error: %w", err)
	}
	rsBytes, err := l.bf.Get(blockfile.BlockFileReceiptTable, meta.BlockHeight)
	if err != nil {
		return nil, fmt.Errorf("get receipts with height %d from blockfile failed: %w", meta.BlockHeight, err)
	}

	rs, err := types.UnmarshalReceipts(rsBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshal receipt bytes error: %w", err)
	}

	return rs[meta.Index], nil
}

// PersistExecutionResult persist the execution result
func (l *ChainLedger) PersistExecutionResult(block *types.Block, receipts []*types.Receipt) error {
	current := time.Now()

	if block == nil {
		return fmt.Errorf("empty block data")
	}

	batcher := l.blockchainStore.NewBatch()

	rs, err := l.prepareReceipts(batcher, block, receipts)
	if err != nil {
		return fmt.Errorf("preapare receipts failed: %w", err)
	}

	ts, err := l.prepareTransactions(batcher, block)
	if err != nil {
		return fmt.Errorf("prepare transactions failed: %w", err)
	}

	b, err := l.prepareBlock(batcher, block)
	if err != nil {
		return fmt.Errorf("prepare block failed: %w", err)
	}

	meta := &types.ChainMeta{
		Height:    block.BlockHeader.Number,
		BlockHash: block.BlockHash,
	}

	if err := l.bf.AppendBlock(l.chainMeta.Height, block.BlockHash.Bytes(), b, rs, ts); err != nil {
		return fmt.Errorf("append block with height %d to blockfile failed: %w", l.chainMeta.Height, err)
	}

	if err := l.persistChainMeta(batcher, meta); err != nil {
		return fmt.Errorf("persist chain meta failed: %w", err)
	}

	batcher.Commit()

	l.UpdateChainMeta(meta)

	l.logger.WithField("time", time.Since(current)).Debug("persist execution result elapsed")

	return nil
}

// UpdateChainMeta update the chain meta data
func (l *ChainLedger) UpdateChainMeta(meta *types.ChainMeta) {
	l.chainMutex.Lock()
	defer l.chainMutex.Unlock()
	l.chainMeta.Height = meta.Height
	l.chainMeta.BlockHash = meta.BlockHash
}

// GetChainMeta get chain meta data
func (l *ChainLedger) GetChainMeta() *types.ChainMeta {
	l.chainMutex.RLock()
	defer l.chainMutex.RUnlock()

	return &types.ChainMeta{
		Height:    l.chainMeta.Height,
		BlockHash: l.chainMeta.BlockHash,
	}
}

// LoadChainMeta load chain meta data
func (l *ChainLedger) LoadChainMeta() (*types.ChainMeta, error) {
	ok := l.blockchainStore.Has([]byte(chainMetaKey))

	chain := &types.ChainMeta{
		Height:    0,
		BlockHash: &types.Hash{},
	}
	if ok {
		body := l.blockchainStore.Get([]byte(chainMetaKey))
		if err := chain.Unmarshal(body); err != nil {
			return nil, fmt.Errorf("unmarshal chain meta: %w", err)
		}
	}

	return chain, nil
}

func (l *ChainLedger) prepareReceipts(_ storage.Batch, _ *types.Block, receipts []*types.Receipt) ([]byte, error) {
	return types.MarshalReceipts(receipts)
}

func (l *ChainLedger) prepareTransactions(batcher storage.Batch, block *types.Block) ([]byte, error) {
	for i, tx := range block.Transactions {
		meta := &types.TransactionMeta{
			BlockHeight: block.BlockHeader.Number,
			BlockHash:   block.BlockHash,
			Index:       uint64(i),
		}

		metaBytes, err := meta.Marshal()
		if err != nil {
			return nil, fmt.Errorf("marshal tx meta error: %s", err)
		}

		batcher.Put(compositeKey(transactionMetaKey, tx.GetHash().String()), metaBytes)
	}

	return types.MarshalTransactions(block.Transactions)
}

func (l *ChainLedger) prepareBlock(batcher storage.Batch, block *types.Block) ([]byte, error) {
	// Generate block header signature
	if block.Signature == nil {
		signed, err := l.repo.Key.PrivKey.Sign(block.BlockHash.Bytes())
		if err != nil {
			return nil, fmt.Errorf("sign block %s failed: %w", block.BlockHash.String(), err)
		}

		block.Signature = signed
	}

	storedBlock := &types.Block{
		BlockHeader:  block.BlockHeader,
		Transactions: nil,
		BlockHash:    block.BlockHash,
		Signature:    block.Signature,
		Extra:        block.Extra,
	}
	bs, err := storedBlock.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal stored block error: %w", err)
	}

	height := block.BlockHeader.Number

	var txHashes []*types.Hash
	for _, tx := range block.Transactions {
		txHashes = append(txHashes, tx.GetHash())
	}

	data, err := json.Marshal(txHashes)
	if err != nil {
		return nil, fmt.Errorf("marshal tx hash error: %w", err)
	}

	batcher.Put(compositeKey(blockTxSetKey, height), data)

	hash := block.BlockHash.String()
	batcher.Put(compositeKey(blockHashKey, hash), []byte(fmt.Sprintf("%d", height)))
	batcher.Put(compositeKey(blockHeightKey, height), []byte(hash))

	return bs, nil
}

func (l *ChainLedger) persistChainMeta(batcher storage.Batch, meta *types.ChainMeta) error {
	data, err := meta.Marshal()
	if err != nil {
		return fmt.Errorf("marshal chain meta error: %w", err)
	}

	batcher.Put([]byte(chainMetaKey), data)

	return nil
}

func (l *ChainLedger) removeChainDataOnBlock(batch storage.Batch, height uint64) error {
	block, err := l.GetBlock(height)
	if err != nil {
		return fmt.Errorf("get block with height %d failed: %w", height, err)
	}

	if err := l.bf.TruncateBlocks(height - 1); err != nil {
		return fmt.Errorf("truncate blocks failed: %w", err)
	}

	batch.Delete(compositeKey(blockTxSetKey, height))
	batch.Delete(compositeKey(blockHashKey, block.BlockHash.String()))
	batch.Delete(compositeKey(interchainMetaKey, height))

	for _, tx := range block.Transactions {
		batch.Delete(compositeKey(transactionMetaKey, tx.GetHash().String()))
	}

	return nil
}

func (l *ChainLedger) RollbackBlockChain(height uint64) error {
	meta := l.GetChainMeta()

	if meta.Height < height {
		return ErrorRollbackToHigherNumber
	}

	if meta.Height == height {
		return nil
	}

	batch := l.blockchainStore.NewBatch()

	for i := meta.Height; i > height; i-- {
		err := l.removeChainDataOnBlock(batch, i)
		if err != nil {
			return fmt.Errorf("remove chain data on block %d failed: %w", i, err)
		}
	}

	if height == 0 {
		batch.Delete([]byte(chainMetaKey))
		meta = &types.ChainMeta{}
	} else {
		block, err := l.GetBlock(height)
		if err != nil {
			return fmt.Errorf("get block with height %d failed: %w", height, err)
		}
		meta = &types.ChainMeta{
			Height:    block.BlockHeader.Number,
			BlockHash: block.BlockHash,
		}

		if err := l.persistChainMeta(batch, meta); err != nil {
			return fmt.Errorf("persist chain meta failed: %w", err)
		}
	}

	batch.Commit()

	l.UpdateChainMeta(meta)

	return nil
}

func (l *ChainLedger) Close() {
	l.blockchainStore.Close()
	l.bf.Close()
}
