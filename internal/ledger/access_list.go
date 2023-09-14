package ledger

import "github.com/axiomesh/axiom-kit/types"

type AccessTupleList []AccessTuple

type AccessTuple struct {
	Address     types.Address `json:"address"        gencodec:"required"`
	StorageKeys []types.Hash  `json:"storageKeys"    gencodec:"required"`
}

func (al AccessTupleList) StorageKeys() int {
	sum := 0
	for _, tuple := range al {
		sum += len(tuple.StorageKeys)
	}
	return sum
}

type AccessList struct {
	addresses map[types.Address]int
	slots     []map[types.Hash]struct{}
}

func (al *AccessList) ContainsAddress(addr types.Address) bool {
	_, ok := al.addresses[addr]
	return ok
}

func (al *AccessList) Contains(addr types.Address, slot types.Hash) (bool, bool) {
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

func NewAccessList() *AccessList {
	return &AccessList{
		addresses: make(map[types.Address]int),
	}
}

func (al *AccessList) Copy() *AccessList {
	cp := NewAccessList()
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

func (al *AccessList) AddAddress(address types.Address) bool {
	if _, present := al.addresses[address]; present {
		return false
	}
	al.addresses[address] = -1
	return true
}

func (al *AccessList) AddSlot(address types.Address, slot types.Hash) (bool, bool) {
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

func (al *AccessList) DeleteSlot(address types.Address, slot types.Hash) {
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

func (al *AccessList) DeleteAddress(address types.Address) {
	delete(al.addresses, address)
}
