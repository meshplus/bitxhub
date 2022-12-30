package executor

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/pkg/utils"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm/vmledger"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/cbergoon/merkletree"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/meshplus/bitxhub-core/agency"
	servicemgr "github.com/meshplus/bitxhub-core/service-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	vm1 "github.com/meshplus/eth-kit/evm"
	ledger2 "github.com/meshplus/eth-kit/ledger"
	types2 "github.com/meshplus/eth-kit/types"
	"github.com/sirupsen/logrus"
)

const (
	GasNormalTx = 21000
	GasFailedTx = 21000
	GasBVMTx    = 21000 * 10
	repeatBlock = "new block is the same as the block in local ledger"
	parallel    = "parallel"
	simple      = "simple"
)

type BlockWrapper struct {
	block     *pb.Block
	invalidTx map[int]agency.InvalidReason
}

func (exec *BlockExecutor) rollbackBlocks(newBlock *pb.Block) error {
	var (
		oldBlock *pb.Block
		err      error
	)
	if err = retry.Retry(func(attempt uint) error {
		oldBlock, err = exec.ledger.GetBlock(newBlock.Height(), false)
		if err != nil {
			exec.logger.WithFields(logrus.Fields{
				"height": newBlock.Height(),
				"err":    err.Error(),
			}).Errorf("get block from ledger error")
			return err
		}
		return nil
	}, strategy.Limit(5), strategy.Wait(1*time.Second),
	); err != nil {
		exec.logger.WithFields(logrus.Fields{
			"height": newBlock.Height(),
			"err":    err.Error(),
		}).Errorf("retry error")
		panic(err)
	}
	// consensus ensure newBlock is approved by quorum nodes
	if oldBlock.BlockHash.String() != newBlock.Hash().String() {
		// query last checked block for generating right parent blockHash
		lastCheckedBlock, err := exec.ledger.GetBlock(newBlock.Height()-1, false)
		if err != nil {
			exec.logger.WithFields(logrus.Fields{
				"height": lastCheckedBlock.Height(),
				"err":    err.Error(),
			}).Errorf("get last checked block from ledger error")
			return err
		}

		// rollback from stateLedger、chainLedger and blockFile
		err = exec.ledger.Rollback(oldBlock.Height() - 1)
		if err != nil {
			exec.logger.WithFields(logrus.Fields{
				"begin height": oldBlock.Height(),
				"end height":   exec.currentHeight,
				"err":          err.Error(),
			}).Errorf("rollback block error")
			return err
		}
		exec.currentHeight = lastCheckedBlock.Height()
		exec.currentBlockHash = lastCheckedBlock.BlockHash
		exec.logger.WithFields(logrus.Fields{
			"begin height":   oldBlock.Height(),
			"current height": exec.currentHeight,
		}).Info("rollback block end")
	} else {
		return fmt.Errorf("does not need to be repeated executor:%s", repeatBlock)
	}
	return nil
}

