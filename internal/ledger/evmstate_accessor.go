package ledger

import (
	"math/big"

	"github.com/axiomesh/axiom-kit/types"
	ledger2 "github.com/axiomesh/eth-kit/ledger"
	"github.com/ethereum/go-ethereum/common"
	etherTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func (l *StateLedger) CreateEVMAccount(addr common.Address) {
	l.GetOrCreateAccount(types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) SubEVMBalance(addr common.Address, amount *big.Int) {
	l.SubBalance(types.NewAddress(addr.Bytes()), amount)
}

func (l *StateLedger) AddEVMBalance(addr common.Address, amount *big.Int) {
	l.AddBalance(types.NewAddress(addr.Bytes()), amount)
}

func (l *StateLedger) GetEVMBalance(addr common.Address) *big.Int {
	return l.GetBalance(types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) GetEVMNonce(addr common.Address) uint64 {
	return l.GetNonce(types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) SetEVMNonce(addr common.Address, nonce uint64) {
	l.SetNonce(types.NewAddress(addr.Bytes()), nonce)
}

func (l *StateLedger) GetEVMCodeHash(addr common.Address) common.Hash {
	return common.BytesToHash(l.GetCodeHash(types.NewAddress(addr.Bytes())).Bytes())
}

func (l *StateLedger) GetEVMCode(addr common.Address) []byte {
	return l.GetCode(types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) SetEVMCode(addr common.Address, code []byte) {
	l.SetCode(types.NewAddress(addr.Bytes()), code)
}

func (l *StateLedger) GetEVMCodeSize(addr common.Address) int {
	return l.GetCodeSize(types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) AddEVMRefund(gas uint64) {
	l.AddRefund(gas)
}

func (l *StateLedger) SubEVMRefund(gas uint64) {
	l.SubRefund(gas)
}

func (l *StateLedger) GetEVMRefund() uint64 {
	return l.GetRefund()
}

func (l *StateLedger) GetEVMCommittedState(addr common.Address, hash common.Hash) common.Hash {
	ret := l.GetCommittedState(types.NewAddress(addr.Bytes()), hash.Bytes())
	return common.BytesToHash(ret)
}

func (l *StateLedger) GetEVMState(addr common.Address, hash common.Hash) common.Hash {
	ok, ret := l.GetState(types.NewAddress(addr.Bytes()), hash.Bytes())
	if !ok {
		return common.Hash{}
	}
	return common.BytesToHash(ret)
}

func (l *StateLedger) SetEVMState(addr common.Address, key, value common.Hash) {
	l.SetState(types.NewAddress(addr.Bytes()), key.Bytes(), value.Bytes())
}

func (l *StateLedger) SetEVMTransientState(addr common.Address, key, value common.Hash) {
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

func (l *StateLedger) GetEVMTransientState(addr common.Address, key common.Hash) common.Hash {
	return l.transientStorage.Get(*types.NewAddress(addr.Bytes()), key)
}

func (l *StateLedger) SuisideEVM(addr common.Address) bool {
	return l.Suiside(types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) HasSuisideEVM(addr common.Address) bool {
	return l.HasSuiside(types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) ExistEVM(addr common.Address) bool {
	return l.Exist(types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) EmptyEVM(addr common.Address) bool {
	return l.Empty(types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) PrepareEVMAccessList(sender common.Address, dest *common.Address, preEVMcompiles []common.Address, txEVMAccesses etherTypes.AccessList) {
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

func (l *StateLedger) AddressInEVMAccessList(addr common.Address) bool {
	return l.AddressInAccessList(*types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) SlotInEVMAceessList(addr common.Address, slot common.Hash) (bool, bool) {
	return l.SlotInAccessList(*types.NewAddress(addr.Bytes()), *types.NewHash(slot.Bytes()))
}

func (l *StateLedger) AddAddressToEVMAccessList(addr common.Address) {
	l.AddAddressToAccessList(*types.NewAddress(addr.Bytes()))
}

func (l *StateLedger) AddSlotToEVMAccessList(addr common.Address, slot common.Hash) {
	l.AddSlotToAccessList(*types.NewAddress(addr.Bytes()), *types.NewHash(slot.Bytes()))
}

func (l *StateLedger) AddEVMPreimage(hash common.Hash, data []byte) {
	l.AddPreimage(*types.NewHash(hash.Bytes()), data)
}

func (l *StateLedger) PrepareEVM(rules params.Rules, sender, coinbase common.Address, dst *common.Address, precompiles []common.Address, list etherTypes.AccessList) {
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

func (l *StateLedger) StateDB() ledger2.StateDB {
	return l
}

func (l *StateLedger) AddEVMLog(log *etherTypes.Log) {
	var topics []*types.Hash
	for _, topic := range log.Topics {
		topics = append(topics, types.NewHash(topic.Bytes()))
	}
	logs := &types.EvmLog{
		Address:          types.NewAddress(log.Address.Bytes()),
		Topics:           topics,
		Data:             log.Data,
		BlockNumber:      log.BlockNumber,
		TransactionHash:  l.thash,
		TransactionIndex: uint64(l.txIndex),
		BlockHash:        types.NewHash(log.BlockHash.Bytes()),
		LogIndex:         uint64(log.Index),
		Removed:          log.Removed,
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
		if receipt.Bloom != nil {
			bin.OrBloom(receipt.Bloom)
		}
	}
	return &bin
}
