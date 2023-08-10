package rbft

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/axiomesh/axiom/internal/order/txcache"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-bft/common/consensus"
	mocknode "github.com/axiomesh/axiom-bft/mock/mock_node"
	rbfttypes "github.com/axiomesh/axiom-bft/types"
	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/axiomesh/axiom/internal/order/rbft/adaptor"
	"github.com/axiomesh/axiom/internal/order/rbft/testutil"
	"github.com/axiomesh/axiom/pkg/repo"
)

func MockMinNode[T any, Constraint consensus.TXConstraint[T]](ctrl *gomock.Controller, t *testing.T) *Node {
	mockRbft := mocknode.NewMockMinimalNode[T, Constraint](ctrl)
	mockRbft.EXPECT().Status().Return(rbft.NodeStatus{
		ID:     uint64(1),
		View:   uint64(1),
		Epoch:  uint64(0),
		Status: rbft.Normal,
	}).AnyTimes()
	logger := log.NewWithModule("order")
	logger.Logger.SetLevel(logrus.DebugLevel)
	orderConf := &order.Config{
		ID:          uint64(1),
		IsNew:       false,
		Config:      repo.DefaultOrderConfig(),
		StoragePath: t.TempDir(),
		StorageType: "leveldb",
		OrderType:   "rbft",
		Nodes: map[uint64]*types.VpInfo{
			1: {Id: uint64(1)},
			2: {Id: uint64(2)},
			3: {Id: uint64(3)},
		},
		Logger:  logger,
		PeerMgr: testutil.MockMiniPeerManager(ctrl),
		Applied: uint64(1),
		Digest:  "digest",
		GetAccountNonce: func(address *types.Address) uint64 {
			return 0
		},
	}

	blockC := make(chan *types.CommitEvent, 1024)
	ctx, cancel := context.WithCancel(context.Background())
	rbftAdaptor, err := adaptor.NewRBFTAdaptor(orderConf, blockC, cancel)
	assert.Nil(t, err)

	rbftConfig, _, err := generateRbftConfig(orderConf)
	assert.Nil(t, err)
	rbftConfig.External = rbftAdaptor
	node := &Node{
		id:         rbftConfig.ID,
		n:          mockRbft,
		memPool:    rbftConfig.RequestPool,
		logger:     logger,
		stack:      rbftAdaptor,
		blockC:     blockC,
		ctx:        ctx,
		cancel:     cancel,
		txCache:    txcache.NewTxCache(rbftConfig.SetTimeout, uint64(rbftConfig.SetSize), orderConf.Logger),
		peerMgr:    orderConf.PeerMgr,
		checkpoint: orderConf.Config.Mempool.CheckpointPeriod,
	}
	return node
}

func TestPrepare(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	order := MockMinNode[types.Transaction](ctrl, t)

	txCache := make(map[string][]byte)
	nonceCache := make(map[string]uint64)
	order.n.(*mocknode.MockNode[types.Transaction, *types.Transaction]).EXPECT().Propose(gomock.Any()).Do(func(set *consensus.RequestSet) {
		for _, val := range set.Requests {
			tx := &types.Transaction{}
			err := tx.RbftUnmarshal(val)
			ast.Nil(err)
			txCache[tx.RbftGetTxHash()] = val
			if _, ok := nonceCache[tx.GetFrom().String()]; !ok {
				nonceCache[tx.GetFrom().String()] = tx.GetNonce()
			} else if nonceCache[tx.GetFrom().String()] < tx.GetNonce() {
				nonceCache[tx.GetFrom().String()] = tx.GetNonce()
			}
		}
	}).Return(nil).AnyTimes()

	order.n.(*mocknode.MockNode[types.Transaction, *types.Transaction]).EXPECT().GetPendingNonceByAccount(gomock.Any()).DoAndReturn(func(addr string) uint64 {
		return nonceCache[addr]
	}).AnyTimes()

	order.n.(*mocknode.MockNode[types.Transaction, *types.Transaction]).EXPECT().GetPendingTxByHash(gomock.Any()).DoAndReturn(func(hash string) []byte {
		data := txCache[hash]
		return data
	}).AnyTimes()

	sk, err := crypto.GenerateKey()
	ast.Nil(err)

	toAddr := crypto.PubkeyToAddress(sk.PublicKey)
	tx1, singer, err := types.GenerateTransactionAndSigner(uint64(0), types.NewAddressByStr(toAddr.String()), big.NewInt(0), []byte("hello"))
	ast.Nil(err)

	err = order.Start()
	ast.Nil(err)
	err = order.Prepare(tx1)
	ast.Nil(err)

	t.Run("GetPendingNonceByAccount", func(t *testing.T) {
		pendingNonce := order.GetPendingNonceByAccount(tx1.GetFrom().String())
		ast.Equal(uint64(0), pendingNonce)
		tx2, err := types.GenerateTransactionWithSigner(uint64(1), types.NewAddressByStr(toAddr.String()), big.NewInt(0), []byte("hello"), singer)
		ast.Nil(err)
		err = order.Prepare(tx2)
		ast.Nil(err)
		pendingNonce = order.GetPendingNonceByAccount(tx1.GetFrom().String())
		ast.Equal(uint64(1), pendingNonce)
	})

	t.Run("GetPendingTxByHash", func(t *testing.T) {
		tx := order.GetPendingTxByHash(tx1.GetHash())
		ast.NotNil(tx.Inner)
		ast.Equal(tx1.GetHash().String(), tx.GetHash().String())
	})
}

