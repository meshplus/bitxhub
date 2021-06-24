package etcdraft

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/order/mempool"
	"github.com/meshplus/bitxhub/pkg/order/syncer"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/sirupsen/logrus"
)

type Node struct {
	id       uint64             // raft id
	leader   uint64             // leader id
	repoRoot string             // project path
	logger   logrus.FieldLogger // logger

	node          raft.Node           // raft node
	peerMgr       peermgr.PeerManager // network manager
	peers         []raft.Peer         // raft peers
	syncer        syncer.Syncer       // state syncer
	raftStorage   *RaftStorage        // the raft backend storage system
	storage       storage.Storage     // db
	mempool       mempool.MemPool     // transaction pool
	txCache       *mempool.TxCache    // cache the transactions received from api
	batchTimerMgr *BatchTimer

	proposeC          chan *raftproto.RequestBatch // proposed ready, input channel
	confChangeC       chan raftpb.ConfChange       // proposed cluster config changes
	commitC           chan *pb.CommitEvent         // the hash commit channel
	errorC            chan<- error                 // errors from raft session
	tickTimeout       time.Duration                // tick timeout
	checkInterval     time.Duration                // interval for rebroadcast
	msgC              chan []byte                  // receive messages from remote peer
	stateC            chan *mempool.ChainState     // receive the executed block state
	rebroadcastTicker chan *raftproto.TxSlice      // receive the executed block state

	confState         raftpb.ConfState     // raft requires ConfState to be persisted within snapshot
	blockAppliedIndex sync.Map             // mapping of block height and apply index in raft log
	appliedIndex      uint64               // current apply index in raft log
	snapCount         uint64               // snapshot count
	snapshotIndex     uint64               // current snapshot apply index in raft log
	lastIndex         uint64               // last apply index in raft log
	lastExec          uint64               // the index of the last-applied block
	readyPool         *sync.Pool           // ready pool, avoiding memory growth fast
	justElected       bool                 // track new leader status
	getChainMetaFunc  func() *pb.ChainMeta // current chain meta
	ctx               context.Context      // context
	haltC             chan struct{}        // exit signal
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
	peers, err := generateRaftPeers(config)

	if err != nil {
		return nil, fmt.Errorf("generate raft peers: %w", err)
	}

	raftConfig, err := generateRaftConfig(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("generate raft txpool config: %w", err)
	}
	mempoolConf := &mempool.Config{
		ID:              config.ID,
		ChainHeight:     config.Applied,
		Logger:          config.Logger,
		StoragePath:     config.StoragePath,
		GetAccountNonce: config.GetAccountNonce,

		BatchSize:      raftConfig.RAFT.MempoolConfig.BatchSize,
		PoolSize:       raftConfig.RAFT.MempoolConfig.PoolSize,
		TxSliceSize:    raftConfig.RAFT.MempoolConfig.TxSliceSize,
		TxSliceTimeout: raftConfig.RAFT.MempoolConfig.TxSliceTimeout,
	}
	mempoolInst, err := mempool.NewMempool(mempoolConf)
	if err != nil {
		return nil, fmt.Errorf("create mempool instance: %w", err)
	}

	var batchTimeout time.Duration
	if raftConfig.RAFT.BatchTimeout == 0 {
		batchTimeout = DefaultBatchTick
	} else {
		batchTimeout = raftConfig.RAFT.BatchTimeout
	}
	batchTimerMgr := NewTimer(batchTimeout, config.Logger)
	txCache := mempool.NewTxCache(mempoolConf.TxSliceTimeout, mempoolConf.TxSliceSize, config.Logger)

	readyPool := &sync.Pool{New: func() interface{} {
		return new(raftproto.RequestBatch)
	}}

	snapCount := raftConfig.RAFT.SyncerConfig.SnapshotCount
	if snapCount == 0 {
		snapCount = DefaultSnapshotCount
	}

	var checkInterval time.Duration
	if raftConfig.RAFT.CheckInterval == 0 {
		checkInterval = DefaultCheckInterval
	} else {
		checkInterval = raftConfig.RAFT.CheckInterval
	}

	node := &Node{
		id:               config.ID,
		lastExec:         config.Applied,
		confChangeC:      make(chan raftpb.ConfChange),
		commitC:          make(chan *pb.CommitEvent, 1024),
		errorC:           make(chan<- error),
		msgC:             make(chan []byte),
		stateC:           make(chan *mempool.ChainState),
		proposeC:         make(chan *raftproto.RequestBatch),
		snapCount:        snapCount,
		repoRoot:         repoRoot,
		peerMgr:          config.PeerMgr,
		txCache:          txCache,
		batchTimerMgr:    batchTimerMgr,
		peers:            peers,
		logger:           config.Logger,
		getChainMetaFunc: config.GetChainMetaFunc,
		storage:          dbStorage,
		raftStorage:      raftStorage,
		readyPool:        readyPool,
		ctx:              context.Background(),
		mempool:          mempoolInst,
		checkInterval:    checkInterval,
	}
	node.raftStorage.SnapshotCatchUpEntries = node.snapCount

	otherPeers := node.peerMgr.OtherPeers()
	peerIds := make([]uint64, 0, len(otherPeers))
	for id, _ := range otherPeers {
		peerIds = append(peerIds, id)
	}
	stateSyncer, err := syncer.New(raftConfig.RAFT.SyncerConfig.SyncBlocks, config.PeerMgr, node.Quorum(), peerIds, log.NewWithModule("syncer"))
	if err != nil {
		return nil, fmt.Errorf("new state syncer error:%s", err.Error())
	}
	node.syncer = stateSyncer

	node.logger.Infof("Raft localID = %d", node.id)
	node.logger.Infof("Raft lastExec = %d  ", node.lastExec)
	node.logger.Infof("Raft snapshotCount = %d", node.snapCount)
	return node, nil
}

