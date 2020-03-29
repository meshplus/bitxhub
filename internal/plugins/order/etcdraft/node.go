package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/internal/plugins/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/internal/plugins/order/etcdraft/txpool"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/meshplus/bitxhub/pkg/storage"
	"github.com/sirupsen/logrus"
)

var defaultSnapshotCount uint64 = 10000

type Node struct {
	id      uint64              // raft id
	node    raft.Node           // raft node
	peerMgr peermgr.PeerManager // network manager
	peers   []raft.Peer         // raft peers

	proposeC    chan *raftproto.Ready    // proposed ready, input channel
	confChangeC <-chan raftpb.ConfChange // proposed cluster config changes
	confState   raftpb.ConfState         // raft requires ConfState to be persisted within snapshot
	commitC     chan *pb.Block           // the hash commit channel
	errorC      chan<- error             // errors from raft session

	raftStorage *RaftStorage    // the raft backend storage system
	tp          *txpool.TxPool  // transaction pool
	storage     storage.Storage // db

	repoRoot string             //project path
	logger   logrus.FieldLogger //logger

	blockAppliedIndex sync.Map // mapping of block height and apply index in raft log
	appliedIndex      uint64   // current apply index in raft log
	snapCount         uint64   // snapshot count
	snapshotIndex     uint64   // current snapshot apply index in raft log
	lastIndex         uint64   // last apply index in raft log

	readyPool  *sync.Pool      // ready pool, avoiding memory growth fast
	readyCache sync.Map        //ready cache
	ctx        context.Context // context
	haltC      chan struct{}   // exit signal
}

// NewNode new raft node
func NewNode(opts ...order.Option) (order.Order, error) {
	config, err := order.GenerateConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}

	repoRoot := config.RepoRoot

	// raft storage directory
	walDir := filepath.Join(config.StoragePath, "wal")
	snapDir := filepath.Join(config.StoragePath, "snap")
	dbDir := filepath.Join(config.StoragePath, "state")
	raftStorage, dbStorage, err := CreateStorage(config.Logger, walDir, snapDir, dbDir, raft.NewMemoryStorage())
	if err != nil {
		return nil, err
	}

	//generate raft peers
	peers, err := GenerateRaftPeers(config)
	if err != nil {
		return nil, fmt.Errorf("generate raft peers: %w", err)
	}

	//generate txpool config
	tpc, err := generateTxPoolConfig(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("generate raft txpool config: %w", err)
	}
	txPool, proposeC := txpool.New(config, dbStorage, tpc.PackSize, tpc.BlockTick)

	readyPool := &sync.Pool{New: func() interface{} {
		return new(raftproto.Ready)
	}}
	return &Node{
		id:          config.ID,
		proposeC:    proposeC,
		confChangeC: make(chan raftpb.ConfChange),
		commitC:     make(chan *pb.Block, 1024),
		errorC:      make(chan<- error),
		tp:          txPool,
		repoRoot:    repoRoot,
		snapCount:   defaultSnapshotCount,
		peerMgr:     config.PeerMgr,
		peers:       peers,
		logger:      config.Logger,
		storage:     dbStorage,
		raftStorage: raftStorage,
		readyPool:   readyPool,
		ctx:         context.Background(),
	}, nil
}

//Start or restart raft node
func (n *Node) Start() error {
	n.blockAppliedIndex.Store(n.tp.GetHeight(), n.loadAppliedIndex())
	rc, err := generateRaftConfig(n.id, n.repoRoot, n.logger, n.raftStorage.ram)
	if err != nil {
		return fmt.Errorf("generate raft config: %w", err)
	}
	if restart {
		n.node = raft.RestartNode(rc)
	} else {
		n.node = raft.StartNode(rc, n.peers)
	}

	go n.run()
	n.logger.Info("Consensus module started")

	return nil
}

//Stop the raft node
func (n *Node) Stop() {
	n.tp.CheckExecute(false)
	n.node.Stop()
	n.logger.Infof("Consensus stopped")
}