func (exec *BlockExecutor) processExecuteEvent(blockWrapper *BlockWrapper) *ledger.BlockData {
	var txHashList []*types.Hash
	current := time.Now()
	block := blockWrapper.block

	// check executor handle the right block
	if block.BlockHeader.Number != exec.currentHeight+1 {
		exec.logger.WithFields(logrus.Fields{"block height": block.BlockHeader.Number,
			"matchedHeight": exec.currentHeight + 1}).Warning("current block height is not matched")
		err := exec.rollbackBlocks(block)
		if err != nil {
			if strings.Contains(err.Error(), repeatBlock) {
				return nil
			}
			panic(err)
		}
	}

	for _, tx := range block.Transactions.Transactions {
		txHashList = append(txHashList, tx.GetHash())
	}

	exec.verifyProofs(blockWrapper)
	exec.logger.WithFields(logrus.Fields{
		"time":  time.Since(current),
		"type":  exec.config.Executor.ProofType,
		"count": len(block.Transactions.Transactions),
	}).Infof("verify proof elapsed")

	exec.evm = newEvm(block.Height(), uint64(block.BlockHeader.Timestamp), exec.evmChainCfg, exec.ledger.StateLedger,
		exec.ledger.ChainLedger, exec.admins[0], exec.evmMaxSize)

	exec.ledger.PrepareBlock(block.BlockHash, block.Height())
	current2 := time.Now()
	receipts := exec.txsExecutor.ApplyTransactions(block.Transactions.Transactions, blockWrapper.invalidTx)

	//for i, receipt := range receipts {
	//	exec.logger.WithFields(logrus.Fields{
	//		"receipt": receipt,
	//		"height":  block.Height(),
	//	}).Infof("get receipt %d", i)
	//}

	applyTxsDuration.Observe(float64(time.Since(current2)) / float64(time.Second))
	exec.logger.WithFields(logrus.Fields{
		"time":  time.Since(current2),
		"count": len(block.Transactions.Transactions),
	}).Info("Apply transactions elapsed")

	calcMerkleStart := time.Now()
	txRoot, err := exec.buildTxMerkleTree(block.Transactions.Transactions)
	if err != nil {
		panic(err)
	}

	receiptRoot, err := exec.calcReceiptMerkleRoot(receipts)
	if err != nil {
		panic(err)
	}

	calcMerkleDuration.Observe(float64(time.Since(calcMerkleStart)) / float64(time.Second))

	invalidTxHashMap, recordFailTxHashMap, err := exec.filterValidTx(receipts)
	if err != nil {
		exec.logger.Errorf("filterValidTx err: %s", err)
	}
	// this block is not in ledger
	txList := blockWrapper.block.Transactions.Transactions
	bxhId := strconv.FormatUint(exec.config.ChainID, 10)
	err = exec.setTimeoutList(block.BlockHeader.Number, txList, invalidTxHashMap, recordFailTxHashMap, bxhId)
	if err != nil {
		exec.logger.Errorf("setTimeoutList err: %s", err)
	}

	timeoutIBTPsMap, err := exec.getTimeoutIBTPsMap(block.BlockHeader.Number)
	if err != nil {
		exec.logger.Errorf("getTimeoutIBTPsMap err: %s", err)
	}

	var timeoutL2Roots []types.Hash
	timeoutCounter := make(map[string]*pb.StringSlice)
	for from, list := range timeoutIBTPsMap {
		root, err := exec.calcTimeoutL2Root(list)
		if err != nil {
			panic(err)
		}
		timeoutCounter[from] = &pb.StringSlice{Slice: list}
		timeoutL2Roots = append(timeoutL2Roots, root)
	}

	timeoutRoots := make([]merkletree.Content, 0, len(timeoutL2Roots))
	sort.Slice(timeoutL2Roots, func(i, j int) bool {
		return bytes.Compare(timeoutL2Roots[i].Bytes(), timeoutL2Roots[j].Bytes()) < 0
	})
	for _, root := range timeoutL2Roots {
		r := root
		timeoutRoots = append(timeoutRoots, &r)
	}
	timeoutRoot, err := calcMerkleRoot(timeoutRoots)
	if err != nil {
		panic(err)
	}

	multiTxIBTPsMap, err := exec.getMultiTxIBTPsMap(exec.currentHeight + 1)
	if err != nil {
		panic(err)
	}
	multiTxCounter := make(map[string]*pb.StringSlice)
	for from, list := range multiTxIBTPsMap {
		multiTxCounter[from] = &pb.StringSlice{Slice: list}
	}

	block.BlockHeader.TxRoot = txRoot
	block.BlockHeader.ReceiptRoot = receiptRoot
	block.BlockHeader.ParentHash = exec.currentBlockHash
	block.BlockHeader.Bloom = ledger.CreateBloom(receipts)
	block.BlockHeader.TimeoutRoot = timeoutRoot

	err = exec.setTimeoutRollback(block.BlockHeader.Number)
	if err != nil {
		exec.logger.Errorf("setTimeoutRollback err: %s", err)
	}
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

	counter := make(map[string]*pb.VerifiedIndexSlice)
	for k, v := range exec.txsExecutor.GetInterchainCounter() {
		counter[k] = &pb.VerifiedIndexSlice{Slice: v}
	}
	interchainMeta := &pb.InterchainMeta{
		Counter:        counter,
		TimeoutCounter: timeoutCounter,
		TimeoutL2Roots: timeoutL2Roots,
		MultiTxCounter: multiTxCounter,
	}

	data := &ledger.BlockData{
		Block:          block,
		Receipts:       receipts,
		Accounts:       accounts,
		InterchainMeta: interchainMeta,
		TxHashList:     txHashList,
	}

	now := time.Now()
	exec.ledger.PersistBlockData(data)
	exec.logger.WithFields(logrus.Fields{
		"height": data.Block.BlockHeader.Number,
		"hash":   data.Block.BlockHash.String(),
		"count":  len(data.Block.Transactions.Transactions),
		"elapse": time.Since(now),
	}).Info("Persisted block")

	exec.currentHeight = block.BlockHeader.Number
	exec.currentBlockHash = block.BlockHash
	exec.postBlockEvent(data.Block, data.InterchainMeta, data.TxHashList)

	exec.logger.WithFields(logrus.Fields{
		"height": blockWrapper.block.BlockHeader.Number,
		"count":  len(blockWrapper.block.Transactions.Transactions),
		"elapse": time.Since(current),
	}).Info("Executed and Persisted block")
	exec.postLogsEvent(data.Receipts)
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

func (exec *BlockExecutor) buildTxMerkleTree(txs []pb.Transaction) (*types.Hash, error) {
	txHashes := make([]merkletree.Content, 0, len(txs))
	for _, tx := range txs {
		txHashes = append(txHashes, tx.GetHash())
	}

	root, err := calcMerkleRoot(txHashes)
	if err != nil {
		return nil, fmt.Errorf("build tx merkle tree error")
	}

	return root, nil
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

	evs := exec.ledger.Events(tx.GetHash().String())
	if len(evs) != 0 {
		receipt.Events = evs

		auditDataUpdate := false
		relatedChainIDList := map[string][]byte{}
		relatedNodeIDList := map[string][]byte{}

		for _, ev := range evs {
			switch ev.EventType {
			case pb.Event_INTERCHAIN:
				m := make(map[string]*pb.EventWrapper)
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
						Index:   v.Index,
						Valid:   valid,
						IsBatch: v.IsBatch,
					})
				}
				normalTx = false
			case pb.Event_NODEMGR:
				nodeEvent := events.NodeEvent{}
				err := json.Unmarshal(ev.Data, &nodeEvent)
				if err != nil {
					panic(err)
				}
				exec.postNodeEvent(nodeEvent)

			case pb.Event_AUDIT_PROPOSAL,
				pb.Event_AUDIT_APPCHAIN,
				pb.Event_AUDIT_RULE,
				pb.Event_AUDIT_SERVICE,
				pb.Event_AUDIT_NODE,
				pb.Event_AUDIT_ROLE,
				pb.Event_AUDIT_INTERCHAIN,
				pb.Event_AUDIT_DAPP:
				auditDataUpdate = true
				auditRelatedObjInfo := pb.AuditRelatedObjInfo{}
				err := json.Unmarshal(ev.Data, &auditRelatedObjInfo)
				if err != nil {
					panic(err)
				}
				for chainID := range auditRelatedObjInfo.RelatedChainIDList {
					relatedChainIDList[chainID] = []byte{}
				}
				for nodeID := range auditRelatedObjInfo.RelatedNodeIDList {
					relatedNodeIDList[nodeID] = []byte{}
				}
			case pb.Event_SERVICE:
				serviceInfo := &servicemgr.Service{}
				err := json.Unmarshal(ev.Data, &serviceInfo)
				if err != nil {
					panic(err)
				}
				chainServiceID := fmt.Sprintf("%s:%s", serviceInfo.ChainID, serviceInfo.ServiceID)
				exec.logger.WithFields(logrus.Fields{"key": chainServiceID, "service": serviceInfo}).Debug("record service change in cache")
				exec.serviceCache.Store(chainServiceID, serviceInfo)
			}
		}
		if auditDataUpdate {
			exec.postAuditEvent(&pb.AuditTxInfo{
				Tx:                 tx.(*pb.BxhTransaction),
				Rec:                receipt,
				BlockHeight:        exec.currentHeight,
				RelatedChainIDList: relatedChainIDList,
				RelatedNodeIDList:  relatedNodeIDList,
			})
			utils.AddAuditPermitBloom(receipt.Bloom, relatedChainIDList, relatedNodeIDList)
		}
	}

	if normalTx {
		exec.txsExecutor.AddNormalTx(tx.GetHash())
	}

	return receipt
}

