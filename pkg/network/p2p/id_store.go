package p2p

import (
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	net "github.com/meshplus/bitxhub/pkg/network"
)

type IDStore struct {
	ids map[net.ID]*peer.AddrInfo
	sync.RWMutex
}

func NewIDStore() *IDStore {
	return &IDStore{
		ids: make(map[net.ID]*peer.AddrInfo),
	}
}

func (store *IDStore) Add(id net.ID, addr *peer.AddrInfo) {
	store.Lock()
	defer store.Unlock()
	store.ids[id] = addr
}

func (store *IDStore) Remove(id net.ID) {
	store.Lock()
	defer store.Unlock()
	delete(store.ids, id)
}

func (store *IDStore) Addr(id net.ID) *peer.AddrInfo {
	store.RLock()
	defer store.RUnlock()
	return store.ids[id]
}

func (store *IDStore) Addrs() map[net.ID]*peer.AddrInfo {
	ret := make(map[net.ID]*peer.AddrInfo)
	for id, addr := range store.ids {
		ret[id] = addr
	}

	return ret
}

type NullIDStore struct {
}

func NewNullIDStore() *NullIDStore {
	return &NullIDStore{}
}

func (null *NullIDStore) Add(net.ID, *peer.AddrInfo) {
}

func (null *NullIDStore) Remove(net.ID) {
}

func (null *NullIDStore) Addr(id net.ID) *peer.AddrInfo {
	return id.(*peer.AddrInfo)
}

// Addrs always returns 0
func (null *NullIDStore) Addrs() map[net.ID]*peer.AddrInfo {
	return nil
}
