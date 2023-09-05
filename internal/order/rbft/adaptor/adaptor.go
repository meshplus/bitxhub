package adaptor

import (
	"context"
	"crypto/ecdsa"
	"strconv"

	"github.com/samber/lo"
	"github.com/sirupsen/logrus"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	network "github.com/axiomesh/axiom-p2p"
	"github.com/axiomesh/axiom/internal/order/common"
	"github.com/axiomesh/axiom/internal/peermgr"
)

var _ rbft.ExternalStack[types.Transaction, *types.Transaction] = (*RBFTAdaptor)(nil)
var _ rbft.Storage = (*RBFTAdaptor)(nil)
var _ rbft.Network = (*RBFTAdaptor)(nil)
var _ rbft.Crypto = (*RBFTAdaptor)(nil)
var _ rbft.ServiceOutbound[types.Transaction, *types.Transaction] = (*RBFTAdaptor)(nil)
var _ rbft.EpochService = (*RBFTAdaptor)(nil)

type RBFTAdaptor struct {
	store             *storageWrapper
	priv              *ecdsa.PrivateKey
	peerMgr           peermgr.PeerManager
	msgPipes          map[int32]network.Pipe
	ReadyC            chan *Ready
	BlockC            chan *common.CommitEvent
	logger            logrus.FieldLogger
	getChainMetaFunc  func() *types.ChainMeta
	StateUpdating     bool
	StateUpdateHeight uint64
	cancel            context.CancelFunc
	config            *common.Config

	EpochInfo      *rbft.EpochInfo
	broadcastNodes []string
}

type Ready struct {
	Txs             []*types.Transaction
	LocalList       []bool
	Height          uint64
	Timestamp       int64
	ProposerAccount string
}

func NewRBFTAdaptor(config *common.Config, blockC chan *common.CommitEvent, cancel context.CancelFunc) (*RBFTAdaptor, error) {
	store, err := newStorageWrapper(config.StoragePath, config.StorageType)
	if err != nil {
		return nil, err
	}

	stack := &RBFTAdaptor{
		store:            store,
		priv:             config.PrivKey,
		peerMgr:          config.PeerMgr,
		ReadyC:           make(chan *Ready, 1024),
		BlockC:           blockC,
		logger:           config.Logger,
		getChainMetaFunc: config.GetChainMetaFunc,
		cancel:           cancel,
		config:           config,
	}

	return stack, nil
}

func (s *RBFTAdaptor) UpdateEpoch() error {
	e, err := s.config.GetCurrentEpochInfoFromEpochMgrContractFunc()
	if err != nil {
		return err
	}
	s.EpochInfo = e
	s.broadcastNodes = lo.Map(lo.Flatten([][]*rbft.NodeInfo{s.EpochInfo.ValidatorSet, s.EpochInfo.CandidateSet}), func(item *rbft.NodeInfo, index int) string {
		return item.P2PNodeID
	})
	return nil
}

func (s *RBFTAdaptor) SetMsgPipes(msgPipes map[int32]network.Pipe) {
	s.msgPipes = msgPipes
}

func (s *RBFTAdaptor) getBlock(id string, i int) (*types.Block, error) {
	m := &pb.Message{
		Type: pb.Message_GET_BLOCK,
		Data: []byte(strconv.Itoa(i)),
	}

	res, err := s.peerMgr.Send(id, m)
	if err != nil {
		return nil, err
	}

	block := &types.Block{}
	if err := block.Unmarshal(res.Data); err != nil {
		return nil, err
	}

	return block, nil
}
