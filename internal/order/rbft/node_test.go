package rbft

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-bft/common/consensus"
	rbfttypes "github.com/axiomesh/axiom-bft/types"
	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order/common"
	"github.com/axiomesh/axiom/internal/order/precheck"
	"github.com/axiomesh/axiom/internal/order/precheck/mock_precheck"
	"github.com/axiomesh/axiom/internal/order/rbft/adaptor"
	"github.com/axiomesh/axiom/internal/order/rbft/testutil"
	"github.com/axiomesh/axiom/internal/order/txcache"
	"github.com/axiomesh/axiom/pkg/repo"
)

var validTxsCh = make(chan *precheck.ValidTxs, 1024)

func MockMinNode(ctrl *gomock.Controller, t *testing.T) *Node {
	mockRbft := rbft.NewMockMinimalNode[types.Transaction, *types.Transaction](ctrl)
	mockRbft.EXPECT().Status().Return(rbft.NodeStatus{
		ID:     uint64(1),
		View:   uint64(1),
		Status: rbft.Normal,
	}).AnyTimes()
	logger := log.NewWithModule("order")
	logger.Logger.SetLevel(logrus.DebugLevel)
	orderConf := testutil.MockOrderConfig(logger, ctrl, repo.KVStorageTypePebble, t)

	blockC := make(chan *common.CommitEvent, 1024)
	ctx, cancel := context.WithCancel(context.Background())
	rbftAdaptor, err := adaptor.NewRBFTAdaptor(orderConf, blockC, cancel)
	assert.Nil(t, err)
	err = rbftAdaptor.UpdateEpoch()
	assert.Nil(t, err)

	mockPrecheckMgr := mock_precheck.NewMockMinPreCheck(ctrl, validTxsCh)

	rbftConfig, _, err := generateRbftConfig(orderConf)
	assert.Nil(t, err)
	node := &Node{
		config:     orderConf,
		n:          mockRbft,
		stack:      rbftAdaptor,
		blockC:     blockC,
		logger:     logger,
		peerMgr:    orderConf.PeerMgr,
		ctx:        ctx,
		cancel:     cancel,
		txCache:    txcache.NewTxCache(rbftConfig.SetTimeout, uint64(rbftConfig.SetSize), orderConf.Logger),
		txFeed:     event.Feed{},
		txPreCheck: mockPrecheckMgr,
	}
	return node
}

func TestPrepare(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	order := MockMinNode(ctrl, t)

	txCache := make(map[string]*types.Transaction)
	nonceCache := make(map[string]uint64)
	order.n.(*rbft.MockNode[types.Transaction, *types.Transaction]).EXPECT().Propose(gomock.Any(), gomock.Any()).Do(func(requests []*types.Transaction, local bool) error {
		for _, tx := range requests {
			txCache[tx.RbftGetTxHash()] = tx
			if _, ok := nonceCache[tx.GetFrom().String()]; !ok {
				nonceCache[tx.GetFrom().String()] = tx.GetNonce()
			} else if nonceCache[tx.GetFrom().String()] < tx.GetNonce() {
				nonceCache[tx.GetFrom().String()] = tx.GetNonce()
			}
		}
		return nil
	}).Return(nil).AnyTimes()

	order.n.(*rbft.MockNode[types.Transaction, *types.Transaction]).EXPECT().GetPendingNonceByAccount(gomock.Any()).DoAndReturn(func(addr string) uint64 {
		return nonceCache[addr]
	}).AnyTimes()

	order.n.(*rbft.MockNode[types.Transaction, *types.Transaction]).EXPECT().GetPendingTxByHash(gomock.Any()).DoAndReturn(func(hash string) *types.Transaction {
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
		wrongTx := order.GetPendingTxByHash(types.NewHashByStr("0x123"))
		ast.Nil(wrongTx)
	})
}

func TestNode_GetPendingNonceByAccount(t *testing.T) {}

func TestStop(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := MockMinNode(ctrl, t)

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
	node := MockMinNode(ctrl, t)
	err := node.Step([]byte("test"))
	ast.NotNil(err)
	msg := &consensus.ConsensusMessage{}
	msgBytes, _ := msg.Marshal()
	err = node.Step(msgBytes)
	ast.Nil(err)
}

func TestReportState(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := MockMinNode(ctrl, t)

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
	node := MockMinNode(ctrl, t)
	node.stack.EpochInfo.ValidatorSet = []*rbft.NodeInfo{}
	node.stack.EpochInfo.ValidatorSet = append(node.stack.EpochInfo.ValidatorSet, &rbft.NodeInfo{ID: 1})
	node.stack.EpochInfo.ValidatorSet = append(node.stack.EpochInfo.ValidatorSet, &rbft.NodeInfo{ID: 2})
	node.stack.EpochInfo.ValidatorSet = append(node.stack.EpochInfo.ValidatorSet, &rbft.NodeInfo{ID: 3})
	node.stack.EpochInfo.ValidatorSet = append(node.stack.EpochInfo.ValidatorSet, &rbft.NodeInfo{ID: 4})

	// N = 3f + 1, f=1
	quorum := node.Quorum()
	ast.Equal(uint64(3), quorum)

	node.stack.EpochInfo.ValidatorSet = append(node.stack.EpochInfo.ValidatorSet, &rbft.NodeInfo{ID: 5})
	// N = 3f + 2, f=1
	quorum = node.Quorum()
	ast.Equal(uint64(4), quorum)

	node.stack.EpochInfo.ValidatorSet = append(node.stack.EpochInfo.ValidatorSet, &rbft.NodeInfo{ID: 6})
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
