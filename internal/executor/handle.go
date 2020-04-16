package executor

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/merkle/merkletree"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
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
	exec.logger.WithFields(logrus.Fields{
		"height": block.BlockHeader.Number,
		"count":  len(block.Transactions),
	}).Infof("Execute block")

	validTxs, invalidReceipts := exec.verifySign(block)
	receipts := exec.applyTransactions(validTxs)

	root, receiptRoot, err := exec.calcMerkleRoots(block.Transactions, append(receipts, invalidReceipts...))
	if err != nil {
		panic(err)
	}

	block.BlockHeader.TxRoot = root
	block.BlockHeader.ReceiptRoot = receiptRoot
	block.BlockHeader.ParentHash = exec.currentBlockHash
	idx, err := json.Marshal(exec.interchainCounter)
	if err != nil {
		panic(err)
	}

	block.BlockHeader.InterchainIndex = idx
	hash, err := exec.ledger.Commit()
	if err != nil {
		panic(err)
	}
	block.BlockHeader.StateRoot = hash
	block.BlockHash = block.Hash()

	exec.logger.WithFields(logrus.Fields{
		"tx_root":      block.BlockHeader.TxRoot.ShortString(),
		"receipt_root": block.BlockHeader.ReceiptRoot.ShortString(),
		"state_root":   block.BlockHeader.StateRoot.ShortString(),
	}).Debug("block meta")

	// persist execution result
	receipts = append(receipts, invalidReceipts...)
	if err := exec.ledger.PersistExecutionResult(block, receipts); err != nil {
		panic(err)
	}

	exec.logger.WithFields(logrus.Fields{
		"height": block.BlockHeader.Number,
		"hash":   block.BlockHash.ShortString(),
		"count":  len(block.Transactions),
	}).Info("Persist block")

	exec.postBlockEvent(block)
	exec.clear()

	exec.currentHeight = block.BlockHeader.Number
	exec.currentBlockHash = block.BlockHash
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
			ok, _ := asym.Verify(asym.ECDSASecp256r1, tx.Signature, tx.SignHash().Bytes(), tx.From)
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
				}
			}
		}

		receipts = append(receipts, receipt)
	}

	exec.logger.WithFields(logrus.Fields{
		"time":  time.Since(current),
		"count": len(txs),
	}).Debug("Apply transactions elapsed")

	return receipts
}

func (exec *BlockExecutor) postBlockEvent(block *pb.Block) {
	go exec.blockFeed.Send(events.NewBlockEvent{Block: block})

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
			instance = boltvm.New(ctx, exec.validationEngine)
		case pb.TransactionData_XVM:
			ctx := vm.NewContext(tx, uint64(i), tx.Data, exec.ledger, exec.logger)
			imports, err := wasm.EmptyImports()
			if err != nil {
				return nil, err
			}
			instance, err = wasm.New(ctx, imports)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("wrong vm type")
		}

		return instance.Run(tx.Data.Payload)
	}
}

func (exec *BlockExecutor) calcMerkleRoot(txs []*pb.Transaction) (types.Hash, error) {
	if len(txs) == 0 {
		return types.Hash{}, nil
	}

	hashes := make([]interface{}, 0, len(txs))
	for _, tx := range txs {
		hashes = append(hashes, pb.TransactionHash(tx.TransactionHash.Bytes()))
	}
	tree := merkletree.NewMerkleTree()
	err := tree.InitMerkleTree(hashes)
	if err != nil {
		return types.Hash{}, err
	}

	return types.Bytes2Hash(tree.GetMerkleRoot()), nil
}

func (exec *BlockExecutor) calcReceiptMerkleRoot(receipts []*pb.Receipt) (types.Hash, error) {
	if len(receipts) == 0 {
		return types.Hash{}, nil
	}

	hashes := make([]interface{}, 0, len(receipts))
	for _, receipt := range receipts {
		hashes = append(hashes, pb.TransactionHash(receipt.Hash().Bytes()))
	}
	tree := merkletree.NewMerkleTree()
	err := tree.InitMerkleTree(hashes)
	if err != nil {
		return types.Hash{}, err
	}

	return types.Bytes2Hash(tree.GetMerkleRoot()), nil
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

func (exec *BlockExecutor) calcMerkleRoots(txs []*pb.Transaction, receipts []*pb.Receipt) (types.Hash, types.Hash, error) {
	current := time.Now()
	root, err := exec.calcMerkleRoot(txs)
	if err != nil {
		return types.Hash{}, types.Hash{}, err
	}

	receiptRoot, err := exec.calcReceiptMerkleRoot(receipts)
	if err != nil {
		return types.Hash{}, types.Hash{}, err
	}

	exec.logger.WithField("time", time.Since(current)).Debug("calculate merkle roots")

	return root, receiptRoot, nil
}