func TestNode_GetPendingNonceByAccount(t *testing.T) {}

func TestStop(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := MockMinNode[types.Transaction](ctrl, t)

	// test start
	err := node.Start()
	ast.Nil(err)
	ast.Nil(node.checkQuorum())

	node.stack.ReadyC <- &adaptor.Ready{
		Height: uint64(2),
	}
	block := <-node.Commit()
	ast.Equal(uint64(2), block.Block.Height())

	// test stop
	node.Stop()
	time.Sleep(1 * time.Second)
	_, ok := <-node.txCache.CloseC
	ast.Equal(false, ok)
}

func TestReadConfig(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	logger := log.NewWithModule("order")
	rbftConf, mempoolConfig, err := generateRbftConfig(testutil.MockOrderConfig(logger, ctrl, "leveldb", t))
	ast.Nil(err)
	rbftConf.Logger.Critical()
	rbftConf.Logger.Criticalf("test critical")
	rbftConf.Logger.Notice()
	rbftConf.Logger.Noticef("test critical")
	ast.Equal(25, rbftConf.SetSize)
	ast.Equal(uint64(500), mempoolConfig.BatchSize)
	ast.Equal(uint64(50000), mempoolConfig.PoolSize)
	ast.Equal(500*time.Millisecond, rbftConf.BatchTimeout)
	ast.Equal(3*time.Minute, rbftConf.CheckPoolTimeout)
	ast.Equal(5*time.Minute, mempoolConfig.ToleranceTime)
}

func TestStep(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := MockMinNode[types.Transaction](ctrl, t)
	err := node.Step([]byte("test"))
	ast.NotNil(err)
	msg := &consensus.ConsensusMessage{}
	msgBytes, _ := msg.Marshal()
	err = node.Step(msgBytes)
	ast.Nil(err)
}

func TestDelNode(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := MockMinNode[types.Transaction](ctrl, t)
	err := node.DelNode(uint64(2))
	ast.Error(err)
}

func TestReportState(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := MockMinNode[types.Transaction](ctrl, t)

	block := testutil.ConstructBlock("blockHash", uint64(20))
	node.stack.StateUpdating = true
	node.stack.StateUpdateHeight = 20
	node.ReportState(uint64(10), block.BlockHash, nil)
	ast.Equal(true, node.stack.StateUpdating)

	state := &rbfttypes.ServiceState{
		MetaState: &rbfttypes.MetaState{
			Height: 20,
			Digest: block.BlockHash.String(),
		},
		Epoch: 0,
	}
	node.n.ReportExecuted(state)
	node.ReportState(uint64(20), block.BlockHash, nil)
	ast.Equal(false, node.stack.StateUpdating)

	node.ReportState(uint64(21), block.BlockHash, nil)
	ast.Equal(false, node.stack.StateUpdating)
}

func TestQuorum(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := MockMinNode[types.Transaction](ctrl, t)
	node.stack.Nodes = make(map[uint64]*types.VpInfo)
	node.stack.Nodes[1] = &types.VpInfo{Id: uint64(1)}
	node.stack.Nodes[2] = &types.VpInfo{Id: uint64(2)}
	node.stack.Nodes[3] = &types.VpInfo{Id: uint64(3)}
	node.stack.Nodes[4] = &types.VpInfo{Id: uint64(4)}
	// N = 3f + 1, f=1
	quorum := node.Quorum()
	ast.Equal(uint64(3), quorum)

	node.stack.Nodes[5] = &types.VpInfo{Id: uint64(5)}
	// N = 3f + 2, f=1
	quorum = node.Quorum()
	ast.Equal(uint64(4), quorum)

	node.stack.Nodes[6] = &types.VpInfo{Id: uint64(6)}
	// N = 3f + 3, f=1
	quorum = node.Quorum()
	ast.Equal(uint64(4), quorum)
}

func TestStatus2String(t *testing.T) {
	ast := assert.New(t)

	assertMapping := map[rbft.StatusType]string{
		rbft.Normal: "Normal",

		rbft.InConfChange:      "system is in conf change",
		rbft.InViewChange:      "system is in view change",
		rbft.InRecovery:        "system is in recovery",
		rbft.StateTransferring: "system is in state update",
		rbft.PoolFull:          "system is too busy",
		rbft.Pending:           "system is in pending state",
		rbft.Stopped:           "system is stopped",
		1000:                   "Unknown status: 1000",
	}

	for status, assertStatusStr := range assertMapping {
		statusStr := status2String(status)
		ast.Equal(assertStatusStr, statusStr)
	}
}