//Add the transaction into txpool and broadcast it to other nodes
func (n *Node) Prepare(tx *pb.Transaction) error {
	if err := n.tp.AddPendingTx(tx); err != nil {
		return err
	}
	if err := n.tp.Broadcast(tx); err != nil {
		return err
	}

	return nil
}

func (n *Node) Commit() chan *pb.Block {
	return n.commitC
}

func (n *Node) ReportState(height uint64, hash types.Hash) {
	appliedIndex, ok := n.blockAppliedIndex.Load(height)
	if !ok {
		n.logger.Errorf("can not found appliedIndex:", height)
		return
	}
	//block already persisted, record the apply index in db
	n.writeAppliedIndex(appliedIndex.(uint64))
	n.blockAppliedIndex.Delete(height)

	n.tp.BuildReqLookUp() //store bloom filter

	ready, ok := n.readyCache.Load(height)
	if !ok {
		n.logger.Errorf("can not found ready:", height)
		return
	}
	// remove redundant tx
	n.tp.RemoveTxs(ready.(*raftproto.Ready).TxHashes, n.IsLeader())
	n.readyCache.Delete(height)

	if height%10 == 0 {
		n.logger.WithFields(logrus.Fields{
			"height": height,
			"hash":   hash.ShortString(),
		}).Info("Report checkpoint")
	}
}

func (n *Node) Quorum() uint64 {
	return uint64(len(n.peers)/2 + 1)
}

func (n *Node) Step(ctx context.Context, msg []byte) error {
	rm := &raftproto.RaftMessage{}
	if err := rm.Unmarshal(msg); err != nil {
		return err
	}
	switch rm.Type {
	case raftproto.RaftMessage_CONSENSUS:
		msg := &raftpb.Message{}
		if err := msg.Unmarshal(rm.Data); err != nil {
			return err
		}
		return n.node.Step(ctx, *msg)
	case raftproto.RaftMessage_GET_TX:
		hash := types.Hash{}
		if err := hash.Unmarshal(rm.Data); err != nil {
			return err
		}
		tx, ok := n.tp.GetTx(hash, true)
		if !ok {
			return nil
		}
		v, err := tx.Marshal()
		if err != nil {
			return err
		}
		txAck := &raftproto.RaftMessage{
			Type: raftproto.RaftMessage_GET_TX_ACK,
			Data: v,
		}
		txAckData, err := txAck.Marshal()
		if err != nil {
			return err
		}
		m := &pb.Message{
			Type: pb.Message_CONSENSUS,
			Data: txAckData,
		}
		return n.peerMgr.Send(rm.FromId, m)
	case raftproto.RaftMessage_GET_TX_ACK:
		fallthrough
	case raftproto.RaftMessage_BROADCAST_TX:
		tx := &pb.Transaction{}
		if err := tx.Unmarshal(rm.Data); err != nil {
			return err
		}
		return n.tp.AddPendingTx(tx)
	default:
		return fmt.Errorf("unexpected raft message received")
	}
}

func (n *Node) IsLeader() bool {
	return n.node.Status().SoftState.Lead == n.id
}

func (n *Node) Ready() bool {
	return n.node.Status().SoftState.Lead != 0
}

