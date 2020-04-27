package ledger

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/storage"
)

// PutBlock put block into store
func (l *ChainLedger) PutBlock(height uint64, block *pb.Block) error {
	data, err := block.Marshal()
	if err != nil {
		return err
	}

	l.blockchainStore.Put(compositeKey(blockKey, height), data)

	return nil
}

// GetBlock get block with height
func (l *ChainLedger) GetBlock(height uint64) (*pb.Block, error) {
	data := l.blockchainStore.Get(compositeKey(blockKey, height))
	if data == nil {
		return nil, storage.ErrorNotFound
	}

	block := &pb.Block{}
	if err := block.Unmarshal(data); err != nil {
		return nil, err
	}

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
func (l *ChainLedger) GetBlockByHash(hash types.Hash) (*pb.Block, error) {
	data := l.blockchainStore.Get(compositeKey(blockHashKey, hash.Hex()))
	if data == nil {
		return nil, storage.ErrorNotFound
	}

	height, err := strconv.Atoi(string(data))
	if err != nil {
		return nil, fmt.Errorf("wrong height, %w", err)
	}

	v := l.blockchainStore.Get(compositeKey(blockKey, height))
	if v == nil {
		return nil, fmt.Errorf("get block: %w", storage.ErrorNotFound)
	}

	block := &pb.Block{}
	if err := block.Unmarshal(v); err != nil {
		return nil, fmt.Errorf("unmarshal block: %w", err)
	}

	return block, nil
}

// GetTransaction get the transaction using transaction hash
func (l *ChainLedger) GetTransaction(hash types.Hash) (*pb.Transaction, error) {
	v := l.blockchainStore.Get(compositeKey(transactionKey, hash.Hex()))
	if v == nil {
		return nil, storage.ErrorNotFound
	}
	tx := &pb.Transaction{}
	if err := tx.Unmarshal(v); err != nil {
		return nil, err
	}

	return tx, nil
}

// GetTransactionMeta get the transaction meta data
func (l *ChainLedger) GetTransactionMeta(hash types.Hash) (*pb.TransactionMeta, error) {
	data := l.blockchainStore.Get(compositeKey(transactionMetaKey, hash.Hex()))
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
func (l *ChainLedger) GetReceipt(hash types.Hash) (*pb.Receipt, error) {
	data := l.blockchainStore.Get(compositeKey(receiptKey, hash.Hex()))
	if data == nil {
		return nil, storage.ErrorNotFound
	}

	r := &pb.Receipt{}
	if err := r.Unmarshal(data); err != nil {
		return nil, err
	}

	return r, nil
}

// PersistExecutionResult persist the execution result
func (l *ChainLedger) PersistExecutionResult(block *pb.Block, receipts []*pb.Receipt) error {
	current := time.Now()

	if block == nil {
		return fmt.Errorf("empty block data")
	}

	batcher := l.blockchainStore.NewBatch()

	if err := l.persistReceipts(batcher, receipts); err != nil {
		return err
	}

	if err := l.persistTransactions(batcher, block); err != nil {
		return err
	}

	if err := l.persistBlock(batcher, block); err != nil {
		return err
	}

	// update chain meta in cache
	count, err := getInterchainTxCount(block.BlockHeader)
	if err != nil {
		return err
	}

	meta := &pb.ChainMeta{
		Height:            block.BlockHeader.Number,
		BlockHash:         block.BlockHash,
		InterchainTxCount: count + l.chainMeta.InterchainTxCount,
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

func getInterchainTxCount(header *pb.BlockHeader) (uint64, error) {
	if header.InterchainIndex == nil {
		return 0, nil
	}

	txCount := make(map[string][]uint64)
	err := json.Unmarshal(header.InterchainIndex, &txCount)
	if err != nil {
		return 0, fmt.Errorf("get interchain tx count: %w", err)
	}

	var ret uint64
	for _, v := range txCount {
		ret += uint64(len(v))
	}

	return ret, nil
}

func (l *ChainLedger) persistReceipts(batcher storage.Batch, receipts []*pb.Receipt) error {
	for _, receipt := range receipts {
		data, err := receipt.Marshal()
		if err != nil {
			return err
		}

		batcher.Put(compositeKey(receiptKey, receipt.TxHash.Hex()), data)
	}

	return nil
}

func (l *ChainLedger) persistTransactions(batcher storage.Batch, block *pb.Block) error {
	for i, tx := range block.Transactions {
		body, err := tx.Marshal()
		if err != nil {
			return err
		}

		batcher.Put(compositeKey(transactionKey, tx.TransactionHash.Hex()), body)

		meta := &pb.TransactionMeta{
			BlockHeight: block.BlockHeader.Number,
			BlockHash:   block.BlockHash.Bytes(),
			Index:       uint64(i),
		}

		bs, err := meta.Marshal()
		if err != nil {
			return fmt.Errorf("marshal tx meta error: %s", err)
		}

		batcher.Put(compositeKey(transactionMetaKey, tx.TransactionHash.Hex()), bs)
	}

	return nil
}

func (l *ChainLedger) persistBlock(batcher storage.Batch, block *pb.Block) error {
	bs, err := block.Marshal()
	if err != nil {
		return err
	}

	height := block.BlockHeader.Number
	batcher.Put(compositeKey(blockKey, height), bs)

	hash := block.BlockHash.Hex()
	batcher.Put(compositeKey(blockHashKey, hash), []byte(fmt.Sprintf("%d", height)))

	return nil
}

func (l *ChainLedger) persistChainMeta(batcher storage.Batch, meta *pb.ChainMeta) error {
	data, err := meta.Marshal()
	if err != nil {
		return err
	}

	batcher.Put([]byte(chainMetaKey), data)

	return nil
}
