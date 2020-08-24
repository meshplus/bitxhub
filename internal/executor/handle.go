package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cbergoon/merkletree"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm"
	"github.com/sirupsen/logrus"
)

func (exec *BlockExecutor) handleExecuteEvent(block *pb.Block) {
	if !exec.isDemandNumber(block.BlockHeader.Number) {
		exec.addPendingExecuteEvent(block)
		return
	}
	exec.processExecuteEvent(block)
	exec.handlePendingExecuteEvent()
}

func (exec *BlockExecutor) addPendingExecuteEvent(block *pb.Block) {
	exec.logger.WithFields(logrus.Fields{
		"received": block.BlockHeader.Number,
		"required": exec.currentHeight + 1,
	}).Warnf("Save wrong block into cache")

	exec.pendingBlockQ.Add(block.BlockHeader.Number, block)
}

func (exec *BlockExecutor) fetchPendingExecuteEvent(num uint64) *pb.Block {
	res, ok := exec.pendingBlockQ.Get(num)
	if !ok {
		return nil
	}

	return res.(*pb.Block)
}

func (exec *BlockExecutor) processExecuteEvent(block *pb.Block) {
	current := time.Now()
	exec.logger.WithFields(logrus.Fields{
		"height": block.BlockHeader.Number,
		"count":  len(block.Transactions),
	}).Infof("Execute block")

	exec.normalTxs = make([]types.Hash, 0)
	validTxs, invalidReceipts := exec.verifySign(block)
	receipts := exec.applyTransactions(validTxs)
	receipts = append(receipts, invalidReceipts...)

	calcMerkleStart := time.Now()
	l1Root, l2Roots, err := exec.buildTxMerkleTree(block.Transactions)
	if err != nil {
		panic(err)
	}

	receiptRoot, err := exec.calcReceiptMerkleRoot(receipts)
	if err != nil {
		panic(err)
	}

	calcMerkleDuration.Observe(float64(time.Since(calcMerkleStart)) / float64(time.Second))

	block.BlockHeader.TxRoot = l1Root
	block.BlockHeader.ReceiptRoot = receiptRoot
	block.BlockHeader.ParentHash = exec.currentBlockHash

	accounts, journal := exec.ledger.FlushDirtyDataAndComputeJournal()

	block.BlockHeader.StateRoot = journal.ChangedHash
	block.BlockHash = block.Hash()

	exec.logger.WithFields(logrus.Fields{
		"tx_root":      block.BlockHeader.TxRoot.ShortString(),
		"receipt_root": block.BlockHeader.ReceiptRoot.ShortString(),
		"state_root":   block.BlockHeader.StateRoot.ShortString(),
	}).Debug("block meta")
	calcBlockSize.Observe(float64(block.Size()))
	executeBlockDuration.Observe(float64(time.Since(current)) / float64(time.Second))

	counter := make(map[string]*pb.Uint64Slice)
	for k, v := range exec.interchainCounter {
		counter[k] = &pb.Uint64Slice{Slice: v}
	}
	interchainMeta := &pb.InterchainMeta{
		Counter: counter,
		L2Roots: l2Roots,
	}
	exec.postBlockEvent(block, interchainMeta)
	exec.clear()

	exec.currentHeight = block.BlockHeader.Number
	exec.currentBlockHash = block.BlockHash

	exec.persistC <- &ledger.BlockData{
		Block:          block,
		Receipts:       receipts,
		Accounts:       accounts,
		Journal:        journal,
		InterchainMeta: interchainMeta,
	}
}

