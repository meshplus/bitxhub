package adaptor

import (
	"context"
	"crypto/ecdsa"
	"strconv"
	"sync"

	"github.com/samber/lo"
	"github.com/sirupsen/logrus"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	"github.com/axiomesh/axiom-ledger/internal/consensus/common"
	"github.com/axiomesh/axiom-ledger/internal/network"
	"github.com/axiomesh/axiom-ledger/internal/storagemgr"
	p2p "github.com/axiomesh/axiom-p2p"
)

var _ rbft.ExternalStack[types.Transaction, *types.Transaction] = (*RBFTAdaptor)(nil)
var _ rbft.Storage = (*RBFTAdaptor)(nil)
var _ rbft.Network = (*RBFTAdaptor)(nil)
var _ rbft.Crypto = (*RBFTAdaptor)(nil)
var _ rbft.ServiceOutbound[types.Transaction, *types.Transaction] = (*RBFTAdaptor)(nil)
var _ rbft.EpochService = (*RBFTAdaptor)(nil)

type RBFTAdaptor struct {
	store             storage.Storage
	priv              *ecdsa.PrivateKey
	network           network.Network
	msgPipes          map[int32]p2p.Pipe
	globalMsgPipe     p2p.Pipe
	ReadyC            chan *Ready
	BlockC            chan *common.CommitEvent
	logger            logrus.FieldLogger
	getChainMetaFunc  func() *types.ChainMeta
	getBlockFunc      func(uint64) (*types.Block, error)
	StateUpdating     bool
	StateUpdateHeight uint64
	cancel            context.CancelFunc
	config            *common.Config
	EpochInfo         *rbft.EpochInfo
	broadcastNodes    []string

	lock sync.Mutex
}

type Ready struct {
	Txs             []*types.Transaction
	LocalList       []bool
	Height          uint64
	Timestamp       int64
	ProposerAccount string
}

func NewRBFTAdaptor(config *common.Config, blockC chan *common.CommitEvent, cancel context.CancelFunc) (*RBFTAdaptor, error) {
	store, err := storagemgr.Open(storagemgr.Consensus)
	if err != nil {
		return nil, err
	}

	stack := &RBFTAdaptor{
		store:            store,
		priv:             config.PrivKey,
		network:          config.Network,
		ReadyC:           make(chan *Ready, 1024),
		BlockC:           blockC,
		logger:           config.Logger,
		getChainMetaFunc: config.GetChainMetaFunc,
		getBlockFunc:     config.GetBlockFunc,
		cancel:           cancel,
		config:           config,
	}

	return stack, nil
}

func (a *RBFTAdaptor) UpdateEpoch() error {
	e, err := a.config.GetCurrentEpochInfoFromEpochMgrContractFunc()
	if err != nil {
		return err
	}
	a.EpochInfo = e
	a.broadcastNodes = lo.Map(lo.Flatten([][]*rbft.NodeInfo{a.EpochInfo.ValidatorSet, a.EpochInfo.CandidateSet}), func(item *rbft.NodeInfo, index int) string {
		return item.P2PNodeID
	})
	return nil
}

func (a *RBFTAdaptor) SetMsgPipes(msgPipes map[int32]p2p.Pipe, globalMsgPipe p2p.Pipe) {
	a.msgPipes = msgPipes
	a.globalMsgPipe = globalMsgPipe
}

func (a *RBFTAdaptor) getBlock(id string, i int) (*types.Block, error) {
	m := &pb.Message{
		Type: pb.Message_GET_BLOCK,
		Data: []byte(strconv.Itoa(i)),
	}

	res, err := a.network.Send(id, m)
	if err != nil {
		return nil, err
	}

	block := &types.Block{}
	if err := block.Unmarshal(res.Data); err != nil {
		return nil, err
	}

	return block, nil
}