// main work loop
func (n *Node) run() {
	snap, err := n.raftStorage.ram.Snapshot()
	if err != nil {
		n.logger.Panic(err)
	}
	n.confState = snap.Metadata.ConfState
	n.snapshotIndex = snap.Metadata.Index
	n.appliedIndex = snap.Metadata.Index
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// handle input request
	go func() {
		//
		// TODO: does it matter that this will restart from 0 whenever we restart a cluster?
		//
		confChangeCount := uint64(0)

		for n.proposeC != nil && n.confChangeC != nil {
			select {
			case ready, ok := <-n.proposeC:
				if !ok {
					n.proposeC = nil
				} else {
					data, err := ready.Marshal()
					if err != nil {
						n.logger.Panic(err)
					}
					n.tp.BatchStore(ready.TxHashes)
					if err := n.node.Propose(n.ctx, data); err != nil {
						n.logger.Panic("Failed to propose block [%d] to raft: %s", ready.Height, err)
					}
				}
			case cc, ok := <-n.confChangeC:
				if !ok {
					n.confChangeC = nil
				} else {
					confChangeCount++
					cc.ID = confChangeCount
					if err := n.node.ProposeConfChange(n.ctx, cc); err != nil {
						n.logger.Panic("Failed to propose configuration update to Raft node: %s", err)
					}
				}
			case <-n.ctx.Done():
				return
			}
		}

	}()

	// handle messages from raft state machine
	for {
		select {
		case <-ticker.C:
			n.node.Tick()
			// when the node is first ready it gives us entries to commit and messages
			// to immediately publish
		case rd := <-n.node.Ready():
			// 1: Write HardState, Entries, and Snapshot to persistent storage if they
			// are not empty.
			if err := n.raftStorage.Store(rd.Entries, rd.HardState, rd.Snapshot); err != nil {
				n.logger.Fatalf("failed to persist etcd/raft data: %s", err)
			}

			// 2: Apply Snapshot (if any) and CommittedEntries to the state machine.
			if len(rd.CommittedEntries) != 0 || rd.SoftState != nil {
				n.tp.CheckExecute(n.IsLeader())
				if ok := n.publishEntries(n.entriesToApply(rd.CommittedEntries)); !ok {
					n.Stop()
					return
				}
			}
			// 3: Send all Messages to the nodes named in the To field.
			n.send(rd.Messages)

			n.maybeTriggerSnapshot()

			// 4: Call Node.Advance() to signal readiness for the next batch of
			// updates.
			n.node.Advance()
		case <-n.ctx.Done():
			n.Stop()
		}
	}
}

// send raft consensus message
func (n *Node) send(messages []raftpb.Message) {
	for _, msg := range messages {
		if msg.To == 0 {
			continue
		}
		status := raft.SnapshotFinish

		data, err := (&msg).Marshal()
		if err != nil {
			n.logger.Error(err)
			continue
		}

		rm := &raftproto.RaftMessage{
			Type: raftproto.RaftMessage_CONSENSUS,
			Data: data,
		}
		rmData, err := rm.Marshal()
		if err != nil {
			n.logger.Error(err)
			continue
		}
		p2pMsg := &pb.Message{
			Type: pb.Message_CONSENSUS,
			Data: rmData,
		}

		err = n.peerMgr.Send(msg.To, p2pMsg)
		if err != nil {
			n.logger.WithFields(logrus.Fields{
				"mgs_to": msg.To,
			}).Debugln("message consensus error")
			n.node.ReportUnreachable(msg.To)
			status = raft.SnapshotFailure
		}

		if msg.Type == raftpb.MsgSnap {
			n.node.ReportSnapshot(msg.To, status)
		}
	}
}

func (n *Node) publishEntries(ents []raftpb.Entry) bool {
	for i := range ents {
		switch ents[i].Type {
		case raftpb.EntryNormal:
			if len(ents[i].Data) == 0 {
				// ignore empty messages
				break
			}

			ready := n.readyPool.Get().(*raftproto.Ready)
			if err := ready.Unmarshal(ents[i].Data); err != nil {
				n.logger.Error(err)
				continue
			}
			// This can happen:
			//
			// if (1) we crashed after applying this block to the chain, but
			//        before writing appliedIndex to LDB.
			// or (2) we crashed in a scenario where we applied further than
			//        raft *durably persisted* its committed index (see
			//        https://github.com/coreos/etcd/pull/7899). In this
			//        scenario, when the node comes back up, we will re-apply
			//        a few entries.
			if n.getBlockAppliedIndex() >= ents[i].Index {
				// after commit, update appliedIndex
				n.appliedIndex = ents[i].Index
				continue
			}
			n.mint(ready)
			n.blockAppliedIndex.Store(ready.Height, ents[i].Index)
		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			if err := cc.Unmarshal(ents[i].Data); err != nil {
				continue
			}
			n.confState = *n.node.ApplyConfChange(cc)
			switch cc.Type {
			case raftpb.ConfChangeAddNode:
				//if len(cc.Context) > 0 {
				//	_ := types.Bytes2Address(cc.Context)
				//}
			case raftpb.ConfChangeRemoveNode:
				//if cc.NodeID == n.id {
				//	n.logger.Infoln("I've been removed from the cluster! Shutting down.")
				//	continue
				//}
			}
		}

		// after commit, update appliedIndex
		n.appliedIndex = ents[i].Index

		// special nil commit to signal replay has finished
		if ents[i].Index == n.lastIndex {
			select {
			case n.commitC <- nil:
			case <-n.haltC:
				return false
			}
		}
	}
	return true
}