func (exec *BlockExecutor) postAuditEvent(auditTxInfo *pb.AuditTxInfo) {
	go exec.auditFeed.Send(auditTxInfo)
}

func (exec *BlockExecutor) postNodeEvent(event events.NodeEvent) {
	go exec.nodeFeed.Send(event)
}

func (exec *BlockExecutor) postBlockEvent(block *pb.Block, interchainMeta *pb.InterchainMeta, txHashList []*types.Hash) {
	go exec.blockFeed.Send(events.ExecutedEvent{
		Block:          block,
		InterchainMeta: interchainMeta,
		TxHashList:     txHashList,
	})
	go exec.blockFeedForRemote.Send(events.ExecutedEvent{
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
	var changer *ledger.ChangeInstance
	if exec.supportParallel && tx.IsIBTP() {
		// make sure each tx has an instance to record ledger changed
		changer = exec.ledger.StateLedger.(*ledger.SimpleLedger).GetInstance()
		opt.Changer = changer
		defer func() {
			exec.ledger.SetNonce(tx.GetFrom(), tx.GetNonce()+1)
			// release change to instance pool
			exec.ledger.StateLedger.(*ledger.SimpleLedger).ReleaseChangeInstance(changer)
		}()
	} else {
		defer func() {
			exec.ledger.SetNonce(tx.GetFrom(), tx.GetNonce()+1)
			exec.ledger.Finalise(true)
		}()
	}

	receipt := &pb.Receipt{
		Version: tx.GetVersion(),
		TxHash:  tx.GetHash(),
	}

	exec.ledger.PrepareEVM(common.BytesToHash(tx.GetHash().Bytes()), i)
	var snapshot int
	switch tx.(type) {
	case *pb.BxhTransaction:
		bxhTx := tx.(*pb.BxhTransaction)
		if exec.supportParallel && tx.IsIBTP() {
			snapshot = exec.ledger.StateLedger.(*ledger.SimpleLedger).SnapshotForParallel(changer)
		} else {
			snapshot = exec.ledger.Snapshot()
		}
		ret, gasUsed, err := exec.applyBxhTransaction(i, bxhTx, invalidReason, opt)
		if err != nil {
			receipt.Status = pb.Receipt_FAILED
			receipt.Ret = []byte(err.Error())
		} else {
			// internal invoke evm
			receipt.EvmLogs = exec.ledger.GetLogs(*tx.GetHash())
			receipt.Status = pb.Receipt_SUCCESS
			if string(ret) == "begin_failure" {
				receipt.TxStatus = pb.TransactionStatus_BEGIN_FAILURE
			}
			receipt.Ret = ret
		}
		receipt.Bloom = ledger.CreateBloom(ledger.EvmReceipts{receipt})
		receipt.GasUsed = gasUsed

		if err := exec.payGasFee(tx, gasUsed); err != nil {
			if !exec.supportParallel || !tx.IsIBTP() {
				exec.ledger.RevertToSnapshot(snapshot)
			} else {
				// revert to prev state, need input unique change instance
				exec.ledger.StateLedger.(*ledger.SimpleLedger).RevertToSnapshotForParallel(snapshot, changer)
			}
			receipt.Status = pb.Receipt_FAILED
			receipt.Ret = []byte(err.Error())
			exec.payLeftAsGasFee(tx)
		}
		return receipt
	case *types2.EthTransaction:
		ethTx := tx.(*types2.EthTransaction)
		receipt := exec.applyEthTransaction(i, ethTx)
		exec.evmInterchain(i, ethTx, receipt)

		return receipt
	}

	receipt.Status = pb.Receipt_FAILED
	receipt.GasUsed = GasFailedTx
	receipt.Ret = []byte(fmt.Errorf("unknown tx type").Error())
	if err := exec.payGasFee(tx, receipt.GasUsed); err != nil {
		receipt.Ret = []byte(err.Error())
		exec.payLeftAsGasFee(tx)
	}

	return receipt
}

func (exec *BlockExecutor) applyBxhTransaction(i int, tx *pb.BxhTransaction, invalidReason agency.InvalidReason, opt *agency.TxOpt) ([]byte, uint64, error) {
	if invalidReason != "" {
		return nil, GasFailedTx, fmt.Errorf(string(invalidReason))
	}

	if tx.IsIBTP() {
		ctx := vm.NewContext(tx, uint64(i), nil, exec.currentHeight+1, exec.ledger, exec.logger,
			exec.config.EnableAudit, exec.getChanger(opt))

		instance := boltvm.New(ctx, exec.validationEngine, exec.evm, exec.getContracts(opt))
		ret, err := instance.HandleIBTP(tx.GetIBTP(), exec.serviceCache)
		return ret, GasBVMTx, err
	}

	if tx.GetPayload() == nil {
		return nil, GasFailedTx, fmt.Errorf("empty transaction data")
	}

	data := &pb.TransactionData{}
	if err := data.Unmarshal(tx.GetPayload()); err != nil {
		return nil, GasFailedTx, err
	}

	snapshot := exec.ledger.Snapshot()
	switch data.Type {
	case pb.TransactionData_NORMAL:
		val, ok := new(big.Int).SetString(data.Amount, 10)
		if !ok {
			val = big.NewInt(0)
		}
		err := exec.transfer(tx.From, tx.To, val)
		if err != nil {
			exec.ledger.RevertToSnapshot(snapshot)
		}
		return nil, GasNormalTx, err
	default:
		var instance vm.VM
		var gasUsed uint64
		switch data.VmType {
		case pb.TransactionData_BVM:
			ctx := vm.NewContext(tx, uint64(i), data, exec.currentHeight, exec.ledger, exec.logger, exec.config.EnableAudit, nil)
			instance = boltvm.New(ctx, exec.validationEngine, exec.evm, exec.getContracts(opt))
			gasUsed = GasBVMTx
		case pb.TransactionData_XVM:
			var err error
			ctx := vm.NewContext(tx, uint64(i), data, exec.currentHeight, exec.ledger, exec.logger, exec.config.EnableAudit, nil)
			context := make(map[string]interface{})
			store := wasm.NewStore()
			libs := vmledger.NewLedgerWasmLibs(context, store)
			instance, err = wasm.New(ctx, libs, context, store)
			if err != nil {
				return nil, GasFailedTx, err
			}
		default:
			return nil, GasFailedTx, fmt.Errorf("wrong vm type")
		}

		ret, gasRunUsed, err := instance.Run(data.Payload, exec.gasLimit)
		if err != nil {
			exec.ledger.RevertToSnapshot(snapshot)
		}
		gasUsed += gasRunUsed
		exec.logger.WithField("gasUsed", gasUsed).Info("Bxh transaction")
		return ret, gasUsed, err
	}
}

func (exec *BlockExecutor) applyEthTransaction(_ int, tx *types2.EthTransaction) *pb.Receipt {
	receipt := &pb.Receipt{
		Version: tx.GetVersion(),
		TxHash:  tx.GetHash(),
	}

	gp := new(core.GasPool).AddGas(exec.gasLimit)
	msg := tx.ToMessage()
	statedb := exec.ledger.StateLedger
	txContext := vm1.NewEVMTxContext(msg)
	snapshot := statedb.Snapshot()
	exec.evm.Reset(txContext, exec.ledger.StateLedger)
	exec.logger.Debugf("msg gas: %v", msg.Gas())
	result, err := vm1.ApplyMessage(exec.evm, msg, gp)
	if err != nil {
		exec.logger.Errorf("apply msg failed: %s", err.Error())
		statedb.RevertToSnapshot(snapshot)
		receipt.Status = pb.Receipt_FAILED
		receipt.Ret = []byte(err.Error())
		exec.ledger.Finalise(true)
		return receipt
	}
	if result.Failed() {
		exec.logger.Warnf("execute tx failed: %s", result.Err.Error())
		receipt.Status = pb.Receipt_FAILED
		if strings.HasPrefix(result.Err.Error(), vm1.ErrExecutionReverted.Error()) {
			receipt.Ret = append([]byte(result.Err.Error()), common.CopyBytes(result.ReturnData)...)
		} else {
			receipt.Ret = []byte(result.Err.Error())
		}
	} else {
		receipt.Status = pb.Receipt_SUCCESS
		receipt.Ret = result.Return()
	}

	receipt.TxHash = tx.GetHash()
	receipt.GasUsed = result.UsedGas
	exec.ledger.Finalise(true)
	if msg.To() == nil || bytes.Equal(msg.To().Bytes(), common.Address{}.Bytes()) {
		receipt.ContractAddress = types.NewAddress(crypto.CreateAddress(exec.evm.TxContext.Origin, tx.GetNonce()).Bytes())
	}

	receipt.EvmLogs = exec.ledger.GetLogs(*tx.GetHash())
	receipt.Bloom = ledger.CreateBloom(ledger.EvmReceipts{receipt})
	// receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return receipt
}

func (exec *BlockExecutor) evmInterchain(i int, tx *types2.EthTransaction, receipt *pb.Receipt) {
	if receipt.Status == pb.Receipt_FAILED {
		return
	}

	for _, log := range receipt.EvmLogs {
		if strings.EqualFold(log.Address.String(), constant.InterBrokerContractAddr.String()) {
			ctx := vm.NewContext(tx, uint64(i), nil, exec.currentHeight, exec.ledger, exec.logger, exec.config.EnableAudit, nil)
			instance := boltvm.New(ctx, exec.validationEngine, exec.evm, exec.registerBoltContracts())

			ret, _, err := instance.InvokeBVM(constant.InterBrokerContractAddr.String(), log.Data)
			if err != nil {
				receipt.Status = pb.Receipt_FAILED
			}
			receipt.Ret = ret
			return
		}
	}

}

func (exec *BlockExecutor) clear() {
	exec.ledger.Clear()
}

func (exec *BlockExecutor) transfer(from, to *types.Address, value *big.Int) error {
	if value == nil || value.Cmp(big.NewInt(0)) == 0 {
		return nil
	}

	fv := exec.ledger.GetBalance(from)
	if fv.Cmp(value) == -1 {
		return fmt.Errorf("not sufficient funds for %s", from.String())
	}

	tv := exec.ledger.GetBalance(to)

	exec.ledger.SetBalance(from, new(big.Int).Sub(fv, value))
	exec.ledger.SetBalance(to, new(big.Int).Add(tv, value))

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

func (exec *BlockExecutor) getChanger(opt *agency.TxOpt) *ledger.ChangeInstance {
	if opt != nil && opt.Changer != nil {
		return opt.Changer.(*ledger.ChangeInstance)
	}

	return nil
}

func newEvm(number uint64, timestamp uint64, chainCfg *params.ChainConfig, db ledger2.StateLedger, chainLedger ledger2.ChainLedger, admin string, maxCodeSize uint64) *vm1.EVM {
	blkCtx := vm1.NewEVMBlockContext(number, timestamp, maxCodeSize, db, chainLedger, admin)

	return vm1.NewEVM(blkCtx, vm1.TxContext{}, db, chainCfg, vm1.Config{})
}

func (exec *BlockExecutor) payGasFee(tx pb.Transaction, gasUsed uint64) error {
	fees := new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), exec.bxhGasPrice)
	have := exec.ledger.GetBalance(tx.GetFrom())
	if have.Cmp(fees) < 0 {
		return fmt.Errorf("insufficient balance: address %v have %v want %v", tx.GetFrom().String(), have, fees)
	}
	exec.ledger.SetBalance(tx.GetFrom(), new(big.Int).Sub(have, fees))
	exec.payAdmins(fees)
	return nil
}

func (exec *BlockExecutor) payLeftAsGasFee(tx pb.Transaction) {
	have := exec.ledger.GetBalance(tx.GetFrom())
	exec.ledger.SetBalance(tx.GetFrom(), big.NewInt(0))
	exec.payAdmins(have)
}

func (exec *BlockExecutor) payAdmins(fees *big.Int) {
	fee := new(big.Int).Div(fees, big.NewInt(int64(len(exec.admins))))
	// lock when pay admin gas parallel
	if exec.supportParallel {
		exec.gasLock.Lock()
		defer exec.gasLock.Unlock()
	}
	for _, admin := range exec.admins {
		addr := types.NewAddressByStr(admin)
		balance := exec.ledger.GetBalance(addr)
		exec.ledger.SetBalance(addr, new(big.Int).Add(balance, fee))
	}
}

func (exec *BlockExecutor) setTimeoutRollback(height uint64) error {
	list, err := exec.getTimeoutList(height)
	if err != nil {
		return err
	}

	for _, id := range list {
		if exec.isGlobalID(id) {
			if err := exec.setGlobalTxStatus(id, pb.TransactionStatus_BEGIN_ROLLBACK); err != nil {
				return fmt.Errorf("set global tx status of id %s: %w", id, err)
			}
		} else {
			record := pb.TransactionRecord{
				Height: height,
				Status: pb.TransactionStatus_BEGIN_ROLLBACK,
			}

			if err := exec.setTxRecord(id, record); err != nil {
				return fmt.Errorf("set tx status of id %s: %w", id, err)
			}
		}
	}

	// TODO: Is it necessary remove timeout ibtp from the timeoutList after handle it？

	return nil
}

func (exec *BlockExecutor) getTimeoutList(height uint64) ([]string, error) {
	ok, val := exec.ledger.GetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.TimeoutKey(height)))
	if !ok {
		return nil, nil
	}

	var list []string
	list = strings.Split(string(val), ",")
	if list[0] == "" {
		return nil, nil
	}
	return list, nil
}

