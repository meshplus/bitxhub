package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cbergoon/merkletree"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm/vmledger"
	vm1 "github.com/meshplus/eth-kit/evm"
	ledger2 "github.com/meshplus/eth-kit/ledger"
	types2 "github.com/meshplus/eth-kit/types"
	"github.com/sirupsen/logrus"
)

type BlockWrapper struct {
	block     *pb.Block
	invalidTx map[int]agency.InvalidReason
}

func (exec *BlockExecutor) processExecuteEvent(blockWrapper *BlockWrapper) *ledger2.BlockData {
	var txHashList []*types.Hash
	current := time.Now()
	block := blockWrapper.block

	for _, tx := range block.Transactions.Transactions {
		txHashList = append(txHashList, tx.GetHash())
	}

	exec.verifyProofs(blockWrapper)
	exec.evm = newEvm(block.Height(), uint64(block.BlockHeader.Timestamp), exec.evmChainCfg, exec.ledger.StateDB(), exec.ledger.ChainLedger)
	exec.ledger.PrepareBlock(block.BlockHash)
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
	block.BlockHeader.Bloom = ledger.CreateBloom(receipts)

	accounts, journal := exec.ledger.FlushDirtyDataAndComputeJournal()

	exec.logger.WithFields(logrus.Fields{
		"tx_root":      block.BlockHeader.TxRoot.String(),
		"receipt_root": block.BlockHeader.ReceiptRoot.String(),
		//"state_root":   block.BlockHeader.StateRoot.String(),
	}).Debug("block meta")
	calcBlockSize.Observe(float64(block.Size()))
	executeBlockDuration.Observe(float64(time.Since(current)) / float64(time.Second))

	counter := make(map[string]*pb.VerifiedIndexSlice)
	for k, v := range exec.txsExecutor.GetInterchainCounter() {
		counter[k] = &pb.VerifiedIndexSlice{Slice: v}
	}
	interchainMeta := &pb.InterchainMeta{
		Counter: counter,
		L2Roots: l2Roots,
	}

	data := &ledger2.BlockData{
		Block:          block,
		Receipts:       receipts,
		Accounts:       accounts,
		Journal:        journal,
		InterchainMeta: interchainMeta,
		TxHashList:     txHashList,
	}

	now := time.Now()
	exec.ledger.PersistBlockData(data)
	exec.postBlockEvent(data.Block, data.InterchainMeta, data.TxHashList)
	exec.postLogsEvent(data.Receipts)
	exec.logger.WithFields(logrus.Fields{
		"height": data.Block.BlockHeader.Number,
		"hash":   data.Block.BlockHash.String(),
		"count":  len(data.Block.Transactions.Transactions),
		"elapse": time.Since(now),
	}).Info("Persisted block")

	exec.currentHeight = block.BlockHeader.Number
	exec.currentBlockHash = block.BlockHash
	exec.clear()

	return nil
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
		go func(addr string, txIndexes []*pb.VerifiedIndex) {
			defer wg.Done()

			verifiedTx := make([]merkletree.Content, 0, len(txIndexes))
			for _, txIndex := range txIndexes {
				verifiedTx = append(verifiedTx, &pb.VerifiedTx{
					Tx:    txs[txIndex.Index].(*pb.BxhTransaction),
					Valid: txIndex.Valid,
				})
			}

			hash, err := calcMerkleRoot(verifiedTx)
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
	normalTx := true

	receipt := exec.applyTransaction(index, tx, invalidReason, opt)

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
					valid := true
					if receipt.Status == pb.Receipt_FAILED &&
						strings.Contains(string(receipt.Ret), contracts.TargetAppchainNotAvailable) {
						valid = false
					}
					exec.txsExecutor.AddInterchainCounter(k, &pb.VerifiedIndex{
						Index: v,
						Valid: valid,
					})
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

func (exec *BlockExecutor) applyTransaction(i int, tx pb.Transaction, invalidReason agency.InvalidReason, opt *agency.TxOpt) *pb.Receipt {
	receipt := &pb.Receipt{
		Version: tx.GetVersion(),
		TxHash:  tx.GetHash(),
	}
	switch tx.(type) {
	case *pb.BxhTransaction:
		bxhTx := tx.(*pb.BxhTransaction)
		ret, err := exec.applyBxhTransaction(i, bxhTx, invalidReason, opt)
		if err != nil {
			receipt.Status = pb.Receipt_FAILED
			receipt.Ret = []byte(err.Error())
		} else {
			receipt.Status = pb.Receipt_SUCCESS
			receipt.Ret = ret
		}
		return receipt
	case *types2.EthTransaction:
		ethTx := tx.(*types2.EthTransaction)
		return exec.applyEthTransaction(i, ethTx)
	}

	receipt.Status = pb.Receipt_FAILED
	receipt.Ret = []byte(fmt.Errorf("unknown tx type").Error())
	return receipt
}

func (exec *BlockExecutor) applyBxhTransaction(i int, tx *pb.BxhTransaction, invalidReason agency.InvalidReason, opt *agency.TxOpt) ([]byte, error) {
	defer func() {
		exec.ledger.SetNonce(tx.From, tx.GetNonce()+1)
		exec.ledger.Finalise(false)
	}()

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
			var err error
			ctx := vm.NewContext(tx, uint64(i), data, exec.ledger, exec.logger)
			imports := vmledger.New()
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

func (exec *BlockExecutor) applyEthTransaction(i int, tx *types2.EthTransaction) *pb.Receipt {
	receipt := &pb.Receipt{
		Version: tx.GetVersion(),
		TxHash:  tx.GetHash(),
	}

	gp := new(core.GasPool).AddGas(exec.gasLimit)
	msg := ledger.NewMessage(tx)
	statedb := exec.ledger.StateDB()
	statedb.PrepareEVM(common.BytesToHash(tx.GetHash().Bytes()), i)
	snapshot := statedb.Snapshot()
	txContext := vm1.NewEVMTxContext(msg)
	exec.evm.Reset(txContext, exec.ledger.StateDB())
	exec.logger.Debugf("msg gas: %v", msg.Gas())
	result, err := vm1.ApplyMessage(exec.evm, msg, gp)
	if err != nil {
		exec.logger.Errorf("apply msg failed: %s", err.Error())
		statedb.RevertToSnapshot(snapshot)
		receipt.Status = pb.Receipt_FAILED
		receipt.Ret = []byte(err.Error())
		exec.ledger.Finalise(false)
		return receipt
	}
	if result.Failed() {
		exec.logger.Warnf("execute tx failed: %s", result.Err.Error())
		receipt.Status = pb.Receipt_FAILED
		receipt.Ret = append([]byte(result.Err.Error()), result.Revert()...)
	} else {
		receipt.Status = pb.Receipt_SUCCESS
		receipt.Ret = result.Return()
	}

	receipt.TxHash = tx.GetHash()
	receipt.GasUsed = result.UsedGas
	exec.ledger.Finalise(false)
	if msg.To() == nil || bytes.Equal(msg.To().Bytes(), common.Address{}.Bytes()) {
		receipt.ContractAddress = types.NewAddress(crypto.CreateAddress(exec.evm.TxContext.Origin, tx.GetNonce()).Bytes())
	}

	receipt.EvmLogs = exec.ledger.GetLogs(*tx.GetHash())
	receipt.Bloom = ledger.CreateBloom(ledger.EvmReceipts{receipt})
	// receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return receipt
}

func (exec *BlockExecutor) clear() {
	exec.ledger.Clear()
}

func (exec *BlockExecutor) transfer(from, to *types.Address, value uint64) error {
	if value == 0 {
		return nil
	}

	fv := exec.ledger.GetBalance(from).Uint64()
	if fv < value {
		return fmt.Errorf("not sufficient funds for %s", from.String())
	}

	tv := exec.ledger.GetBalance(to).Uint64()

	exec.ledger.SetBalance(from, new(big.Int).SetUint64(fv-value))
	exec.ledger.SetBalance(to, new(big.Int).SetUint64(tv+value))

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

func newEvm(number uint64, timestamp uint64, chainCfg *params.ChainConfig, db ledger2.StateDB, chainLedger ledger2.ChainLedger) *vm1.EVM {
	blkCtx := vm1.NewEVMBlockContext(number, timestamp, db, chainLedger)

	return vm1.NewEVM(blkCtx, vm1.TxContext{}, db, chainCfg, vm1.Config{})
}