//mint the block
func (n *Node) mint(ready *raftproto.Ready) {
	loseTxs := make([]types.Hash, 0)
	txs := make([]*pb.Transaction, 0, len(ready.TxHashes))
	for _, hash := range ready.TxHashes {
		_, ok := n.tp.GetTx(hash, false)
		if !ok {
			loseTxs = append(loseTxs, hash)
		}
	}

	//handler missing tx
	if len(loseTxs) != 0 {
		var wg sync.WaitGroup
		wg.Add(len(loseTxs))
		for _, hash := range loseTxs {
			go func(hash types.Hash) {
				defer wg.Done()
				n.tp.FetchTx(hash)
			}(hash)
		}
		wg.Wait()
	}

	for _, hash := range ready.TxHashes {
		tx, _ := n.tp.GetTx(hash, false)
		txs = append(txs, tx)
	}
	//follower node update the block height
	if !n.IsLeader() {
		n.tp.UpdateHeight()
	}
	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Version:   []byte("1.0.0"),
			Number:    ready.Height,
			Timestamp: time.Now().UnixNano(),
		},
		Transactions: txs,
	}

	n.logger.WithFields(logrus.Fields{
		"txpool_size": n.tp.PoolSize(),
	}).Debugln("current tx pool size")
	n.readyCache.Store(ready.Height, ready)
	n.commitC <- block
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

//Determines whether the current apply index triggers a snapshot
func (n *Node) maybeTriggerSnapshot() {
	if n.appliedIndex-n.snapshotIndex <= n.snapCount {
		return
	}
	data := n.raftStorage.Snapshot().Data
	n.logger.Infof("Start snapshot [applied index: %d | last snapshot index: %d]", n.appliedIndex, n.snapshotIndex)
	snap, err := n.raftStorage.ram.CreateSnapshot(n.appliedIndex, &n.confState, data)
	if err != nil {
		panic(err)
	}
	if err := n.raftStorage.saveSnap(snap); err != nil {
		panic(err)
	}

	compactIndex := uint64(1)
	if n.appliedIndex > n.raftStorage.SnapshotCatchUpEntries {
		compactIndex = n.appliedIndex - n.raftStorage.SnapshotCatchUpEntries
	}
	if err := n.raftStorage.ram.Compact(compactIndex); err != nil {
		panic(err)
	}

	n.logger.Infof("compacted log at index %d", compactIndex)
	n.snapshotIndex = n.appliedIndex
}

func GenerateRaftPeers(config *order.Config) ([]raft.Peer, error) {
	nodes := config.Nodes
	peers := make([]raft.Peer, 0, len(nodes))
	for id, node := range nodes {
		peers = append(peers, raft.Peer{ID: id, Context: node.Bytes()})
	}
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

//Load the lastAppliedIndex of
func (n *Node) loadAppliedIndex() uint64 {
	dat, err := n.storage.Get(appliedDbKey)
	var lastAppliedIndex uint64
	if err != nil {
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
	if err := n.storage.Put(appliedDbKey, buf); err != nil {
		n.logger.Errorf("persisted the latest applied index: %s", err)
	}
}