// Start or restart raft node
func (n *Node) Start() error {
	n.blockAppliedIndex.Store(n.lastExec, n.loadAppliedIndex())
	rc, tickTimeout, err := generateEtcdRaftConfig(n.id, n.repoRoot, n.logger, n.raftStorage.ram)
	if err != nil {
		return fmt.Errorf("generate raft config: %w", err)
	}
	if restart {
		n.node = raft.RestartNode(rc)
	} else {
		n.node = raft.StartNode(rc, n.peers)
	}
	n.tickTimeout = tickTimeout

	go n.run()
	go n.txCache.ListenEvent()
	n.logger.Info("Consensus module started")

	return nil
}

// Stop the raft node
func (n *Node) Stop() {
	n.node.Stop()
	n.logger.Infof("Consensus stopped")
}

// Add the transaction into txpool and broadcast it to other nodes
func (n *Node) Prepare(tx *pb.Transaction) error {
	if err := n.Ready(); err != nil {
		return err
	}
	if n.txCache.IsFull() && n.mempool.IsPoolFull() {
		return errors.New("transaction cache are full, we will drop this transaction")
	}
	n.txCache.RecvTxC <- tx
	return nil
}

func (n *Node) Commit() chan *pb.CommitEvent {
	return n.commitC
}

func (n *Node) ReportState(height uint64, blockHash *types.Hash, txHashList []*types.Hash) {
	state := &mempool.ChainState{
		Height:     height,
		BlockHash:  blockHash,
		TxHashList: txHashList,
	}
	n.stateC <- state
}

func (n *Node) Quorum() uint64 {
	return uint64(len(n.peers)/2 + 1)
}

func (n *Node) Step(msg []byte) error {
	n.msgC <- msg
	return nil
}

func (n *Node) Ready() error {
	hasLeader := n.leader != 0
	if !hasLeader {
		return errors.New("in leader election status")
	}
	return nil
}

func (n *Node) GetPendingNonceByAccount(account string) uint64 {
	return n.mempool.GetPendingNonceByAccount(account)
}