func (exec *BlockExecutor) setTimeoutList(height uint64, txList []pb.Transaction, invalidMap map[string]bool, failMap map[string]bool, bxhId string) error {
	addTimeoutListMap := make(map[uint64]string, len(txList))
	removeTimeoutListMap := make(map[uint64]string, len(txList))

	for _, tx := range txList {
		switch tx.(type) {
		case *types2.EthTransaction:
			continue
		case *pb.BxhTransaction:
			if !tx.IsIBTP() {
				continue
			}
			ibtp := tx.GetIBTP()

			// if bxh is destAppchain, needn't add into timeoutList
			if exec.isDstChainFromBxh(ibtp.To, bxhId) {
				continue
			}
			// handle multiIbtp in transaction manager
			if ibtp.Group != nil {
				continue
			}

			// invalidTx needn't add into timeoutList
			// tx of status is begin_fail needn't add into timeoutList
			if ok := invalidMap[tx.GetHash().String()]; ok {
				continue
			} else if ok = failMap[tx.GetHash().String()]; ok {
				continue
			}

			txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)

			if pb.IBTP_REQUEST == ibtp.Category() {
				// record timeout height
				var timeoutHeight uint64
				// timeoutHeight illegal, not record it
				if ibtp.TimeoutHeight <= 0 || uint64(ibtp.TimeoutHeight) >= math.MaxUint64-height {
					continue
				} else {
					timeoutHeight = height + uint64(ibtp.TimeoutHeight)
				}
				str, ok := addTimeoutListMap[timeoutHeight]
				if !ok {
					addTimeoutListMap[timeoutHeight] = txId
				} else {
					addTimeoutListMap[timeoutHeight] = exec.writeToStr(str, txId)
				}
			}
			if pb.IBTP_RESPONSE == ibtp.Category() {
				ok, val := exec.ledger.GetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.TxInfoKey(txId)))
				if !ok {
					if ok, val = exec.ledger.GetState(constant.TransactionMgrContractAddr.Address(), []byte(txId)); !ok {
						err := fmt.Errorf("can't read record from leadger")
						return err
					}
					// handle multiIbtp in transaction manager
					continue

				}
				record := pb.TransactionRecord{}
				if err := record.Unmarshal(val); err != nil {
					return err
				}

				// The fail receipt had already rollback, don't execute timeoutRollback
				if ibtp.Type == pb.IBTP_RECEIPT_FAILURE && record.Status == pb.TransactionStatus_FAILURE {
					continue
				}
				str, ok := removeTimeoutListMap[record.Height]
				if !ok {
					removeTimeoutListMap[record.Height] = txId
				} else {
					removeTimeoutListMap[record.Height] = exec.writeToStr(str, txId)
				}
			}
		}

	}
	for timeoutHeight, txidList := range addTimeoutListMap {
		newStr := exec.addTimeoutList(timeoutHeight, txidList)
		exec.ledger.AddState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.TimeoutKey(timeoutHeight)), []byte(newStr))
	}
	for recordHeight, txidList := range removeTimeoutListMap {
		newStr := exec.removeTimeoutList(recordHeight, txidList)
		exec.ledger.SetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.TimeoutKey(recordHeight)), []byte(newStr), nil)
	}
	return nil
}

