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
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm/vmledger"
	"github.com/sirupsen/logrus"
)

type BlockWrapper struct {
	block     *pb.Block
	invalidTx map[int]agency.InvalidReason
}

func (exec *BlockExecutor) processExecuteEvent(blockWrapper *BlockWrapper) *ledger.BlockData {
	var txHashList []*types.Hash
	current := time.Now()
	block := blockWrapper.block

	for _, tx := range block.Transactions.Transactions {
		txHashList = append(txHashList, tx.GetHash())
	}

	exec.verifyProofs(blockWrapper)
	receipts := exec.txsExecutor.ApplyTransactions(block.Transactions.Transactions, blockWrapper.invalidTx)

	applyTxsDuration.Observe(float64(time.Since(current)) / float64(time.Second))
	exec.logger.WithFields(logrus.Fields{
		"time":  time.Since(current),
		"count": len(block.Transactions.Transactions),
	}).Debug("Apply transactions elapsed")

	calcMerkleStart := time.Now()
	l1Root, l2Roots, err := exec.buildTxMerkleTree(block.Transactions.Transactions)
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
		"tx_root":      block.BlockHeader.TxRoot.String(),
		"receipt_root": block.BlockHeader.ReceiptRoot.String(),
		"state_root":   block.BlockHeader.StateRoot.String(),
	}).Debug("block meta")
	calcBlockSize.Observe(float64(block.Size()))
	executeBlockDuration.Observe(float64(time.Since(current)) / float64(time.Second))

	counter := make(map[string]*pb.Uint64Slice)
	for k, v := range exec.txsExecutor.GetInterchainCounter() {
		counter[k] = &pb.Uint64Slice{Slice: v}
	}
	interchainMeta := &pb.InterchainMeta{
		Counter: counter,
		L2Roots: l2Roots,
	}
	exec.clear()

	exec.currentHeight = block.BlockHeader.Number
	exec.currentBlockHash = block.BlockHash

	return &ledger.BlockData{
		Block:          block,
		Receipts:       receipts,
		Accounts:       accounts,
		Journal:        journal,
		InterchainMeta: interchainMeta,
		TxHashList:     txHashList,
	}
}

func (exec *BlockExecutor) listenPreExecuteEvent() {
	for {
		select {
		case commitEvent := <-exec.preBlockC:
			now := time.Now()
			blockWrapper := exec.verifySign(commitEvent)
			exec.logger.WithFields(logrus.Fields{
				"height": commitEvent.Block.BlockHeader.Number,
				"count":  len(commitEvent.Block.Transactions.Transactions),
				"elapse": time.Since(now),
			}).Debug("Verified signature")
			exec.blockC <- blockWrapper
		case <-exec.ctx.Done():
			return
		}
	}
}

func (exec *BlockExecutor) buildTxMerkleTree(txs []pb.Transaction) (*types.Hash, []types.Hash, error) {
	var (
		groupCnt = len(exec.txsExecutor.GetInterchainCounter()) + 1
		wg       = sync.WaitGroup{}
		lock     = sync.Mutex{}
		l2Roots  = make([]types.Hash, 0, groupCnt)
		errorCnt = int32(0)
	)

	wg.Add(groupCnt - 1)
	for addr, txIndexes := range exec.txsExecutor.GetInterchainCounter() {
		go func(addr string, txIndexes []uint64) {
			defer wg.Done()

			txHashes := make([]merkletree.Content, 0, len(txIndexes))
			for _, txIndex := range txIndexes {
				txHashes = append(txHashes, txs[txIndex].GetHash())
			}

			hash, err := calcMerkleRoot(txHashes)
			if err != nil {
				atomic.AddInt32(&errorCnt, 1)
				return
			}

			lock.Lock()
			defer lock.Unlock()
			l2Roots = append(l2Roots, *hash)
		}(addr, txIndexes)
	}

	txHashes := make([]merkletree.Content, 0, len(exec.txsExecutor.GetNormalTxs()))
	for _, txHash := range exec.txsExecutor.GetNormalTxs() {
		txHashes = append(txHashes, txHash)
	}

	hash, err := calcMerkleRoot(txHashes)
	if err != nil {
		atomic.AddInt32(&errorCnt, 1)
	}

	lock.Lock()
	l2Roots = append(l2Roots, *hash)
	lock.Unlock()

	wg.Wait()
	if errorCnt != 0 {
		return nil, nil, fmt.Errorf("build tx merkle tree error")
	}

	sort.Slice(l2Roots, func(i, j int) bool {
		return bytes.Compare(l2Roots[i].Bytes(), l2Roots[j].Bytes()) < 0
	})

	contents := make([]merkletree.Content, 0, groupCnt)
	for _, l2Root := range l2Roots {
		r := l2Root
		contents = append(contents, &r)
	}
	root, err := calcMerkleRoot(contents)
	if err != nil {
		return nil, nil, err
	}

	return root, l2Roots, nil
}

func (exec *BlockExecutor) verifySign(commitEvent *pb.CommitEvent) *BlockWrapper {
	blockWrapper := &BlockWrapper{
		block:     commitEvent.Block,
		invalidTx: make(map[int]agency.InvalidReason),
	}

	if commitEvent.Block.BlockHeader.Number == 1 {
		return blockWrapper
	}

	var (
		wg    sync.WaitGroup
		mutex sync.Mutex
	)
	txs := commitEvent.Block.Transactions.Transactions
	txsLen := len(commitEvent.LocalList)
	wg.Add(len(txs))
	for i, tx := range txs {
		// if the tx is received from api, we will pass the verify.
		if txsLen > i && commitEvent.LocalList[i] {
			wg.Done()
			continue
		}
		go func(i int, tx pb.Transaction) {
			defer wg.Done()
			err := tx.VerifySignature()
			if err != nil {
				mutex.Lock()
				defer mutex.Unlock()
				blockWrapper.invalidTx[i] = agency.InvalidReason(err.Error())
			}
		}(i, tx)
	}
	wg.Wait()

	return blockWrapper
}

