package etcdraft

import (
	"encoding/binary"
	"sort"
	"time"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/meshplus/bitxhub/pkg/order"
)

const (
	DefaultBatchTick = 500 * time.Millisecond
)

func generateRaftPeers(config *order.Config) ([]raft.Peer, error) {
	peers := make([]raft.Peer, 0)
	for id, vpInfo := range config.Nodes {
		vpIngoBytes, err := vpInfo.Marshal()
		if err != nil {
			return nil, err
		}
		peers = append(peers, raft.Peer{ID: id, Context: vpIngoBytes})
	}
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].ID < peers[j].ID
	})
	return peers, nil
}

//Get the raft apply index of the highest block
func (n *Node) getBlockAppliedIndex() uint64 {
	height := uint64(0)
	n.blockAppliedIndex.Range(
		func(key, value interface{}) bool {
			k := key.(uint64)
			if k > height {
				height = k
			}
			return true
		})
	appliedIndex, ok := n.blockAppliedIndex.Load(height)
	if !ok {
		return 0
	}
	return appliedIndex.(uint64)
}

//Load the lastAppliedIndex of block height
func (n *Node) loadAppliedIndex() uint64 {
	dat := n.storage.Get(appliedDbKey)
	var lastAppliedIndex uint64
	if dat == nil {
		lastAppliedIndex = 0
	} else {
		lastAppliedIndex = binary.LittleEndian.Uint64(dat)
	}

	return lastAppliedIndex
}

//Write the lastAppliedIndex
func (n *Node) writeAppliedIndex(index uint64) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, index)
	n.storage.Put(appliedDbKey, buf)
}

func (n *Node) isLeader() bool {
	return n.leader == n.id
}

//Determine whether the current apply index is normal
func (n *Node) entriesToApply(allEntries []raftpb.Entry) (entriesToApply []raftpb.Entry) {
	if len(allEntries) == 0 {
		return
	}
	firstIdx := allEntries[0].Index
	if firstIdx > n.appliedIndex+1 {
		n.logger.Fatalf("first index of committed entry[%d] should <= progress.appliedIndex[%d]+1", firstIdx, n.appliedIndex)
	}
	if n.appliedIndex-firstIdx+1 < uint64(len(allEntries)) {
		entriesToApply = allEntries[n.appliedIndex-firstIdx+1:]
	}
	return entriesToApply
}

func (n *Node) ramLastIndex() uint64 {
	i, _ := n.raftStorage.ram.LastIndex()
	n.logger.Infof("New Leader's last index is %d, appliedIndex is %d", i, n.appliedIndex)
	return i
}