func (exec *BlockExecutor) addTimeoutList(timeoutHeight uint64, txIdList string) string {
	ok, val := exec.ledger.GetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.TimeoutKey(timeoutHeight)))
	if !ok {
		return exec.writeToStr("", txIdList)
	}
	newstr := exec.writeToStr(string(val), txIdList)
	return newstr
}
func (exec *BlockExecutor) removeTimeoutList(recordHeight uint64, txidList string) string {
	ok, val := exec.ledger.GetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.TimeoutKey(recordHeight)))
	if !ok {
		exec.logger.Errorf("there is wrong TimeoutHeight:[%d]", recordHeight)
	}
	currentStr := string(val)
	for _, txId := range strings.Split(txidList, ",") {
		currentStr = exec.removeFromStr(currentStr, txId)
	}
	return currentStr
}

func (exec *BlockExecutor) writeToStr(str string, txId string) string {
	var builder strings.Builder
	if str == "" {
		builder.WriteString(txId)
		return builder.String()
	}
	builder.WriteString(str)
	builder.WriteString(",")
	builder.WriteString(txId)
	return builder.String()
}

func (exec *BlockExecutor) removeFromStr(str string, txId string) string {
	list := strings.Split(str, ",")
	for index, val := range list {
		if val == txId {
			list = append(list[:index], list[index+1:]...)
		}
	}
	return strings.Join(list, ",")
}

