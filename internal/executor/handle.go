package executor

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/cbergoon/merkletree"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/eth-kit/adaptor"
	vm1 "github.com/meshplus/eth-kit/evm"
	ledger2 "github.com/meshplus/eth-kit/ledger"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

const (
	GasNormalTx = 21000
	GasFailedTx = 21000
	GasBVMTx    = 21000 * 10
)

type InvalidReason string

type BlockWrapper struct {
	block     *types.Block
	invalidTx map[int]InvalidReason
}

func (exec *BlockExecutor) applyTransactions(txs []*types.Transaction, invalidTxs map[int]InvalidReason) []*types.Receipt {
	receipts := make([]*types.Receipt, 0, len(txs))

	for i, tx := range txs {
		receipts = append(receipts, exec.applyTransaction(i, tx, invalidTxs[i]))
	}

	exec.logger.Debugf("executor executed %d txs", len(txs))

	return receipts
}

func (exec *BlockExecutor) processExecuteEvent(blockWrapper *BlockWrapper) *ledger.BlockData {
	var txHashList []*types.Hash
	current := time.Now()
	block := blockWrapper.block

	// check executor handle the right block
	if block.BlockHeader.Number != exec.currentHeight+1 {
		exec.logger.WithFields(logrus.Fields{"block height": block.BlockHeader.Number,
			"matchedHeight": exec.currentHeight + 1}).Warning("current block height is not matched")
		return nil
	}

	for _, tx := range block.Transactions {
		txHashList = append(txHashList, tx.GetHash())
	}

	//TODO: CHANGE COINBASE ADDRESSS
	exec.evm = newEvm(block.Height(), uint64(block.BlockHeader.Timestamp), exec.evmChainCfg, exec.ledger.StateLedger, exec.ledger.ChainLedger, exec.admins[0])
	exec.ledger.PrepareBlock(block.BlockHash, block.Height())
	receipts := exec.applyTransactions(block.Transactions, blockWrapper.invalidTx)
	applyTxsDuration.Observe(float64(time.Since(current)) / float64(time.Second))
	exec.logger.WithFields(logrus.Fields{
		"time":  time.Since(current),
		"count": len(block.Transactions),
	}).Debug("Apply transactions elapsed")

	calcMerkleStart := time.Now()
	txRoot, err := exec.buildTxMerkleTree(block.Transactions)
	if err != nil {
		panic(err)
	}

	receiptRoot, err := exec.calcReceiptMerkleRoot(receipts)
	if err != nil {
		panic(err)
	}

	calcMerkleDuration.Observe(float64(time.Since(calcMerkleStart)) / float64(time.Second))

	block.BlockHeader.TxRoot = txRoot
	block.BlockHeader.ReceiptRoot = receiptRoot
	block.BlockHeader.ParentHash = exec.currentBlockHash
	block.BlockHeader.Bloom = ledger.CreateBloom(receipts)
	gasPrice, err := exec.GasPrice()
	if err != nil {
		panic(err)
	}
	block.BlockHeader.GasPrice = gasPrice.Int64()

	accounts, journalHash := exec.ledger.FlushDirtyData()

	block.BlockHeader.StateRoot = journalHash
	block.BlockHash = block.Hash()

	exec.logger.WithFields(logrus.Fields{
		"tx_root":      block.BlockHeader.TxRoot.String(),
		"receipt_root": block.BlockHeader.ReceiptRoot.String(),
		"state_root":   block.BlockHeader.StateRoot.String(),
	}).Debug("block meta")
	calcBlockSize.Observe(float64(block.Size()))
	executeBlockDuration.Observe(float64(time.Since(current)) / float64(time.Second))

	data := &ledger.BlockData{
		Block:      block,
		Receipts:   receipts,
		Accounts:   accounts,
		TxHashList: txHashList,
	}

	exec.logger.WithFields(logrus.Fields{
		"height": blockWrapper.block.BlockHeader.Number,
		"count":  len(blockWrapper.block.Transactions),
		"elapse": time.Since(current),
	}).Info("Executed block")

	now := time.Now()
	exec.ledger.PersistBlockData(data)
	exec.postBlockEvent(data.Block, data.TxHashList)
	exec.postLogsEvent(data.Receipts)
	exec.logger.WithFields(logrus.Fields{
		"gasPrice": data.Block.BlockHeader.GasPrice,
		"height":   data.Block.BlockHeader.Number,
		"hash":     data.Block.BlockHash.String(),
		"count":    len(data.Block.Transactions),
		"elapse":   time.Since(now),
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
				"count":  len(commitEvent.Block.Transactions),
				"elapse": time.Since(now),
			}).Debug("Verified signature")
			exec.blockC <- blockWrapper
		case <-exec.ctx.Done():
			return
		}
	}
}

