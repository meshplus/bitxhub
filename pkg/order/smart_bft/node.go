package smart_bft

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"time"

	"github.com/meshplus/bitxhub-kit/fileutil"

	"github.com/SmartBFT-Go/consensus/pkg/consensus"
	bft "github.com/SmartBFT-Go/consensus/pkg/types"
	"github.com/SmartBFT-Go/consensus/pkg/wal"
	"github.com/SmartBFT-Go/consensus/smartbftprotos"
	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/proto"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/order"
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	proto2 "github.com/meshplus/bitxhub/pkg/order/smart_bft/proto"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

type GetAccountNonceFunc func(address *types.Address) uint64

type Node struct {
	id                  uint64                        // raft id
	leader              uint64                        // leader id
	repoRoot            string                        // project path
	storage             storage.Storage               // db
	logger              logrus.FieldLogger            // logger
	peerMgr             orderPeerMgr.OrderPeerManager // network manager
	commitC             chan *pb.CommitEvent          // the hash commit channel
	node                *consensus.Consensus
	stopChan            chan struct{}
	lastExec            uint64
	getChainMetaFunc    func() *pb.ChainMeta
	getAccountNonceFunc GetAccountNonceFunc
	clock               *time.Ticker
	secondClock         *time.Ticker
}

func init() {
	agency.RegisterOrderConstructor("smartbft", NewNode)
}

// NewNode new smartbft node
func NewNode(opts ...order.Option) (order.Order, error) {
	var options []order.Option
	for i, _ := range opts {
		options = append(options, opts[i])
	}
	config, err := order.GenerateConfig(options...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}

	var writeAheadLog *wal.WriteAheadLogFile
	basicLog, err := zap.NewDevelopment()
	walDir := filepath.Join(config.StoragePath, "wal")
	if fileutil.Exist(walDir) {
		writeAheadLog, err = wal.Open(basicLog.Sugar(), walDir, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot open WAL at %s, err:%v", walDir, err)
		}
		_, err = writeAheadLog.ReadAll()
	} else {
		writeAheadLog, err = wal.Create(basicLog.Sugar(), walDir, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot create WAL at %s, err:%v", walDir, err)
	}

	dbDir := filepath.Join(config.StoragePath, "state")
	db, err := leveldb.New(dbDir)
	if err != nil {
		return nil, fmt.Errorf("failed to new leveldb: %s", err)
	}

	node := &Node{
		id:                  config.ID,
		lastExec:            config.Applied,
		peerMgr:             config.PeerMgr,
		storage:             db,
		getAccountNonceFunc: config.GetAccountNonce,
		commitC:             make(chan *pb.CommitEvent, 1024),
		logger:              config.Logger,
		clock:               time.NewTicker(time.Second),
		secondClock:         time.NewTicker(time.Second),
		getChainMetaFunc:    config.GetChainMetaFunc,
		stopChan:            make(chan struct{}),
	}

	bftConfig := bft.DefaultConfig
	bftConfig.SelfID = config.ID

	//TODO: generate consensus configuration by file
	node.node = &consensus.Consensus{
		Config:             bftConfig,
		ViewChangerTicker:  node.secondClock.C,
		Scheduler:          node.clock.C,
		Logger:             config.Logger,
		Comm:               node,
		Signer:             node,
		MembershipNotifier: node,
		Verifier:           node,
		Application:        node,
		Assembler:          node,
		RequestInspector:   node,
		Synchronizer:       node,
		WAL:                writeAheadLog,
		Metadata: smartbftprotos.ViewMetadata{
			LatestSequence: config.Applied,
			//TODO:reload viewId by other peers
			ViewId: 0,
		},
	}
	return node, nil
}

func (n *Node) Start() error {
	if err := n.node.Start(); err != nil {
		return err
	}
	return nil
}

func (n *Node) Stop() {
	select {
	case <-n.stopChan:
		break
	default:
		close(n.stopChan)
	}
	n.clock.Stop()
	n.node.Stop()
}

func (n *Node) Prepare(tx pb.Transaction) error {
	if err := n.Ready(); err != nil {
		return fmt.Errorf("node get ready failed: %w", err)
	}
	data, err := tx.MarshalWithFlag()
	if err != nil {
		return err
	}
	return n.node.SubmitRequest(data)
}

func (n *Node) Commit() chan *pb.CommitEvent {
	return n.commitC
}

func (n *Node) Step(msg []byte) error {
	bm := proto2.BftMessage{}
	if err := bm.Unmarshal(msg); err != nil {
		return fmt.Errorf("unmarshal bft message error: %w", err)
	}
	switch bm.Type {
	case proto2.BftMessage_CONSENSUS:
		var msg smartbftprotos.Message
		if err := proto.Unmarshal(bm.Data, &msg); err != nil {
			return err
		}
		n.node.HandleMessage(bm.FromId, &msg)
	case proto2.BftMessage_BROADCAST_TX:
		n.node.HandleRequest(bm.FromId, bm.Data)
	}
	return nil
}

func (n *Node) Ready() error {
	hasLeader := n.node.GetLeaderID() != 0
	if !hasLeader {
		return errors.New("in leader election status")
	}
	return nil
}

func (n *Node) ReportState(height uint64, blockHash *types.Hash, txHashList []*types.Hash) {

}

func (n *Node) Quorum() uint64 {
	N := len(n.Nodes())
	F := (int(N) - 1) / 3
	Q := uint64(math.Ceil((float64(N) + float64(F) + 1) / 2.0))
	return Q
}

func (n *Node) GetPendingNonceByAccount(account string) uint64 {
	//TODO: retrieve account nonce from pool
	return n.getAccountNonceFunc(types.NewAddressByStr(account))
}

func (n *Node) GetPendingTxByHash(hash *types.Hash) pb.Transaction {
	return nil
}

func (n *Node) DelNode(delID uint64) error {
	return nil
}

func (n *Node) SubscribeTxEvent(events chan<- pb.Transactions) event.Subscription {
	return event.NewSubscription(func(i <-chan struct{}) error {
		return nil
	})
}