func (exec *BlockExecutor) getTxInfoByGlobalID(id string) (*contracts.TransactionInfo, error) {
	ok, val := exec.ledger.GetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.GlobalTxInfoKey(id)))
	if !ok {
		return nil, fmt.Errorf("cannot get tx info by global ID: %s", id)
	}

	var txInfo contracts.TransactionInfo
	if err := json.Unmarshal(val, &txInfo); err != nil {
		return nil, err
	}

	return &txInfo, nil
}

func (exec *BlockExecutor) getMultiTxIBTPsMap(height uint64) (map[string][]string, error) {
	ok, val := exec.ledger.GetState(constant.InterchainContractAddr.Address(), []byte(contracts.MultiTxNotifyKey(height)))
	if !ok {
		return make(map[string][]string), nil
	}

	m := make(map[string][]string)
	if err := json.Unmarshal(val, &m); err != nil {
		return nil, fmt.Errorf("unmarshal multi tx IBTPs map error: %w", err)
	}

	return m, nil
}

func (exec *BlockExecutor) setTxRecord(id string, record pb.TransactionRecord) error {
	value, err := record.Marshal()
	if err != nil {
		return fmt.Errorf("marshal record error: %w", err)
	}

	exec.ledger.SetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.TxInfoKey(id)), value, nil)

	return nil
}

