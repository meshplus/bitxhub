package etcdraft

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/gogo/protobuf/sortkeys"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/order/mempool"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/sirupsen/logrus"
)

var defaultSnapshotCount uint64 = 10000

type Node struct {
	id      uint64              // raft id
	leader  uint64              // leader id
	node    raft.Node           // raft node
	peerMgr peermgr.PeerManager // network manager
	peers   []raft.Peer         // raft peers

	proposeC    chan *raftproto.Ready    // proposed ready, input channel
	confChangeC <-chan raftpb.ConfChange // proposed cluster config changes
	confState   raftpb.ConfState         // raft requires ConfState to be persisted within snapshot
	commitC     chan *pb.Block           // the hash commit channel
	errorC      chan<- error             // errors from raft session
	tickTimeout time.Duration            // tick timeout

	raftStorage *RaftStorage    // the raft backend storage system
	storage     storage.Storage // db
	mempool     mempool.MemPool // transaction pool

	repoRoot string             // project path
	logger   logrus.FieldLogger // logger

	blockAppliedIndex sync.Map // mapping of block height and apply index in raft log
	appliedIndex      uint64   // current apply index in raft log
	snapCount         uint64   // snapshot count
	snapshotIndex     uint64   // current snapshot apply index in raft log
	lastIndex         uint64   // last apply index in raft log

	readyPool  *sync.Pool // ready pool, avoiding memory growth fast
	readyCache sync.Map   // ready cache

	justElected bool
	isRestart   bool

	ctx   context.Context // context
	haltC chan struct{}   // exit signal

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

	// generate raft peers
	peers, err := GenerateRaftPeers(config)

	if err != nil {
		return nil, fmt.Errorf("generate raft peers: %w", err)
	}

	batchC := make(chan *raftproto.Ready)
	memConfig, err := generateMempoolConfig(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("generate raft txpool config: %w", err)
	}
	mempoolConf := &mempool.Config{
		ID:                 config.ID,
		PeerMgr:            config.PeerMgr,
		ChainHeight:        config.Applied,
		GetTransactionFunc: config.GetTransactionFunc,
		Logger:             config.Logger,

		BatchSize:      memConfig.BatchSize,
		BatchTick:      memConfig.BatchTick,
		PoolSize:       memConfig.PoolSize,
		TxSliceSize:    memConfig.TxSliceSize,
		FetchTimeout:   memConfig.FetchTimeout,
		TxSliceTimeout: memConfig.TxSliceTimeout,
	}
	mempoolInst := mempool.NewMempool(mempoolConf, dbStorage, batchC)

	readyPool := &sync.Pool{New: func() interface{} {
		return new(raftproto.Ready)
	}}
	return &Node{
		id:          config.ID,
		proposeC:    batchC,
		confChangeC: make(chan raftpb.ConfChange),
		commitC:     make(chan *pb.Block, 1024),
		errorC:      make(chan<- error),
		repoRoot:    repoRoot,
		snapCount:   defaultSnapshotCount,
		peerMgr:     config.PeerMgr,
		peers:       peers,
		logger:      config.Logger,
		storage:     dbStorage,
		raftStorage: raftStorage,
		readyPool:   readyPool,
		ctx:         context.Background(),
		mempool:     mempoolInst,
	}, nil
}

// Start or restart raft node
func (n *Node) Start() error {
	n.blockAppliedIndex.Store(n.mempool.GetChainHeight(), n.loadAppliedIndex())
	rc, tickTimeout, err := generateRaftConfig(n.id, n.repoRoot, n.logger, n.raftStorage.ram)
	if err != nil {
		return fmt.Errorf("generate raft config: %w", err)
	}
	if restart {
		n.node = raft.RestartNode(rc)
		n.isRestart = true
	} else {
		n.node = raft.StartNode(rc, n.peers)
	}
	n.tickTimeout = tickTimeout

	go n.run()
	n.mempool.Start()
	n.logger.Info("Consensus module started")

	return nil
}

// Stop the raft node
func (n *Node) Stop() {
	n.mempool.Stop()
	n.node.Stop()
	n.logger.Infof("Consensus stopped")
}

// Add the transaction into txpool and broadcast it to other nodes
func (n *Node) Prepare(tx *pb.Transaction) error {
	if err := n.Ready(); err != nil {
		return err
	}
	return n.mempool.RecvTransaction(tx)
}

func (n *Node) Commit() chan *pb.Block {
	return n.commitC
}