func (exec *BlockExecutor) buildTxMerkleTree(txs []*pb.Transaction) (types.Hash, []types.Hash, error) {
	var (
		groupCnt = len(exec.interchainCounter) + 1
		wg       = sync.WaitGroup{}
		lock     = sync.Mutex{}
		l2Roots  = make([]types.Hash, 0, groupCnt)
		errorCnt = int32(0)
	)

	wg.Add(groupCnt - 1)
	for addr, txIndexes := range exec.interchainCounter {
		go func(addr string, txIndexes []uint64) {
			defer wg.Done()

			txHashes := make([]merkletree.Content, 0, len(txIndexes))
			for _, txIndex := range txIndexes {
				txHashes = append(txHashes, pb.TransactionHash(txs[txIndex].TransactionHash.Bytes()))
			}

			hash, err := calcMerkleRoot(txHashes)
			if err != nil {
				atomic.AddInt32(&errorCnt, 1)
				return
			}

			lock.Lock()
			defer lock.Unlock()
			l2Roots = append(l2Roots, hash)
		}(addr, txIndexes)
	}

	txHashes := make([]merkletree.Content, 0, len(exec.normalTxs))
	for _, txHash := range exec.normalTxs {
		txHashes = append(txHashes, pb.TransactionHash(txHash.Bytes()))
	}

	hash, err := calcMerkleRoot(txHashes)
	if err != nil {
		atomic.AddInt32(&errorCnt, 1)
	}

	lock.Lock()
	l2Roots = append(l2Roots, hash)
	lock.Unlock()

	wg.Wait()
	if errorCnt != 0 {
		return types.Hash{}, nil, fmt.Errorf("build tx merkle tree error")
	}

	sort.Slice(l2Roots, func(i, j int) bool {
		return bytes.Compare(l2Roots[i].Bytes(), l2Roots[j].Bytes()) < 0
	})

	contents := make([]merkletree.Content, 0, groupCnt)
	for _, l2Root := range l2Roots {
		contents = append(contents, pb.TransactionHash(l2Root.Bytes()))
	}
	root, err := calcMerkleRoot(contents)
	if err != nil {
		return types.Hash{}, nil, err
	}

	return root, l2Roots, nil
}

func (exec *BlockExecutor) verifySign(block *pb.Block) ([]*pb.Transaction, []*pb.Receipt) {
	if block.BlockHeader.Number == 1 {
		return block.Transactions, nil
	}

	txs := block.Transactions

	var (
		wg       sync.WaitGroup
		receipts []*pb.Receipt
		mutex    sync.Mutex
		index    []int
	)

	receiptsM := make(map[int]*pb.Receipt)

	wg.Add(len(txs))
	for i, tx := range txs {
		go func(i int, tx *pb.Transaction) {
			defer wg.Done()
			ok, _ := asym.Verify(crypto.Secp256k1, tx.Signature, tx.SignHash().Bytes(), tx.From)
			mutex.Lock()
			defer mutex.Unlock()
			if !ok {
				receiptsM[i] = &pb.Receipt{
					Version: tx.Version,
					TxHash:  tx.TransactionHash,
					Ret:     []byte("invalid signature"),
					Status:  pb.Receipt_FAILED,
				}

				index = append(index, i)
			}
		}(i, tx)

	}

	wg.Wait()

	if len(index) > 0 {
		sort.Ints(index)
		count := 0
		for _, idx := range index {
			receipts = append(receipts, receiptsM[idx])
			idx -= count
			txs = append(txs[:idx], txs[idx+1:]...)
			count++

		}
	}

	return txs, receipts
}

func (exec *BlockExecutor) applyTransactions(txs []*pb.Transaction) []*pb.Receipt {
	current := time.Now()
	receipts := make([]*pb.Receipt, 0, len(txs))

	for i, tx := range txs {
		receipt := &pb.Receipt{
			Version: tx.Version,
			TxHash:  tx.TransactionHash,
		}

		normalTx := true

		ret, err := exec.applyTransaction(i, tx)
		if err != nil {
			receipt.Status = pb.Receipt_FAILED
			receipt.Ret = []byte(err.Error())
		} else {
			receipt.Status = pb.Receipt_SUCCESS
			receipt.Ret = ret
		}

		events := exec.ledger.Events(tx.TransactionHash.Hex())
		if len(events) != 0 {
			receipt.Events = events
			for _, ev := range events {
				if ev.Interchain {
					m := make(map[string]uint64)
					err := json.Unmarshal(ev.Data, &m)
					if err != nil {
						panic(err)
					}

					for k, v := range m {
						exec.interchainCounter[k] = append(exec.interchainCounter[k], v)
					}
					normalTx = false
				}
			}
		}

		if normalTx {
			exec.normalTxs = append(exec.normalTxs, tx.TransactionHash)
		}

		receipts = append(receipts, receipt)
	}

	applyTxsDuration.Observe(float64(time.Since(current)) / float64(time.Second))
	exec.logger.WithFields(logrus.Fields{
		"time":  time.Since(current),
		"count": len(txs),
	}).Debug("Apply transactions elapsed")

	return receipts
}

