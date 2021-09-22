package etcdraft

import (
	"os"
	"testing"

	"github.com/coreos/etcd/raft/raftpb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
)

// Todo(jz): fix the occasional error
//func TestRecoverFromSnapshot(t *testing.T) {
//	ast := assert.New(t)
//	defer os.RemoveAll("./testdata/storage")
//	node, err := mockRaftNode(t)
//	ast.Nil(err)
//	err = node.Start()
//	ast.Nil(err)
//	snap := raftpb.Snapshot{Data: []byte("test"), Metadata: raftpb.SnapshotMetadata{Index: uint64(2), Term: uint64(1)}}
//	err = node.raftStorage.snap.SaveSnap(snap)
//	ast.Nil(err)
//	node.recoverFromSnapshot()
//	ast.NotEqual(uint64(2), node.appliedIndex, "wrong type data")
//	ast.NotEqual(uint64(2), node.snapshotIndex, "wrong type data")
//
//	go func() {
//		block := <-node.commitC
//		ast.Equal(uint64(2), block.Block.BlockHeader.Number)
//	}()
//	blockHash := &types.Hash{
//		RawHash: [types.HashLength]byte{2},
//	}
//	snapData := &pb.ChainMeta{Height: uint64(2), BlockHash: blockHash}
//	snapDataBytes, _ := snapData.Marshal()
//	snap.Data = snapDataBytes
//	err = node.raftStorage.snap.SaveSnap(snap)
//	ast.Nil(err)
//	node.recoverFromSnapshot()
//	ast.Equal(uint64(2), node.appliedIndex)
//	ast.Equal(uint64(2), node.snapshotIndex)
//}

func TestTakeSnapshotAndGC(t *testing.T) {
	ast := assert.New(t)
	defer os.RemoveAll("./testdata/storage")
	node, err := mockRaftNode(t)
	ast.Nil(err)
	err = node.Start()
	ast.Nil(err)
	entries := []raftpb.Entry{raftpb.Entry{Term: uint64(1), Index: uint64(1)}}
	node.raftStorage.ram.Append(entries)
	err = node.raftStorage.TakeSnapshot(uint64(1), node.confState, []byte("test"))
	ast.Nil(err)
	ast.Equal(1, len(node.raftStorage.snapshotIndex))

	entries = []raftpb.Entry{raftpb.Entry{Term: uint64(1), Index: uint64(2)}}
	node.raftStorage.ram.Append(entries)
	err = node.raftStorage.TakeSnapshot(uint64(2), node.confState, []byte("test2"))

	entries = []raftpb.Entry{raftpb.Entry{Term: uint64(1), Index: uint64(3)}}
	node.raftStorage.ram.Append(entries)
	err = node.raftStorage.TakeSnapshot(uint64(3), node.confState, []byte("test3"))

	entries = []raftpb.Entry{raftpb.Entry{Term: uint64(1), Index: uint64(4)}}
	node.raftStorage.ram.Append(entries)
	err = node.raftStorage.TakeSnapshot(uint64(4), node.confState, []byte("test4"))
	ast.Equal(4, len(node.raftStorage.snapshotIndex))

	entries = []raftpb.Entry{raftpb.Entry{Term: uint64(1), Index: uint64(5)}}
	node.raftStorage.ram.Append(entries)
	err = node.raftStorage.TakeSnapshot(uint64(5), node.confState, []byte("test5"))
	ast.Equal(5, len(node.raftStorage.snapshotIndex))

	entries = []raftpb.Entry{raftpb.Entry{Term: uint64(1), Index: uint64(6)}}
	node.raftStorage.ram.Append(entries)
	err = node.raftStorage.TakeSnapshot(uint64(6), node.confState, []byte("test6"))
	ast.Equal(5, len(node.raftStorage.snapshotIndex), "gc Snap and wal 1")

	err = node.raftStorage.Close()
	ast.Nil(err)
}

func TestCreateOrReadWAL(t *testing.T) {
	ast := assert.New(t)
	defer os.RemoveAll("./testdata/storage")
	node, err := mockRaftNode(t)
	ast.Nil(err)
	err = node.Start()
	ast.Nil(err)
	entries := []raftpb.Entry{raftpb.Entry{Term: uint64(1), Index: uint64(1)}}
	snap := raftpb.Snapshot{Data: []byte("test"), Metadata: raftpb.SnapshotMetadata{Index: uint64(2), Term: uint64(1)}}
	blockHash := &types.Hash{
		RawHash: [types.HashLength]byte{2},
	}
	snapData := &pb.ChainMeta{Height: uint64(2), BlockHash: blockHash}
	snapDataBytes, _ := snapData.Marshal()
	snap.Data = snapDataBytes
	hardstate := raftpb.HardState{Term: uint64(1), Commit: uint64(0)}
	err = node.raftStorage.Store(entries, hardstate, snap)
	ast.Nil(err)
	_, _, _, err = createOrReadWAL(node.logger, node.raftStorage.walDir, &snap)
	ast.NotNil(err, "requested entry index 1 is older than the existing snapshot 2")

	entries = []raftpb.Entry{raftpb.Entry{Term: uint64(1), Index: uint64(4)}}
	snap.Metadata.Index = uint64(3)
	err = node.raftStorage.Store(entries, hardstate, snap)
	ast.Nil(err)
	_ = node.raftStorage.wal.Close()
	_, st, ents, err1 := createOrReadWAL(node.logger, node.raftStorage.walDir, &snap)
	ast.Nil(err1)
	ast.Equal(uint64(1), st.Term)
	ast.Equal(1, len(ents))
}