func (n *Node) ReportState(height uint64, hash *types.Hash) {
	if height%10 == 0 {
		n.logger.WithFields(logrus.Fields{
			"height": height,
		}).Info("Report checkpoint")
	}
	appliedIndex, ok := n.blockAppliedIndex.Load(height)
	if !ok {
		n.logger.Debugf("can not found appliedIndex:", height)
		return
	}
	// block already persisted, record the apply index in db
	n.writeAppliedIndex(appliedIndex.(uint64))
	n.blockAppliedIndex.Delete(height - 1)

	// TODO: delete readyCache
	readyBytes, ok := n.readyCache.Load(height)
	if !ok {
		n.logger.Debugf("can not found ready:", height)
		return
	}
	ready := readyBytes.(*raftproto.Ready)

	// clean related mempool info
	n.mempool.CommitTransactions(ready)

	n.readyCache.Delete(height)
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
		fetchTxnRequest := &mempool.FetchTxnRequest{}
		if err := fetchTxnRequest.Unmarshal(rm.Data); err != nil {
			return err
		}
		n.mempool.RecvFetchTxnRequest(fetchTxnRequest)

	case raftproto.RaftMessage_GET_TX_ACK:
		fetchTxnResponse := &mempool.FetchTxnResponse{}
		if err := fetchTxnResponse.Unmarshal(rm.Data); err != nil {
			return err
		}
		n.mempool.RecvFetchTxnResponse(fetchTxnResponse)

	case raftproto.RaftMessage_BROADCAST_TX:
		txSlice := &mempool.TxSlice{}
		if err := txSlice.Unmarshal(rm.Data); err != nil {
			return err
		}
		n.mempool.RecvForwardTxs(txSlice)

	default:
		return fmt.Errorf("unexpected raft message received")
	}
	return nil
}

func (n *Node) IsLeader() bool {
	return n.leader == n.id
}

func (n *Node) Ready() error {
	hasLeader := n.leader != 0
	if !hasLeader {
		return errors.New("in leader election status")
	}
	return nil
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
	ticker := time.NewTicker(n.tickTimeout)
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
					n.logger.Debugf("Proposed block %d to raft core consensus", ready.Height)
					if err := n.node.Propose(n.ctx, data); err != nil {
						n.logger.Errorf("Failed to propose block [%d] to raft: %s", ready.Height, err)
					}
				}
			case cc, ok := <-n.confChangeC:
				if !ok {
					n.confChangeC = nil
				} else {
					confChangeCount++
					cc.ID = confChangeCount
					if err := n.node.ProposeConfChange(n.ctx, cc); err != nil {
						n.logger.Errorf("Failed to propose configuration update to Raft node: %s", err)
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
				n.logger.Errorf("failed to persist etcd/raft data: %s", err)
			}

			if rd.SoftState != nil {
				newLeader := atomic.LoadUint64(&rd.SoftState.Lead)
				if newLeader != n.leader {
					n.logger.Infof("Raft leader changed: %d -> %d", n.leader, newLeader)
					oldLeader := n.leader
					n.leader = newLeader
					if newLeader == n.id {
						// If the cluster is started for the first time, the leader node starts listening requests directly.
						if !n.isRestart && n.getBlockAppliedIndex() == uint64(0) {
							n.mempool.UpdateLeader(n.leader)
						} else {
							// new leader should not serve requests
							n.justElected = true
						}
					}
					// old leader node stop batch block
					if oldLeader == n.id {
						n.mempool.UpdateLeader(n.leader)
					}
				}
			}
			// 2: Apply Snapshot (if any) and CommittedEntries to the state machine.
			if len(rd.CommittedEntries) != 0 {
				if ok := n.publishEntries(n.entriesToApply(rd.CommittedEntries)); !ok {
					n.Stop()
					return
				}
			}

			if n.justElected {
				msgInflight := n.ramLastIndex() > n.appliedIndex+1
				if msgInflight {
					n.logger.Debugf("There are in flight blocks, new leader should not serve requests")
					continue
				}
				n.justElected = false
				n.mempool.UpdateLeader(n.leader)
			}

			// 3: AsyncSend all Messages to the nodes named in the To field.
			n.send(rd.Messages)

			n.maybeTriggerSnapshot()

			// 4: Call Node.Advance() to signal readiness for the next batch of updates.
			n.node.Advance()
		case <-n.ctx.Done():
			n.Stop()
		}
	}
}

func (n *Node) ramLastIndex() uint64 {
	i, _ := n.raftStorage.ram.LastIndex()
	n.logger.Infof("New Leader's last index is %d, appliedIndex is %d", i, n.appliedIndex)
	return i
}

