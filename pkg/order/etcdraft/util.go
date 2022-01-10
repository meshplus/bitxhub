package etcdraft

import (
	"encoding/binary"
	"fmt"
	"sort"
	"time"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/meshplus/bitxhub-core/order"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/sirupsen/logrus"
)

const (
	DefaultBatchTick     = 500 * time.Millisecond
	DefaultSnapshotCount = 1000
	DefaultCheckInterval = 3 * time.Minute
)

func generateRaftPeers(config *order.Config) ([]raft.Peer, error) {
	peers := make([]raft.Peer, 0)
	for id, vpInfo := range config.Nodes {
		vpIngoBytes, err := vpInfo.Marshal()
		if err != nil {
			return nil, fmt.Errorf("mashal vp info error: %w", err)
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
	if len(entriesToApply) > 0 {
		n.logger.Infof("start index:%d, end index:%d", entriesToApply[0].Index, entriesToApply[len(entriesToApply)-1].Index)
	}
	return entriesToApply
}

func (n *Node) ramLastIndex() uint64 {
	i, _ := n.raftStorage.ram.LastIndex()
	n.logger.Infof("New Leader's last index is %d, appliedIndex is %d", i, n.appliedIndex)
	return i
}

func msgToConsensusPbMsg(data []byte, tyr raftproto.RaftMessage_Type, replicaID uint64) *pb.Message {
	rm := &raftproto.RaftMessage{
		Type:   tyr,
		FromId: replicaID,
		Data:   data,
	}
	cmData, err := rm.Marshal()
	if err != nil {
		return nil
	}
	msg := &pb.Message{
		Type: pb.Message_CONSENSUS,
		Data: cmData,
	}
	return msg
}

func (n *Node) getSnapshot() ([]byte, error) {
	cm := pb.ChainMeta{
		Height: n.lastExec,
	}
	return cm.Marshal()
}

func (n *Node) recoverFromSnapshot() {
	snapshot, err := n.raftStorage.snap.Load()
	if err != nil {
		n.logger.Errorf("load snapshot failed: %s", err.Error())
		return
	}
	targetChainMeta := &pb.ChainMeta{}
	err = targetChainMeta.Unmarshal(snapshot.Data)
	if err != nil {
		n.logger.Errorf("unmarshal target chain meta error: %s", err.Error())
		return
	}

	syncBlocks := func() {
		chainMeta := n.getChainMetaFunc()
		n.logger.WithFields(logrus.Fields{
			"target":       targetChainMeta.Height,
			"current":      chainMeta.Height,
			"current_hash": chainMeta.BlockHash.String(),
		}).Info("State Update")

		blockCh := make(chan *pb.Block, 1024)
		go n.syncer.SyncCFTBlocks(chainMeta.Height+1, targetChainMeta.Height, blockCh)
		for block := range blockCh {
			// indicates that the synchronization blocks function has been completed
			if block == nil {
				break
			}
			if block.Height() == n.lastExec+1 {
				localList := make([]bool, len(block.Transactions.Transactions))
				for i := 0; i < len(block.Transactions.Transactions); i++ {
					localList[i] = false
				}
				executeEvent := &pb.CommitEvent{
					Block:     block,
					LocalList: localList,
				}
				n.commitC <- executeEvent
				n.lastExec = block.Height()
			}
		}
	}

	for {
		syncBlocks()
		if n.lastExec == targetChainMeta.Height {
			break
		}
		n.logger.Warnf("The lastExec is %d, but not equal the target block height %d", n.lastExec, targetChainMeta.Height)
	}

	n.appliedIndex = snapshot.Metadata.Index
	n.snapshotIndex = snapshot.Metadata.Index
}
