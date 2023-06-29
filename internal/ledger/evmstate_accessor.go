package ledger

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	etherTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	vm1 "github.com/meshplus/eth-kit/evm"
	ledger2 "github.com/meshplus/eth-kit/ledger"
	types2 "github.com/meshplus/eth-kit/types"
)

func (l *SimpleLedger) CreateEVMAccount(addr common.Address) {
	l.GetOrCreateAccount(types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) SubEVMBalance(addr common.Address, amount *big.Int) {
	l.SubBalance(types.NewAddress(addr.Bytes()), amount)
}

func (l *SimpleLedger) AddEVMBalance(addr common.Address, amount *big.Int) {
	l.AddBalance(types.NewAddress(addr.Bytes()), amount)
}

func (l *SimpleLedger) GetEVMBalance(addr common.Address) *big.Int {
	return l.GetBalance(types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) GetEVMNonce(addr common.Address) uint64 {
	return l.GetNonce(types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) SetEVMNonce(addr common.Address, nonce uint64) {
	l.SetNonce(types.NewAddress(addr.Bytes()), nonce)
}

func (l *SimpleLedger) GetEVMCodeHash(addr common.Address) common.Hash {
	return common.BytesToHash(l.GetCodeHash(types.NewAddress(addr.Bytes())).Bytes())
}

func (l *SimpleLedger) GetEVMCode(addr common.Address) []byte {
	return l.GetCode(types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) SetEVMCode(addr common.Address, code []byte) {
	l.SetCode(types.NewAddress(addr.Bytes()), code)
}

func (l *SimpleLedger) GetEVMCodeSize(addr common.Address) int {
	return l.GetCodeSize(types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) AddEVMRefund(gas uint64) {
	l.AddRefund(gas)
}

func (l *SimpleLedger) SubEVMRefund(gas uint64) {
	l.SubRefund(gas)
}

func (l *SimpleLedger) GetEVMRefund() uint64 {
	return l.GetRefund()
}

func (l *SimpleLedger) GetEVMCommittedState(addr common.Address, hash common.Hash) common.Hash {
	ret := l.GetCommittedState(types.NewAddress(addr.Bytes()), hash.Bytes())
	return common.BytesToHash(ret)
}

func (l *SimpleLedger) GetEVMState(addr common.Address, hash common.Hash) common.Hash {
	ok, ret := l.GetState(types.NewAddress(addr.Bytes()), hash.Bytes())
	if !ok {
		return common.Hash{}
	}
	return common.BytesToHash(ret)
}

func (l *SimpleLedger) SetEVMState(addr common.Address, key, value common.Hash) {
	l.SetState(types.NewAddress(addr.Bytes()), key.Bytes(), value.Bytes())
}

func (l *SimpleLedger) SetEVMTransientState(addr common.Address, key, value common.Hash) {
	prev := l.GetEVMTransientState(addr, key)
	if prev == value {
		return
	}
	l.changer.append(transientStorageChange{
		account:  types.NewAddress(addr.Bytes()),
		key:      key.Bytes(),
		prevalue: prev.Bytes(),
	})
	l.setTransientState(*types.NewAddress(addr.Bytes()), key.Bytes(), value.Bytes())
}

func (l *SimpleLedger) GetEVMTransientState(addr common.Address, key common.Hash) common.Hash {
	return l.transientStorage.Get(*types.NewAddress(addr.Bytes()), key)
}

func (l *SimpleLedger) SuisideEVM(addr common.Address) bool {
	return l.Suiside(types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) HasSuisideEVM(addr common.Address) bool {
	return l.HasSuiside(types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) ExistEVM(addr common.Address) bool {
	return l.Exist(types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) EmptyEVM(addr common.Address) bool {
	return l.Empty(types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) PrepareEVMAccessList(sender common.Address, dest *common.Address, preEVMcompiles []common.Address, txEVMAccesses etherTypes.AccessList) {
	var precompiles []types.Address
	for _, compile := range preEVMcompiles {
		precompiles = append(precompiles, *types.NewAddress(compile.Bytes()))
	}
	var txAccesses ledger2.AccessTupleList
	for _, list := range txEVMAccesses {
		var storageKeys []types.Hash
		for _, keys := range list.StorageKeys {
			storageKeys = append(storageKeys, *types.NewHash(keys.Bytes()))
		}
		txAccesses = append(txAccesses, ledger2.AccessTuple{Address: *types.NewAddress(list.Address.Bytes()), StorageKeys: storageKeys})
	}
	l.PrepareAccessList(*types.NewAddress(sender.Bytes()), types.NewAddress(dest.Bytes()), precompiles, txAccesses)
}

func (l *SimpleLedger) AddressInEVMAccessList(addr common.Address) bool {
	return l.AddressInAccessList(*types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) SlotInEVMAceessList(addr common.Address, slot common.Hash) (bool, bool) {
	return l.SlotInAccessList(*types.NewAddress(addr.Bytes()), *types.NewHash(slot.Bytes()))
}

func (l *SimpleLedger) AddAddressToEVMAccessList(addr common.Address) {
	l.AddAddressToAccessList(*types.NewAddress(addr.Bytes()))
}

func (l *SimpleLedger) AddSlotToEVMAccessList(addr common.Address, slot common.Hash) {
	l.AddSlotToAccessList(*types.NewAddress(addr.Bytes()), *types.NewHash(slot.Bytes()))
}

func (l *SimpleLedger) AddEVMPreimage(hash common.Hash, data []byte) {
	l.AddPreimage(*types.NewHash(hash.Bytes()), data)
}

func (l *SimpleLedger) PrepareEVM(rules params.Rules, sender, coinbase common.Address, dst *common.Address, precompiles []common.Address, list etherTypes.AccessList) {
	// l.logs.thash = types.NewHash(hash.Bytes())
	// l.logs.txIndex = index
	l.accessList = ledger2.NewAccessList()
	if rules.IsBerlin {
		// Clear out any leftover from previous executions
		al := ledger2.NewAccessList()
		l.accessList = al

		al.AddAddress(*types.NewAddress(sender.Bytes()))
		if dst != nil {
			al.AddAddress(*types.NewAddress(dst.Bytes()))
			// If it's a create-tx, the destination will be added inside evm.create
		}
		for _, addr := range precompiles {
			al.AddAddress(*types.NewAddress(addr.Bytes()))
		}
		for _, el := range list {
			al.AddAddress(*types.NewAddress(el.Address.Bytes()))
			for _, key := range el.StorageKeys {
				al.AddSlot(*types.NewAddress(el.Address.Bytes()), *types.NewHash(key.Bytes()))
			}
		}
		// if rules.IsShanghai { // EIP-3651: warm coinbase
		// 	al.AddAddress(coinbase)
		// }
	}
	// Reset transient storage at the beginning of transaction execution
	l.transientStorage = newTransientStorage()
}

func (l *SimpleLedger) StateDB() ledger2.StateDB {
	return l
}

func (l *SimpleLedger) AddEVMLog(log *etherTypes.Log) {
	var topics []types.Hash
	for _, topic := range log.Topics {
		topics = append(topics, *types.NewHash(topic.Bytes()))
	}
	logs := &pb.EvmLog{
		Address:          types.NewAddress(log.Address.Bytes()),
		Topics:           topics,
		Data:             log.Data,
		BlockNumber:      log.BlockNumber,
		TransactionHash:  types.NewHash(log.TxHash.Bytes()),
		TransactionIndex: uint64(log.TxIndex),
		BlockHash:        types.NewHash(log.BlockHash.Bytes()),
		LogIndex:         uint64(log.Index),
		Removed:          log.Removed,
	}
	l.AddLog(logs)
}

type evmLogs struct {
	logs         map[types.Hash][]*pb.EvmLog
	logSize      uint
	thash, bhash *types.Hash
	txIndex      int
}

func NewEvmLogs() *evmLogs {
	return &evmLogs{
		logs: make(map[types.Hash][]*pb.EvmLog),
	}
}

func (s *evmLogs) SetBHash(hash *types.Hash) {
	s.bhash = hash
}

func (s *evmLogs) SetTHash(hash *types.Hash) {
	s.thash = hash
}

func (s *evmLogs) SetIndex(i int) {
	s.txIndex = i
}

type EvmReceipts []*pb.Receipt

func CreateBloom(receipts EvmReceipts) *types.Bloom {
	var bin types.Bloom
	for _, receipt := range receipts {
		for _, log := range receipt.EvmLogs {
			bin.Add(log.Address.Bytes())
			for _, b := range log.Topics {
				bin.Add(b.Bytes())
			}
		}
		if receipt.Bloom != nil {
			bin.OrBloom(receipt.Bloom)
		}
	}
	return &bin
}

func NewBxhTxFromEth(tx *types2.EthTransaction) *pb.BxhTransaction {
	return &pb.BxhTransaction{
		Version:         tx.GetVersion(),
		From:            tx.GetFrom(),
		To:              tx.GetTo(),
		Timestamp:       tx.GetTimeStamp(),
		TransactionHash: tx.GetHash(),
		Payload:         tx.GetPayload(),
		IBTP:            tx.GetIBTP(),
		Nonce:           tx.GetNonce(),
		Amount:          tx.GetValue().String(),
		Typ:             pb.TxType_EthSignedBxhTx,
		Signature:       tx.GetSignature(),
		Extra:           tx.GetExtra(),
	}
}

func NewMessageFromBxh(tx *pb.BxhTransaction) *vm1.Message {
	from := common.BytesToAddress(tx.GetFrom().Bytes())
	var to *common.Address
	if tx.GetTo() != nil {
		toAddr := common.BytesToAddress(tx.GetTo().Bytes())
		to = &toAddr
	}

	isFake := false
	if v, _, _ := tx.GetRawSignature(); v == nil {
		isFake = true
	}

	msg := &vm1.Message{
		Nonce:             tx.GetNonce(),
		GasLimit:          tx.GetGas(),
		GasPrice:          new(big.Int).Set(tx.GetGasPrice()),
		GasFeeCap:         new(big.Int).SetInt64(0),
		GasTipCap:         new(big.Int).SetInt64(0),
		From:              from,
		To:                to,
		Value:             tx.GetValue(),
		Data:              tx.GetPayload(),
		AccessList:        *new(etherTypes.AccessList),
		SkipAccountChecks: isFake,
	}
	// If baseFee provided, set gasPrice to effectiveGasPrice.
	// if baseFee != nil {
	// 	msg.GasPrice = cmath.BigMin(msg.GasPrice.Add(msg.GasTipCap, baseFee), msg.GasFeeCap)
	// }
	return msg
}