// DelNode sends a delete vp request by given id.
func (n *Node) DelNode(delID uint64) error {
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
	rebroadcastTicker := time.NewTicker(n.checkInterval)
	defer ticker.Stop()
	defer rebroadcastTicker.Stop()

	// handle input request
	go func() {
		//
		// TODO: does it matter that this will restart from 0 whenever we restart a cluster?
		//
		confChangeCount := uint64(0)

		for n.proposeC != nil && n.confChangeC != nil {
			select {
			case batch := <-n.proposeC:
				data, err := batch.Marshal()
				if err != nil {
					n.logger.Error("Marshal batch failed")
				}
				n.logger.Debugf("Proposed block %d to raft core consensus", batch.Height)
				if err := n.node.Propose(n.ctx, data); err != nil {
					n.logger.Errorf("Failed to propose block [%d] to raft: %s", batch.Height, err)
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

		case msg := <-n.msgC:
			if err := n.processRaftCoreMsg(msg); err != nil {
				n.logger.Errorf("Process consensus message failed, err: %s", err.Error())
			}

		case txSet := <-n.txCache.TxSetC:
			// 1. send transactions to other peer
			data, err := txSet.Marshal()
			if err != nil {
				n.logger.Errorf("Marshal failed, err: %s", err.Error())
				return
			}
			pbMsg := msgToConsensusPbMsg(data, raftproto.RaftMessage_BROADCAST_TX, n.id)
			_ = n.peerMgr.Broadcast(pbMsg)

			// 2. process transactions
			n.processTransactions(txSet.TxList, true)

		case state := <-n.stateC:
			n.reportState(state)

		case <-rebroadcastTicker.C:
			// check periodically if there are long-pending txs in mempool
			rebroadcastTxs := n.mempool.GetTimeoutTransactions(n.checkInterval)
			for _, txSlice := range rebroadcastTxs {
				txSet := &raftproto.TxSlice{TxList: txSlice}
				data, err := txSet.Marshal()
				if err != nil {
					n.logger.Errorf("Marshal failed, err: %s", err.Error())
					return
				}
				pbMsg := msgToConsensusPbMsg(data, raftproto.RaftMessage_BROADCAST_TX, n.id)
				_ = n.peerMgr.Broadcast(pbMsg)
			}
		case <-n.batchTimerMgr.BatchTimeoutEvent():
			n.batchTimerMgr.StopBatchTimer()
			// call txPool module to generate a tx batch
			if n.isLeader() {
				n.logger.Debug("Leader batch timer expired, try to create a batch")
				if n.mempool.HasPendingRequest() {
					if batch := n.mempool.GenerateBlock(); batch != nil {
						n.postProposal(batch)
					}
				} else {
					n.logger.Debug("The length of priorityIndex is 0, skip the batch timer")
				}
			} else {
				n.logger.Warningf("Replica %d try to generate batch, but the leader is %d", n.id, n.leader)
			}

		// when the node is first ready it gives us entries to commit and messages
		// to immediately publish
		case rd := <-n.node.Ready():
			// 1: Write HardState, Entries, and Snapshot to persistent storage if they
			// are not empty.
			if err := n.raftStorage.Store(rd.Entries, rd.HardState, rd.Snapshot); err != nil {
				n.logger.Errorf("Failed to persist etcd/raft data: %s", err)
			}

			if !raft.IsEmptySnap(rd.Snapshot) {
				n.recoverFromSnapshot()
			}
			if rd.SoftState != nil {
				newLeader := atomic.LoadUint64(&rd.SoftState.Lead)
				if newLeader != n.leader {
					// new leader should not serve requests directly.
					if newLeader == n.id {
						n.justElected = true
					}
					// notify old leader to stop batching
					if n.leader == n.id {
						n.becomeFollower()
					}
					n.logger.Infof("Raft leader changed: %d -> %d", n.leader, newLeader)
					n.leader = newLeader
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
				n.mempool.SetBatchSeqNo(n.lastExec)
				msgInflight := n.ramLastIndex() > n.appliedIndex+1
				if msgInflight {
					n.logger.Debug("There are in flight blocks, new leader should not generate new batches")
				} else {
					n.justElected = false
				}
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

func (n *Node) processTransactions(txList []*pb.Transaction, isLocal bool) {
	// leader node would check if this transaction triggered generating a batch or not
	if n.isLeader() {
		// start batch timer when this node receives the first transaction
		if !n.batchTimerMgr.IsBatchTimerActive() {
			n.batchTimerMgr.StartBatchTimer()
		}
		// If this transaction triggers generating a batch, stop batch timer
		if batch := n.mempool.ProcessTransactions(txList, true, isLocal); batch != nil {
			n.batchTimerMgr.StopBatchTimer()
			n.postProposal(batch)
		}
	} else {
		n.mempool.ProcessTransactions(txList, false, isLocal)
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
			requestBatch := n.readyPool.Get().(*raftproto.RequestBatch)
			if err := requestBatch.Unmarshal(ents[i].Data); err != nil {
				n.logger.Error(err)
				continue
			}
			// strictly avoid writing the same block
			if requestBatch.Height != n.lastExec+1 {
				n.logger.Warningf("Replica %d expects to execute seq=%d, but get seq=%d, ignore it",
					n.id, n.lastExec+1, requestBatch.Height)
				continue
			}
			n.mint(requestBatch)
			n.blockAppliedIndex.Store(requestBatch.Height, ents[i].Index)
			n.setLastExec(requestBatch.Height)
			// update followers' batch sequence number
			if !n.isLeader() {
				n.mempool.SetBatchSeqNo(requestBatch.Height)
			}

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
func (n *Node) mint(requestBatch *raftproto.RequestBatch) {
	n.logger.WithFields(logrus.Fields{
		"height": requestBatch.Height,
		"count":  len(requestBatch.TxList),
	}).Debugln("block will be mint")
	n.logger.Infof("======== Replica %d call execute, height=%d", n.id, requestBatch.Height)
	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Version:   []byte("1.0.0"),
			Number:    requestBatch.Height,
			Timestamp: time.Now().UnixNano(),
		},
		Transactions: requestBatch.TxList,
	}
	// TODO (YH): refactor localLost
	localList := make([]bool, len(requestBatch.TxList))
	for i := 0; i < len(requestBatch.TxList); i++ {
		localList[i] = false
	}
	executeEvent := &pb.CommitEvent{
		Block:     block,
		LocalList: localList,
	}
	n.commitC <- executeEvent
}

//Determines whether the current apply index triggers a snapshot
func (n *Node) maybeTriggerSnapshot() {
	if n.appliedIndex-n.snapshotIndex < n.snapCount {
		return
	}
	data, err := n.getSnapshot()
	if err != nil {
		n.logger.Error(err)
		return
	}
	err = n.raftStorage.TakeSnapshot(n.appliedIndex, n.confState, data)
	if err != nil {
		n.logger.Error(err)
		return
	}
	n.snapshotIndex = n.appliedIndex
}

func (n *Node) reportState(state *mempool.ChainState) {
	height := state.Height
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
	n.mempool.CommitTransactions(state)
}

func (n *Node) processRaftCoreMsg(msg []byte) error {
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
		return n.node.Step(context.Background(), *msg)

	case raftproto.RaftMessage_BROADCAST_TX:
		txSlice := &raftproto.TxSlice{}
		if err := txSlice.Unmarshal(rm.Data); err != nil {
			return err
		}
		n.processTransactions(txSlice.TxList, false)

	default:
		return fmt.Errorf("unexpected raft message received")
	}
	return nil
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

func (n *Node) postProposal(batch *raftproto.RequestBatch) {
	n.proposeC <- batch
	n.batchTimerMgr.StartBatchTimer()
}

func (n *Node) becomeFollower() {
	n.logger.Debugf("Replica %d became follower", n.id)
	n.batchTimerMgr.StopBatchTimer()
}

func (n *Node) setLastExec(height uint64) {
	n.lastExec = height
}