func (exec *BlockExecutor) buildTxMerkleTree(txs []*types.Transaction) (*types.Hash, error) {
	hash, err := calcMerkleRoot(lo.Map(txs, func(item *types.Transaction, index int) merkletree.Content {
		return item.GetHash()
	}))
	if err != nil {
		return nil, err
	}

	return hash, nil
}

func (exec *BlockExecutor) verifySign(commitEvent *types.CommitEvent) *BlockWrapper {
	blockWrapper := &BlockWrapper{
		block:     commitEvent.Block,
		invalidTx: make(map[int]InvalidReason),
	}

	if commitEvent.Block.BlockHeader.Number == 1 {
		return blockWrapper
	}

	var (
		wg    sync.WaitGroup
		mutex sync.Mutex
	)
	txs := commitEvent.Block.Transactions
	txsLen := len(commitEvent.LocalList)
	wg.Add(len(txs))
	for i, tx := range txs {
		// if the tx is received from api, we will pass the verify.
		if txsLen > i && commitEvent.LocalList[i] {
			wg.Done()
			continue
		}
		go func(i int, tx *types.Transaction) {
			defer wg.Done()
			err := tx.VerifySignature()
			if err != nil {
				mutex.Lock()
				defer mutex.Unlock()
				blockWrapper.invalidTx[i] = InvalidReason(err.Error())
			}
		}(i, tx)
	}
	wg.Wait()

	return blockWrapper
}

func (exec *BlockExecutor) postBlockEvent(block *types.Block, txHashList []*types.Hash) {
	go exec.blockFeed.Send(events.ExecutedEvent{
		Block:      block,
		TxHashList: txHashList,
	})
	go exec.blockFeedForRemote.Send(events.ExecutedEvent{
		Block:      block,
		TxHashList: txHashList,
	})
}

func (exec *BlockExecutor) postLogsEvent(receipts []*types.Receipt) {
	go func() {
		logs := make([]*types.EvmLog, 0)
		for _, receipt := range receipts {
			logs = append(logs, receipt.EvmLogs...)
		}

		exec.logsFeed.Send(logs)
	}()
}

// TODO: process invalidReason
func (exec *BlockExecutor) applyTransaction(i int, tx *types.Transaction, _ InvalidReason) *types.Receipt {
	defer func() {
		exec.ledger.SetNonce(tx.GetFrom(), tx.GetNonce()+1)
		exec.ledger.Finalise(true)
	}()

	exec.logger.Debugf("tx gas: %v", tx.GetGas())
	exec.logger.Debugf("tx gas price: %v", tx.GetGasPrice())

	exec.ledger.SetTxContext(tx.GetHash(), i)

	receipt := exec.applyEthTransaction(i, tx)
	if err := exec.payGasFee(tx, receipt.GasUsed); err != nil {
		receipt.Ret = []byte(err.Error())
		exec.payLeftAsGasFee(tx)
	}

	return receipt
}