func (exec *BlockExecutor) postBlockEvent(block *pb.Block, interchainMeta *pb.InterchainMeta) {
	go exec.blockFeed.Send(events.NewBlockEvent{Block: block, InterchainMeta: interchainMeta})
}

func (exec *BlockExecutor) handlePendingExecuteEvent() {
	if exec.pendingBlockQ.Len() > 0 {
		for exec.pendingBlockQ.Contains(exec.getDemandNumber()) {
			block := exec.fetchPendingExecuteEvent(exec.getDemandNumber())
			exec.processExecuteEvent(block)
		}
	}
}

func (exec *BlockExecutor) applyTransaction(i int, tx *pb.Transaction) ([]byte, error) {
	if tx.Data == nil {
		return nil, fmt.Errorf("empty transaction data")
	}

	switch tx.Data.Type {
	case pb.TransactionData_NORMAL:
		err := exec.transfer(tx.From, tx.To, tx.Data.Amount)
		return nil, err
	default:
		var instance vm.VM
		switch tx.Data.VmType {
		case pb.TransactionData_BVM:
			ctx := vm.NewContext(tx, uint64(i), tx.Data, exec.ledger, exec.logger)
			instance = boltvm.New(ctx, exec.validationEngine, exec.boltContracts)
		case pb.TransactionData_XVM:
			ctx := vm.NewContext(tx, uint64(i), tx.Data, exec.ledger, exec.logger)
			imports, err := wasm.EmptyImports()
			if err != nil {
				return nil, err
			}
			instance, err = wasm.New(ctx, imports, exec.wasmInstances)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("wrong vm type")
		}

		return instance.Run(tx.Data.Payload)
	}
}

func (exec *BlockExecutor) clear() {
	exec.interchainCounter = make(map[string][]uint64)
	exec.ledger.Clear()
}

func (exec *BlockExecutor) transfer(from, to types.Address, value uint64) error {
	if value == 0 {
		return nil
	}

	fv := exec.ledger.GetBalance(from)
	if fv < value {
		return fmt.Errorf("not sufficient funds for %s", from.Hex())
	}

	tv := exec.ledger.GetBalance(to)

	exec.ledger.SetBalance(from, fv-value)
	exec.ledger.SetBalance(to, tv+value)

	return nil
}

func (exec *BlockExecutor) calcReceiptMerkleRoot(receipts []*pb.Receipt) (types.Hash, error) {
	current := time.Now()

	receiptHashes := make([]merkletree.Content, 0, len(receipts))
	for _, receipt := range receipts {
		receiptHashes = append(receiptHashes, pb.TransactionHash(receipt.Hash().Bytes()))
	}
	receiptRoot, err := calcMerkleRoot(receiptHashes)
	if err != nil {
		return types.Hash{}, err
	}

	exec.logger.WithField("time", time.Since(current)).Debug("Calculate receipt merkle roots")

	return receiptRoot, nil
}

func calcMerkleRoot(contents []merkletree.Content) (types.Hash, error) {
	if len(contents) == 0 {
		return types.Hash{}, nil
	}

	tree, err := merkletree.NewTree(contents)
	if err != nil {
		return types.Hash{}, err
	}

	return types.Bytes2Hash(tree.MerkleRoot()), nil
}
