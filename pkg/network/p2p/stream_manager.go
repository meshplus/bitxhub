package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/libp2p/go-libp2p-core/network"
)

var (
	newStreamTimeout = 5 * time.Second
)

type streamMgr struct {
	ctx        context.Context
	protocolID protocol.ID
	host       host.Host

	streams map[peer.ID]network.Stream
	sync.RWMutex
}

func newStreamMng(ctx context.Context, host host.Host, protocolID protocol.ID) *streamMgr {
	return &streamMgr{
		ctx:        ctx,
		protocolID: protocolID,
		host:       host,
		streams:    make(map[peer.ID]network.Stream),
	}
}

func (mng *streamMgr) get(pid peer.ID) (network.Stream, error) {
	mng.Lock()
	defer mng.Unlock()

	s := mng.streams[pid]
	if s != nil {
		return s, nil
	}

	ctx, cancel := context.WithTimeout(mng.ctx, newStreamTimeout)
	defer cancel()

	s, err := mng.host.NewStream(ctx, pid, mng.protocolID)
	if err != nil {
		return nil, fmt.Errorf("new stream: %w", err)
	}

	mng.streams[pid] = s

	return s, nil
}

func (mng *streamMgr) remove(pid peer.ID) {
	mng.Lock()
	defer mng.Unlock()
	delete(mng.streams, pid)
}