func (exec *BlockExecutor) setGlobalTxStatus(globalID string, status pb.TransactionStatus) error {
	txInfo, err := exec.getTxInfoByGlobalID(globalID)
	if err != nil {
		return err
	}

	txInfo.GlobalState = status
	for id := range txInfo.ChildTxInfo {
		txInfo.ChildTxInfo[id] = status
	}

	data, err := json.Marshal(txInfo)
	if err != nil {
		return fmt.Errorf("marshal txInfo %v: %w", txInfo, err)
	}

	exec.ledger.SetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.GlobalTxInfoKey(globalID)), data, nil)

	return nil
}

func (exec *BlockExecutor) calcTimeoutL2Root(list []string) (types.Hash, error) {
	hashes := make([]merkletree.Content, 0, len(list))
	for _, id := range list {
		hash := sha256.Sum256([]byte(id))
		hashes = append(hashes, types.NewHash(hash[:]))
	}

	tree, err := merkletree.NewTree(hashes)
	if err != nil {
		return types.Hash{}, fmt.Errorf("init merkle tree: %w", err)
	}

	return *types.NewHash(tree.MerkleRoot()), nil
}

func (exec *BlockExecutor) getTimeoutIBTPsMap(height uint64) (map[string][]string, error) {
	timeoutList, err := exec.getTimeoutList(height)
	if err != nil {
		return nil, fmt.Errorf("get timeout list failed: %w", err)
	}

	bxhID := fmt.Sprintf("%d", exec.config.Genesis.ChainID)
	timeoutIBTPsMap := make(map[string][]string)

	for _, value := range timeoutList {
		if exec.isGlobalID(value) {
			txInfo, err := exec.getTxInfoByGlobalID(value)
			if err != nil {
				return nil, err
			}

			for id := range txInfo.ChildTxInfo {
				if err := exec.addTxIdToSrcTimeoutIBTPsMap(timeoutIBTPsMap, id, bxhID, pb.NotifyChain_NOTIFY_SRC); err != nil {
					return nil, err
				}
				childStatus := txInfo.ChildTxInfo[id]
				// if childStatus had arrived final status, bxh need notify ibtp to rollback
				if pb.IsFinalStatus(childStatus) {
					if err := exec.addTxIdToSrcTimeoutIBTPsMap(timeoutIBTPsMap, id, bxhID, pb.NotifyChain_NOTIFY_DEST); err != nil {
						return nil, err
					}
				}
			}
		} else {
			if err := exec.addTxIdToSrcTimeoutIBTPsMap(timeoutIBTPsMap, value, bxhID, pb.NotifyChain_NOTIFY_SRC); err != nil {
				return nil, err
			}
		}
	}

	return timeoutIBTPsMap, nil
}

