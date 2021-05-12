package ledger

import "github.com/meshplus/bitxhub-kit/types"

type AccessList []AccessTuple

type AccessTuple struct {
	Address     types.Address `json:"address"        gencodec:"required"`
	StorageKeys []types.Hash  `json:"storageKeys"    gencodec:"required"`
}

func (al AccessList) StorageKeys() int {
	sum := 0
	for _, tuple := range al {
		sum += len(tuple.StorageKeys)
	}
	return sum
}

type accessList struct {
	addresses map[types.Address]int
	slots     []map[types.Hash]struct{}
}

func (al *accessList) ContainsAddress(addr types.Address) bool {
	_, ok := al.addresses[addr]
	return ok
}

func (al *accessList) Contains(addr types.Address, slot types.Hash) (bool, bool) {
	idx, ok := al.addresses[addr]
	if !ok {
		return false, false
	}
	if idx == -1 {
		return true, false
	}
	_, slotPresent := al.slots[idx][slot]
	return true, slotPresent
}

func newAccessList() *accessList {
	return &accessList{
		addresses: make(map[types.Address]int),
	}
}

func (al *accessList) Copy() *accessList {
	cp := newAccessList()
	for k, v := range al.addresses {
		cp.addresses[k] = v
	}
	cp.slots = make([]map[types.Hash]struct{}, len(al.slots))
	for i, slotMap := range al.slots {
		newSlotmap := make(map[types.Hash]struct{}, len(slotMap))
		for k := range slotMap {
			newSlotmap[k] = struct{}{}
		}
		cp.slots[i] = newSlotmap
	}
	return cp
}

func (al *accessList) AddAddress(address types.Address) bool {
	if _, present := al.addresses[address]; present {
		return false
	}
	al.addresses[address] = -1
	return true
}

func (al *accessList) AddSlot(address types.Address, slot types.Hash) (bool, bool) {
	idx, addrPresent := al.addresses[address]
	if !addrPresent || idx == -1 {
		al.addresses[address] = len(al.slots)
		slotmap := map[types.Hash]struct{}{slot: {}}
		al.slots = append(al.slots, slotmap)
		return !addrPresent, true
	}
	slotmap := al.slots[idx]
	if _, ok := slotmap[slot]; !ok {
		slotmap[slot] = struct{}{}
		return false, true
	}
	return false, false
}

func (al *accessList) DeleteSlot(address types.Address, slot types.Hash) {
	idx, ok := al.addresses[address]
	if !ok {
		panic("reverting slot change, address not present in list")
	}
	slotmap := al.slots[idx]
	delete(slotmap, slot)
	if len(slotmap) == 0 {
		al.slots = al.slots[:idx]
		al.addresses[address] = -1
	}
}

func (al *accessList) DeleteAddress(address types.Address) {
	delete(al.addresses, address)
}
