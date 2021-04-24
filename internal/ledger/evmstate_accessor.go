package ledger

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	etherTypes "github.com/ethereum/go-ethereum/core/types"
	vm "github.com/meshplus/bitxhub-kit/evm"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

func (l *ChainLedger) CreateEVMAccount(addr common.Address) {
	l.GetOrCreateAccount(types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) SubEVMBalance(addr common.Address, amount *big.Int) {
	l.SubBalance(types.NewAddress(addr.Bytes()), amount)
}

func (l *ChainLedger) AddEVMBalance(addr common.Address, amount *big.Int) {
	l.AddBalance(types.NewAddress(addr.Bytes()), amount)
}

func (l *ChainLedger) GetEVMBalance(addr common.Address) *big.Int {
	return l.GetBalance(types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) GetEVMNonce(addr common.Address) uint64 {
	return l.GetNonce(types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) SetEVMNonce(addr common.Address, nonce uint64) {
	l.SetNonce(types.NewAddress(addr.Bytes()), nonce)
}

func (l *ChainLedger) GetEVMCodeHash(addr common.Address) common.Hash {
	return common.BytesToHash(l.GetCodeHash(types.NewAddress(addr.Bytes())).Bytes())
}

func (l *ChainLedger) GetEVMCode(addr common.Address) []byte {
	return l.GetCode(types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) SetEVMCode(addr common.Address, code []byte) {
	l.SetCode(types.NewAddress(addr.Bytes()), code)
}

func (l *ChainLedger) GetEVMCodeSize(addr common.Address) int {
	return l.GetCodeSize(types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) AddEVMRefund(gas uint64) {
	l.AddRefund(gas)
}

func (l *ChainLedger) SubEVMRefund(gas uint64) {
	l.SubRefund(gas)
}

func (l *ChainLedger) GetEVMRefund() uint64 {
	return l.GetRefund()
}

func (l *ChainLedger) GetEVMCommittedState(addr common.Address, hash common.Hash) common.Hash {
	ret := l.GetCommittedState(types.NewAddress(addr.Bytes()), hash.Bytes())
	return common.BytesToHash(ret)
}

func (l *ChainLedger) GetEVMState(addr common.Address, hash common.Hash) common.Hash {
	ok, ret := l.GetState(types.NewAddress(addr.Bytes()), hash.Bytes())
	if !ok {
		return common.Hash{}
	}
	return common.BytesToHash(ret)
}

func (l *ChainLedger) SetEVMState(addr common.Address, key, value common.Hash) {
	l.SetState(types.NewAddress(addr.Bytes()), key.Bytes(), value.Bytes())
}

func (l *ChainLedger) SuisideEVM(addr common.Address) bool {
	return l.Suiside(types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) HasSuisideEVM(addr common.Address) bool {
	return l.HasSuiside(types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) ExistEVM(addr common.Address) bool {
	return l.Exist(types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) EmptyEVM(addr common.Address) bool {
	return l.Empty(types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) PrepareEVMAccessList(sender common.Address, dest *common.Address, preEVMcompiles []common.Address, txEVMAccesses etherTypes.AccessList) {
	var precompiles []types.Address
	for _, compile := range preEVMcompiles {
		precompiles = append(precompiles, *types.NewAddress(compile.Bytes()))
	}
	var txAccesses AccessList
	for _, list := range txEVMAccesses {
		var storageKeys []types.Hash
		for _, keys := range list.StorageKeys {
			storageKeys = append(storageKeys, *types.NewHash(keys.Bytes()))
		}
		txAccesses = append(txAccesses, AccessTuple{Address: *types.NewAddress(list.Address.Bytes()), StorageKeys: storageKeys})
	}
	l.PrepareAccessList(*types.NewAddress(sender.Bytes()), types.NewAddress(dest.Bytes()), precompiles, txAccesses)
}

func (l *ChainLedger) AddressInEVMAccessList(addr common.Address) bool {
	return l.AddressInAccessList(*types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) SlotInEVMAceessList(addr common.Address, slot common.Hash) (bool, bool) {
	return l.SlotInAccessList(*types.NewAddress(addr.Bytes()), *types.NewHash(slot.Bytes()))
}

func (l *ChainLedger) AddAddressToEVMAccessList(addr common.Address) {
	l.AddAddressToAccessList(*types.NewAddress(addr.Bytes()))
}

func (l *ChainLedger) AddSlotToEVMAccessList(addr common.Address, slot common.Hash) {
	l.AddSlotToAccessList(*types.NewAddress(addr.Bytes()), *types.NewHash(slot.Bytes()))
}

func (l *ChainLedger) AddEVMPreimage(hash common.Hash, data []byte) {
	l.AddPreimage(*types.NewHash(hash.Bytes()), data)
}

func (l *ChainLedger) PrepareEVM(hash common.Hash, index int) {
	l.logs.thash = types.NewHash(hash.Bytes())
	l.logs.txIndex = index
	l.accessList = newAccessList()
}

func (l *ChainLedger) StateDB() vm.StateDB {
	return l
}

func (l *ChainLedger) GetBlockEVMHash(height uint64) common.Hash {
	return common.BytesToHash(l.GetBlockHash(height).Bytes())
}

func (l *ChainLedger) AddEVMLog(log *etherTypes.Log) {
	var topics []types.Hash
	for _, topic := range log.Topics {
		topics = append(topics, *types.NewHash(topic.Bytes()))
	}
	logs := &pb.EvmLog{
		Address:     types.NewAddress(log.Address.Bytes()),
		Topics:      topics,
		Data:        log.Data,
		BlockNumber: log.BlockNumber,
		TxHash:      types.NewHash(log.TxHash.Bytes()),
		TxIndex:     uint64(log.TxIndex),
		BlockHash:   types.NewHash(log.BlockHash.Bytes()),
		Index:       uint64(log.Index),
		Removed:     log.Removed,
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
	}
	return &bin
}

func NewMessage(tx *pb.EthTransaction) etherTypes.Message {
	from := common.BytesToAddress(tx.GetFrom().Bytes())
	to := common.BytesToAddress(tx.GetTo().Bytes())
	nonce := tx.GetNonce()
	amount := new(big.Int).SetUint64(tx.GetAmount())
	gas := tx.GetGas()
	gasPrice := tx.GetGasPrice()
	data := tx.GetPayload()
	accessList := tx.AccessList()
	return etherTypes.NewMessage(from, &to, nonce, amount, gas, gasPrice, data, accessList, true)
}