func (exec *BlockExecutor) addTxIdToSrcTimeoutIBTPsMap(timeoutIBTPsMap map[string][]string, txId string, bitXHubID string, notifyFlag pb.NotifyChain) error {
	listArray := strings.Split(txId, "-")
	bxhID, chainID, _, err := exec.parseChainServiceID(listArray[notifyFlag])
	if err != nil {
		return err
	}
	notifyChain := chainID
	if bxhID != bitXHubID {
		notifyChain = contracts.DEFAULT_UNION_PIER_ID
	}
	if list, has := timeoutIBTPsMap[notifyChain]; has {
		list := append(list, txId)
		timeoutIBTPsMap[notifyChain] = list
	} else {
		timeoutIBTPsMap[notifyChain] = []string{txId}
	}

	return nil
}

func (exec *BlockExecutor) parseChainServiceID(id string) (string, string, string, error) {
	splits := strings.Split(id, ":")

	if len(splits) != 3 {
		return "", "", "", fmt.Errorf("invalid chain service id %s", id)
	}

	return splits[0], splits[1], splits[2], nil
}

func (exec *BlockExecutor) isGlobalID(id string) bool {
	return !strings.Contains(id, "-")
}

func (exec *BlockExecutor) filterValidTx(receipts []*pb.Receipt) (map[string]bool, map[string]bool, error) {
	// filter invalidTx
	recordFailTxHashMap := make(map[string]bool)
	invalidTxHashMap := make(map[string]bool)

	for _, receipt := range receipts {
		// failTx is not validTx
		if receipt.TxStatus == pb.TransactionStatus_BEGIN_FAILURE {
			failTxHash := receipt.TxHash
			recordFailTxHashMap[failTxHash.String()] = true
		}
		// batch ibtp needn't rollback
		if !receipt.IsSuccess() || string(receipt.Ret) == "batch_ibtp" {
			invalidTxHash := receipt.TxHash
			invalidTxHashMap[invalidTxHash.String()] = true
		}
	}

	return invalidTxHashMap, recordFailTxHashMap, nil

}

func (exec *BlockExecutor) isDstChainFromBxh(to string, bxhId string) bool {
	_, chainId, _, _ := exec.parseChainServiceID(to)
	if chainId == bxhId {
		return true
	}
	return false
}
