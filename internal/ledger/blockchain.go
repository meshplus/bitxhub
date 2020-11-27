package ledger

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

// PutBlock put block into store
func (l *ChainLedger) PutBlock(height uint64, block *pb.Block) error {
	// deprecated

	return nil
}

// GetBlock get block with height
func (l *ChainLedger) GetBlock(height uint64) (*pb.Block, error) {
	data, err := l.bf.Get(blockfile.BlockFileBodiesTable, height)
	if err != nil {
		return nil, err
	}

	block := &pb.Block{}
	if err := block.Unmarshal(data); err != nil {
		return nil, err
	}

	txHashesData := l.blockchainStore.Get(compositeKey(blockTxSetKey, height))
	if txHashesData == nil {
		return nil, fmt.Errorf("cannot get tx hashes of block")
	}
	txHashes := make([]*types.Hash, 0)
	if err := json.Unmarshal(txHashesData, &txHashes); err != nil {
		return nil, err
	}

	txs := &pb.Transactions{}
	txsBytes, err := l.bf.Get(blockfile.BlockFileTXsTable, height)
	if err != nil {
		return nil, err
	}
	if err := txs.Unmarshal(txsBytes); err != nil {
		return nil, err
	}

	block.Transactions = txs.Transactions

	return block, nil
}

// GetBlockSign get the signature of block
func (l *ChainLedger) GetBlockSign(height uint64) ([]byte, error) {
	block, err := l.GetBlock(height)
	if err != nil {
		return nil, err
	}

	return block.Signature, nil
}