// send raft consensus message
func (n *Node) send(messages []raftpb.Message) {
	for _, msg := range messages {
		go func(msg raftpb.Message) {
			if msg.To == 0 {
				return
			}
			status := raft.SnapshotFinish

			data, err := (&msg).Marshal()
			if err != nil {
				n.logger.Error(err)
				return
			}

			rm := &raftproto.RaftMessage{
				Type: raftproto.RaftMessage_CONSENSUS,
				Data: data,
			}
			rmData, err := rm.Marshal()
			if err != nil {
				n.logger.Error(err)
				return
			}
			p2pMsg := &pb.Message{
				Type: pb.Message_CONSENSUS,
				Data: rmData,
			}

			err = n.peerMgr.AsyncSend(msg.To, p2pMsg)
			if err != nil {
				n.logger.WithFields(logrus.Fields{
					"from":     n.id,
					"to":       msg.To,
					"msg_type": msg.Type,
					"err":      err.Error(),
				}).Debugf("async send msg")
				n.node.ReportUnreachable(msg.To)
				status = raft.SnapshotFailure
			}

			if msg.Type == raftpb.MsgSnap {
				n.node.ReportSnapshot(msg.To, status)
			}
		}(msg)
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

			// This can happen:
			//
			// if (1) we crashed after applying this block to the chain, but
			//        before writing appliedIndex to LDB.
			// or (2) we crashed in a scenario where we applied further than
			//        raft *durably persisted* its committed index (see
			//        https://github.com/coreos/etcd/pull/7899). In this
			//        scenario, when the node comes back up, we will re-apply
			//        a few entries.
			blockAppliedIndex := n.getBlockAppliedIndex()
			if blockAppliedIndex >= ents[i].Index {
				n.appliedIndex = ents[i].Index
				continue
			}

			ready := n.readyPool.Get().(*raftproto.Ready)
			if err := ready.Unmarshal(ents[i].Data); err != nil {
				n.logger.Error(err)
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

// mint the block
func (n *Node) mint(ready *raftproto.Ready) {
	n.logger.WithFields(logrus.Fields{
		"height": ready.Height,
		"count":  len(ready.TxHashes),
	}).Debugln("block will be generated")

	// follower node update the block height
	expectHeight := n.mempool.GetChainHeight()
	isLeader := n.IsLeader()
	if !isLeader && expectHeight != ready.Height-1 {
		n.logger.Warningf("Receive batch %d, but not match, expect height: %d", ready.Height, expectHeight+1)
		return
	}

	missingTxsHash, txList := n.mempool.GetBlockByHashList(ready)
	// handle missing txs
	if len(missingTxsHash) != 0 {
		waitLostTxnC := make(chan bool)
		lostTxnEvent := &mempool.LocalMissingTxnEvent{
			Height:             ready.Height,
			WaitC:              waitLostTxnC,
			MissingTxnHashList: missingTxsHash,
		}

		// NOTE!!! block until finishing fetching the missing txs
		n.mempool.FetchTxn(lostTxnEvent)
		select {
		case isSuccess := <-waitLostTxnC:
			if !isSuccess {
				n.logger.Error("Fetch missing txn failed")
				return
			}
			n.logger.Debug("Fetch missing transactions success")

		case <-time.After(mempool.DefaultFetchTxnTimeout):
			// TODO: add fetch request resend timer
			n.logger.Debugf("Fetch missing transactions timeout, block height: %d", ready.Height)
			return
		}

		if missingTxsHash, txList = n.mempool.GetBlockByHashList(ready); len(missingTxsHash) != 0 {
			n.logger.Error("Still missing transaction")
			return
		}
	}
	if !isLeader {
		n.mempool.IncreaseChainHeight()
	}
	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Version:   []byte("1.0.0"),
			Number:    ready.Height,
			Timestamp: time.Now().UnixNano(),
		},
		Transactions: txList,
	}
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
	// sort by node id
	idSlice := make([]uint64, len(nodes))
	i := 0
	for id := range nodes {
		idSlice[i] = id
		i++
	}
	sortkeys.Uint64s(idSlice)

	for _, id := range idSlice {
		addr := nodes[id]
		peers = append(peers, raft.Peer{ID: id, Context: addr.Bytes()})
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

func (n *Node) GetPendingNonceByAccount(account string) uint64 {
	return n.mempool.GetPendingNonceByAccount(account)
}
