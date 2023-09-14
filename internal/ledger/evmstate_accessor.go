package ledger

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	etherTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/axiomesh/axiom-kit/types"
	vm "github.com/axiomesh/eth-kit/evm"
)

func (l *StateLedgerImpl) CreateEVMAccount(addr common.Address) {
	l.GetOrCreateAccount(types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) SubEVMBalance(addr common.Address, amount *big.Int) {
	l.SubBalance(types.NewAddress(addr.Bytes()), amount)
}

func (l *StateLedgerImpl) AddEVMBalance(addr common.Address, amount *big.Int) {
	l.AddBalance(types.NewAddress(addr.Bytes()), amount)
}

func (l *StateLedgerImpl) GetEVMBalance(addr common.Address) *big.Int {
	return l.GetBalance(types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) GetEVMNonce(addr common.Address) uint64 {
	return l.GetNonce(types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) SetEVMNonce(addr common.Address, nonce uint64) {
	l.SetNonce(types.NewAddress(addr.Bytes()), nonce)
}

func (l *StateLedgerImpl) GetEVMCodeHash(addr common.Address) common.Hash {
	return common.BytesToHash(l.GetCodeHash(types.NewAddress(addr.Bytes())).Bytes())
}

func (l *StateLedgerImpl) GetEVMCode(addr common.Address) []byte {
	return l.GetCode(types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) SetEVMCode(addr common.Address, code []byte) {
	l.SetCode(types.NewAddress(addr.Bytes()), code)
}

func (l *StateLedgerImpl) GetEVMCodeSize(addr common.Address) int {
	return l.GetCodeSize(types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) AddEVMRefund(gas uint64) {
	l.AddRefund(gas)
}

func (l *StateLedgerImpl) SubEVMRefund(gas uint64) {
	l.SubRefund(gas)
}

func (l *StateLedgerImpl) GetEVMRefund() uint64 {
	return l.GetRefund()
}

func (l *StateLedgerImpl) GetEVMCommittedState(addr common.Address, hash common.Hash) common.Hash {
	ret := l.GetCommittedState(types.NewAddress(addr.Bytes()), hash.Bytes())
	return common.BytesToHash(ret)
}

func (l *StateLedgerImpl) GetEVMState(addr common.Address, hash common.Hash) common.Hash {
	ok, ret := l.GetState(types.NewAddress(addr.Bytes()), hash.Bytes())
	if !ok {
		return common.Hash{}
	}
	return common.BytesToHash(ret)
}

func (l *StateLedgerImpl) SetEVMState(addr common.Address, key, value common.Hash) {
	l.SetState(types.NewAddress(addr.Bytes()), key.Bytes(), value.Bytes())
}

func (l *StateLedgerImpl) SetEVMTransientState(addr common.Address, key, value common.Hash) {
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

func (l *StateLedgerImpl) GetEVMTransientState(addr common.Address, key common.Hash) common.Hash {
	return l.transientStorage.Get(*types.NewAddress(addr.Bytes()), key)
}

func (l *StateLedgerImpl) SuicideEVM(addr common.Address) bool {
	return l.Suicide(types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) HasSuicideEVM(addr common.Address) bool {
	return l.HasSuicide(types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) ExistEVM(addr common.Address) bool {
	return l.Exist(types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) EmptyEVM(addr common.Address) bool {
	return l.Empty(types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) PrepareEVMAccessList(sender common.Address, dest *common.Address, preEVMcompiles []common.Address, txEVMAccesses etherTypes.AccessList) {
	var precompiles []types.Address
	for _, compile := range preEVMcompiles {
		precompiles = append(precompiles, *types.NewAddress(compile.Bytes()))
	}
	var txAccesses AccessTupleList
	for _, list := range txEVMAccesses {
		var storageKeys []types.Hash
		for _, keys := range list.StorageKeys {
			storageKeys = append(storageKeys, *types.NewHash(keys.Bytes()))
		}
		txAccesses = append(txAccesses, AccessTuple{Address: *types.NewAddress(list.Address.Bytes()), StorageKeys: storageKeys})
	}
	l.PrepareAccessList(*types.NewAddress(sender.Bytes()), types.NewAddress(dest.Bytes()), precompiles, txAccesses)
}

func (l *StateLedgerImpl) AddressInEVMAccessList(addr common.Address) bool {
	return l.AddressInAccessList(*types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) SlotInEVMAceessList(addr common.Address, slot common.Hash) (bool, bool) {
	return l.SlotInAccessList(*types.NewAddress(addr.Bytes()), *types.NewHash(slot.Bytes()))
}

func (l *StateLedgerImpl) AddAddressToEVMAccessList(addr common.Address) {
	l.AddAddressToAccessList(*types.NewAddress(addr.Bytes()))
}

func (l *StateLedgerImpl) AddSlotToEVMAccessList(addr common.Address, slot common.Hash) {
	l.AddSlotToAccessList(*types.NewAddress(addr.Bytes()), *types.NewHash(slot.Bytes()))
}

func (l *StateLedgerImpl) AddEVMPreimage(hash common.Hash, data []byte) {
	l.AddPreimage(*types.NewHash(hash.Bytes()), data)
}

func (l *StateLedgerImpl) PrepareEVM(rules params.Rules, sender, coinbase common.Address, dst *common.Address, precompiles []common.Address, list etherTypes.AccessList) {
	// l.logs.thash = types.NewHash(hash.Bytes())
	// l.logs.txIndex = index
	l.accessList = NewAccessList()
	if rules.IsBerlin {
		// Clear out any leftover from previous executions
		al := NewAccessList()
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

func (l *StateLedgerImpl) StateDB() vm.StateDB {
	return l
}

func (l *StateLedgerImpl) AddEVMLog(log *etherTypes.Log) {
	var topics []*types.Hash
	for _, topic := range log.Topics {
		topics = append(topics, types.NewHash(topic.Bytes()))
	}
	logs := &types.EvmLog{
		Address:     types.NewAddress(log.Address.Bytes()),
		Topics:      topics,
		Data:        log.Data,
		BlockNumber: log.BlockNumber,
		BlockHash:   types.NewHash(log.BlockHash.Bytes()),
		LogIndex:    uint64(log.Index),
		Removed:     log.Removed,
	}
	l.AddLog(logs)
}

type evmLogs struct {
	logs         map[types.Hash][]*types.EvmLog
	logSize      uint
	thash, bhash *types.Hash
	txIndex      int
}

func NewEvmLogs() *evmLogs {
	return &evmLogs{
		logs: make(map[types.Hash][]*types.EvmLog),
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

type EvmReceipts []*types.Receipt

func CreateBloom(receipts EvmReceipts) *types.Bloom {
	var bin types.Bloom
	for _, receipt := range receipts {
		for _, log := range receipt.EvmLogs {
			bin.Add(log.Address.Bytes())
			for _, b := range log.Topics {
				bin.Add(b.Bytes())
			}
		}
	}
	return &bin
}