// GetBlockByHash get the block using block hash
func (l *ChainLedger) GetBlockByHash(hash *types.Hash) (*pb.Block, error) {
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
func (l *ChainLedger) GetTransaction(hash *types.Hash) (*pb.Transaction, error) {
	metaBytes := l.blockchainStore.Get(compositeKey(transactionMetaKey, hash.String()))
	if metaBytes == nil {
		return nil, storage.ErrorNotFound
	}
	meta := &pb.TransactionMeta{}
	if err := meta.Unmarshal(metaBytes); err != nil {
		return nil, err
	}
	txsBytes, err := l.bf.Get(blockfile.BlockFileTXsTable, meta.BlockHeight)
	if err != nil {
		return nil, err
	}
	txs := &pb.Transactions{}
	if err := txs.Unmarshal(txsBytes); err != nil {
		return nil, err
	}

	return txs.Transactions[meta.Index], nil
}

func (l *ChainLedger) GetTransactionCount(height uint64) (uint64, error) {
	txHashesData := l.blockchainStore.Get(compositeKey(blockTxSetKey, height))
	if txHashesData == nil {
		return 0, fmt.Errorf("cannot get tx hashes of block")
	}
	txHashes := make([]types.Hash, 0)
	if err := json.Unmarshal(txHashesData, &txHashes); err != nil {
		return 0, err
	}

	return uint64(len(txHashes)), nil
}

// GetTransactionMeta get the transaction meta data
func (l *ChainLedger) GetTransactionMeta(hash *types.Hash) (*pb.TransactionMeta, error) {
	data := l.blockchainStore.Get(compositeKey(transactionMetaKey, hash.String()))
	if data == nil {
		return nil, storage.ErrorNotFound
	}

	meta := &pb.TransactionMeta{}
	if err := meta.Unmarshal(data); err != nil {
		return nil, err
	}

	return meta, nil
}

// GetReceipt get the transaction receipt
func (l *ChainLedger) GetReceipt(hash *types.Hash) (*pb.Receipt, error) {
	metaBytes := l.blockchainStore.Get(compositeKey(transactionMetaKey, hash.String()))
	if metaBytes == nil {
		return nil, storage.ErrorNotFound
	}
	meta := &pb.TransactionMeta{}
	if err := meta.Unmarshal(metaBytes); err != nil {
		return nil, err
	}
	rsBytes, err := l.bf.Get(blockfile.BlockFileReceiptTable, meta.BlockHeight)
	if err != nil {
		return nil, err
	}
	rs := &pb.Receipts{}
	if err := rs.Unmarshal(rsBytes); err != nil {
		return nil, err
	}

	return rs.Receipts[meta.Index], nil
}

// PersistExecutionResult persist the execution result
func (l *ChainLedger) PersistExecutionResult(block *pb.Block, receipts []*pb.Receipt, interchainMeta *pb.InterchainMeta) error {
	current := time.Now()

	if block == nil {
		return fmt.Errorf("empty block data")
	}

	batcher := l.blockchainStore.NewBatch()

	rs, err := l.prepareReceipts(batcher, block, receipts)
	if err != nil {
		return err
	}

	ts, err := l.prepareTransactions(batcher, block)
	if err != nil {
		return err
	}

	b, err := l.prepareBlock(batcher, block)
	if err != nil {
		return err
	}

	im, err := interchainMeta.Marshal()
	if err != nil {
		return err
	}

	// update chain meta in cache
	var count uint64
	for _, v := range interchainMeta.Counter {
		count += uint64(len(v.Slice))
	}

	meta := &pb.ChainMeta{
		Height:            block.BlockHeader.Number,
		BlockHash:         block.BlockHash,
		InterchainTxCount: count + l.chainMeta.InterchainTxCount,
	}

	if err := l.bf.AppendBlock(l.chainMeta.Height, block.BlockHash.Bytes(), b, rs, ts, im); err != nil {
		return err
	}

	if err := l.persistChainMeta(batcher, meta); err != nil {
		return err
	}

	batcher.Commit()

	l.UpdateChainMeta(meta)

	l.logger.WithField("time", time.Since(current)).Debug("persist execution result elapsed")

	return nil
}

// UpdateChainMeta update the chain meta data
func (l *ChainLedger) UpdateChainMeta(meta *pb.ChainMeta) {
	l.chainMutex.Lock()
	defer l.chainMutex.Unlock()
	l.chainMeta.Height = meta.Height
	l.chainMeta.BlockHash = meta.BlockHash
	l.chainMeta.InterchainTxCount = meta.InterchainTxCount
}

// GetChainMeta get chain meta data
func (l *ChainLedger) GetChainMeta() *pb.ChainMeta {
	l.chainMutex.RLock()
	defer l.chainMutex.RUnlock()

	return &pb.ChainMeta{
		Height:            l.chainMeta.Height,
		BlockHash:         l.chainMeta.BlockHash,
		InterchainTxCount: l.chainMeta.InterchainTxCount,
	}
}

func (l *ChainLedger) GetInterchainMeta(height uint64) (*pb.InterchainMeta, error) {
	data, err := l.bf.Get(blockfile.BlockFileInterchainTable, height)
	if err != nil {
		return nil, err
	}

	meta := &pb.InterchainMeta{}
	if err := meta.Unmarshal(data); err != nil {
		return nil, err
	}

	return meta, nil
}

func (l *ChainLedger) prepareReceipts(batcher storage.Batch, block *pb.Block, receipts []*pb.Receipt) ([]byte, error) {
	rs := &pb.Receipts{
		Receipts: receipts,
	}

	return rs.Marshal()
}

func (l *ChainLedger) prepareTransactions(batcher storage.Batch, block *pb.Block) ([]byte, error) {
	for i, tx := range block.Transactions {
		meta := &pb.TransactionMeta{
			BlockHeight: block.BlockHeader.Number,
			BlockHash:   block.BlockHash.Bytes(),
			Index:       uint64(i),
		}

		metaBytes, err := meta.Marshal()
		if err != nil {
			return nil, fmt.Errorf("marshal tx meta error: %s", err)
		}

		batcher.Put(compositeKey(transactionMetaKey, tx.TransactionHash.String()), metaBytes)
	}

	ts := &pb.Transactions{
		Transactions: block.Transactions,
	}
	return ts.Marshal()
}

func (l *ChainLedger) prepareBlock(batcher storage.Batch, block *pb.Block) ([]byte, error) {
	// Generate block header signature
	signed, err := l.repo.Key.PrivKey.Sign(block.BlockHash.Bytes())
	if err != nil {
		return nil, err
	}

	block.Signature = signed

	storedBlock := &pb.Block{
		BlockHeader:  block.BlockHeader,
		Transactions: nil,
		BlockHash:    block.BlockHash,
		Signature:    block.Signature,
		Extra:        block.Extra,
	}
	bs, err := storedBlock.Marshal()
	if err != nil {
		return nil, err
	}

	height := block.BlockHeader.Number

	var txHashes []types.Hash
	for _, tx := range block.Transactions {
		txHashes = append(txHashes, *tx.TransactionHash)
	}

	data, err := json.Marshal(txHashes)
	if err != nil {
		return nil, err
	}

	batcher.Put(compositeKey(blockTxSetKey, height), data)

	hash := block.BlockHash.String()
	batcher.Put(compositeKey(blockHashKey, hash), []byte(fmt.Sprintf("%d", height)))

	return bs, nil
}

func (l *ChainLedger) persistChainMeta(batcher storage.Batch, meta *pb.ChainMeta) error {
	data, err := meta.Marshal()
	if err != nil {
		return err
	}

	batcher.Put([]byte(chainMetaKey), data)

	return nil
}

func (l *ChainLedger) removeChainDataOnBlock(batch storage.Batch, height uint64) (uint64, error) {
	block, err := l.GetBlock(height)
	if err != nil {
		return 0, err
	}
	interchainMeta, err := l.GetInterchainMeta(height)
	if err != nil {
		return 0, err
	}

	if err := l.bf.TruncateBlocks(height - 1); err != nil {
		return 0, err
	}

	batch.Delete(compositeKey(blockTxSetKey, height))
	batch.Delete(compositeKey(blockHashKey, block.BlockHash.String()))
	batch.Delete(compositeKey(interchainMetaKey, height))

	for _, tx := range block.Transactions {
		batch.Delete(compositeKey(transactionMetaKey, tx.TransactionHash.String()))
	}

	return getInterchainTxCount(interchainMeta), nil
}

func (l *ChainLedger) rollbackBlockChain(height uint64) error {
	meta := l.GetChainMeta()

	if meta.Height < height {
		return ErrorRollbackToHigherNumber
	}

	if meta.Height == height {
		return nil
	}

	batch := l.blockchainStore.NewBatch()

	for i := meta.Height; i > height; i-- {
		count, err := l.removeChainDataOnBlock(batch, i)
		if err != nil {
			return err
		}
		meta.InterchainTxCount -= count
	}

	if height == 0 {
		batch.Delete([]byte(chainMetaKey))
		meta = &pb.ChainMeta{}
	} else {
		block, err := l.GetBlock(height)
		if err != nil {
			return err
		}
		meta = &pb.ChainMeta{
			Height:            block.BlockHeader.Number,
			BlockHash:         block.BlockHash,
			InterchainTxCount: meta.InterchainTxCount,
		}

		if err := l.persistChainMeta(batch, meta); err != nil {
			return err
		}
	}

	batch.Commit()

	l.UpdateChainMeta(meta)

	return nil
}

func getInterchainTxCount(interchainMeta *pb.InterchainMeta) uint64 {
	var ret uint64
	for _, v := range interchainMeta.Counter {
		ret += uint64(len(v.Slice))
	}

	return ret
}