func (exec *BlockExecutor) applyEthTransaction(_ int, tx *types.Transaction) *types.Receipt {
	receipt := &types.Receipt{
		Version: tx.GetVersion(),
		TxHash:  tx.GetHash(),
	}

	gp := new(vm1.GasPool).AddGas(exec.gasLimit)
	msg := adaptor.TransactionToMessage(tx)
	statedb := exec.ledger.StateLedger
	txContext := vm1.NewEVMTxContext(msg)
	snapshot := statedb.Snapshot()
	exec.evm.Reset(txContext, exec.ledger.StateLedger)
	exec.logger.Debugf("msg gas: %v", msg.GasPrice)
	result, err := vm1.ApplyMessage(exec.evm, msg, gp)
	if err != nil {
		exec.logger.Errorf("apply msg failed: %s", err.Error())
		statedb.RevertToSnapshot(snapshot)
		receipt.Status = types.ReceiptFAILED
		receipt.Ret = []byte(err.Error())
		exec.ledger.Finalise(true)
		return receipt
	}
	if result.Failed() {
		exec.logger.Warnf("execute tx failed: %s", result.Err.Error())
		receipt.Status = types.ReceiptFAILED
		receipt.Ret = []byte(result.Err.Error())
		if strings.HasPrefix(result.Err.Error(), vm1.ErrExecutionReverted.Error()) {
			receipt.Ret = append(receipt.Ret, common.CopyBytes(result.ReturnData)...)
		}
	} else {
		receipt.Status = types.ReceiptSUCCESS
		receipt.Ret = result.Return()
	}

	receipt.TxHash = tx.GetHash()
	receipt.GasUsed = result.UsedGas
	exec.ledger.Finalise(true)
	if msg.To == nil || bytes.Equal(msg.To.Bytes(), common.Address{}.Bytes()) {
		receipt.ContractAddress = types.NewAddress(crypto.CreateAddress(exec.evm.TxContext.Origin, tx.GetNonce()).Bytes())
	}

	receipt.EvmLogs = exec.ledger.GetLogs(*tx.GetHash())
	receipt.Bloom = ledger.CreateBloom(ledger.EvmReceipts{receipt})
	return receipt
}

func (exec *BlockExecutor) clear() {
	exec.ledger.Clear()
}

func (exec *BlockExecutor) calcReceiptMerkleRoot(receipts []*types.Receipt) (*types.Hash, error) {
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

func newEvm(number uint64, timestamp uint64, chainCfg *params.ChainConfig, db ledger2.StateLedger, chainLedger ledger2.ChainLedger, admin string) *vm1.EVM {
	blkCtx := vm1.NewEVMBlockContext(number, timestamp, db, chainLedger, admin)

	return vm1.NewEVM(blkCtx, vm1.TxContext{}, db, chainCfg, vm1.Config{})
}

func (exec *BlockExecutor) GetEvm(txCtx vm1.TxContext, vmConfig vm1.Config) *vm1.EVM {
	var blkCtx vm1.BlockContext
	meta := exec.ledger.GetChainMeta()
	block, err := exec.ledger.GetBlock(meta.Height)
	if err != nil {
		exec.logger.Errorf("fail to get block at %d: %v", meta.Height, err.Error())
		return nil
	}
	blkCtx = vm1.NewEVMBlockContext(meta.Height, uint64(block.BlockHeader.Timestamp), exec.ledger.StateLedger, exec.ledger.ChainLedger, exec.admins[0])
	return vm1.NewEVM(blkCtx, txCtx, exec.ledger.StateLedger, exec.evmChainCfg, vmConfig)
}

func (exec *BlockExecutor) payGasFee(tx *types.Transaction, gasUsed uint64) error {
	gasPrice, err := exec.GasPrice()
	if err != nil {
		return errors.Wrap(err, "pay gas fee failed")
	}
	fees := new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), gasPrice)
	have := exec.ledger.GetBalance(tx.GetFrom())
	if have.Cmp(fees) < 0 {
		return fmt.Errorf("insufficeient balance: address %v have %v want %v", tx.GetFrom().String(), have, fees)
	}
	exec.ledger.SetBalance(tx.GetFrom(), new(big.Int).Sub(have, fees))
	exec.payAdmins(fees)
	return nil
}

func (exec *BlockExecutor) payLeftAsGasFee(tx *types.Transaction) {
	have := exec.ledger.GetBalance(tx.GetFrom())
	exec.ledger.SetBalance(tx.GetFrom(), big.NewInt(0))
	exec.payAdmins(have)
}

func (exec *BlockExecutor) payAdmins(fees *big.Int) {
	fee := new(big.Int).Div(fees, big.NewInt(int64(len(exec.admins))))
	for _, admin := range exec.admins {
		addr := types.NewAddressByStr(admin)
		balance := exec.ledger.GetBalance(addr)
		exec.ledger.SetBalance(addr, new(big.Int).Add(balance, fee))
	}
}