func (exec *BlockExecutor) applyTx(index int, tx pb.Transaction, invalidReason agency.InvalidReason, opt *agency.TxOpt) *pb.Receipt {
	receipt := &pb.Receipt{
		Version: tx.GetVersion(),
		TxHash:  tx.GetHash(),
	}
	normalTx := true

	ret, err := exec.applyTransaction(index, tx, invalidReason, opt)
	if err != nil {
		receipt.Status = pb.Receipt_FAILED
		receipt.Ret = []byte(err.Error())
	} else {
		receipt.Status = pb.Receipt_SUCCESS
		receipt.Ret = ret
	}

	events := exec.ledger.Events(tx.GetHash().String())
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
					exec.txsExecutor.AddInterchainCounter(k, v)
				}
				normalTx = false
			}
		}
	}

	if normalTx {
		exec.txsExecutor.AddNormalTx(tx.GetHash())
	}

	return receipt
}

func (exec *BlockExecutor) postBlockEvent(block *pb.Block, interchainMeta *pb.InterchainMeta, txHashList []*types.Hash) {
	go exec.blockFeed.Send(events.ExecutedEvent{
		Block:          block,
		InterchainMeta: interchainMeta,
		TxHashList:     txHashList,
	})
}

func (exec *BlockExecutor) postLogsEvent(receipts []*pb.Receipt) {
	go func() {
		logs := make([]*pb.EvmLog, 0)
		for _, receipt := range receipts {
			logs = append(logs, receipt.EvmLogs...)
		}

		exec.logsFeed.Send(logs)
	}()
}

func (exec *BlockExecutor) applyTransaction(i int, tx pb.Transaction, invalidReason agency.InvalidReason, opt *agency.TxOpt) ([]byte, error) {
	switch tx.(type) {
	case *pb.BxhTransaction:
		bxhTx := tx.(*pb.BxhTransaction)
		return exec.applyBxhTransaction(i, bxhTx, invalidReason, opt)
	case *pb.EthTransaction:
		// TODO

	}

	return nil, fmt.Errorf("unknown tx type")
}

func (exec *BlockExecutor) applyBxhTransaction(i int, tx *pb.BxhTransaction, invalidReason agency.InvalidReason, opt *agency.TxOpt) ([]byte, error) {
	defer exec.ledger.SetNonce(tx.From, tx.GetNonce()+1)

	if invalidReason != "" {
		return nil, fmt.Errorf(string(invalidReason))
	}

	if tx.IsIBTP() {
		ctx := vm.NewContext(tx, uint64(i), nil, exec.ledger, exec.logger)
		instance := boltvm.New(ctx, exec.validationEngine, exec.getContracts(opt))
		return instance.HandleIBTP(tx.GetIBTP())
	}

	if tx.GetPayload() == nil {
		return nil, fmt.Errorf("empty transaction data")
	}

	data := &pb.TransactionData{}
	if err := data.Unmarshal(tx.GetPayload()); err != nil {
		return nil, err
	}

	switch data.Type {
	case pb.TransactionData_NORMAL:
		err := exec.transfer(tx.From, tx.To, data.Amount)
		return nil, err
	default:
		var instance vm.VM
		switch data.VmType {
		case pb.TransactionData_BVM:
			ctx := vm.NewContext(tx, uint64(i), data, exec.ledger, exec.logger)
			instance = boltvm.New(ctx, exec.validationEngine, exec.getContracts(opt))
		case pb.TransactionData_XVM:
			ctx := vm.NewContext(tx, uint64(i), data, exec.ledger, exec.logger)
			imports, err := vmledger.New()
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

		return instance.Run(data.Payload)
	}
}

func (exec *BlockExecutor) clear() {
	exec.ledger.Clear()
}

func (exec *BlockExecutor) transfer(from, to *types.Address, value uint64) error {
	if value == 0 {
		return nil
	}

	fv := exec.ledger.GetBalance(from)
	if fv < value {
		return fmt.Errorf("not sufficient funds for %s", from.String())
	}

	tv := exec.ledger.GetBalance(to)

	exec.ledger.SetBalance(from, fv-value)
	exec.ledger.SetBalance(to, tv+value)

	return nil
}

func (exec *BlockExecutor) calcReceiptMerkleRoot(receipts []*pb.Receipt) (*types.Hash, error) {
	current := time.Now()

	receiptHashes := make([]merkletree.Content, 0, len(receipts))
	for _, receipt := range receipts {
		receiptHashes = append(receiptHashes, receipt.Hash())
	}
	receiptRoot, err := calcMerkleRoot(receiptHashes)
	if err != nil {
		return nil, err
	}

	exec.logger.WithField("time", time.Since(current)).Debug("Calculate receipt merkle roots")

	return receiptRoot, nil
}

func calcMerkleRoot(contents []merkletree.Content) (*types.Hash, error) {
	if len(contents) == 0 {
		return &types.Hash{}, nil
	}

	tree, err := merkletree.NewTree(contents)
	if err != nil {
		return nil, err
	}

	return types.NewHash(tree.MerkleRoot()), nil
}

func (exec *BlockExecutor) getContracts(opt *agency.TxOpt) map[string]agency.Contract {
	if opt != nil && opt.Contracts != nil {
		return opt.Contracts
	}

	return exec.txsExecutor.GetBoltContracts()
}
