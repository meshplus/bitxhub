package adaptor

import (
	"errors"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-bft/common/consensus"
	rbfttypes "github.com/axiomesh/axiom-bft/types"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order/common"
)

func (s *RBFTAdaptor) Execute(requests []*types.Transaction, localList []bool, seqNo uint64, timestamp int64, proposerAccount string) {
	s.ReadyC <- &Ready{
		TXs:       requests,
		LocalList: localList,
		Height:    seqNo,
		Timestamp: timestamp,
	}
}

// TODO: process epoch update and checkpoints
func (s *RBFTAdaptor) StateUpdate(seqNo uint64, digest string, checkpoints []*consensus.SignedCheckpoint, epochChanges ...*consensus.QuorumCheckpoint) {
	s.StateUpdating = true
	s.StateUpdateHeight = seqNo

	var peers []string
	for _, v := range s.EpochInfo.ValidatorSet {
		if v.AccountAddress != s.config.SelfAccountAddress {
			peers = append(peers, v.P2PNodeID)
		}
	}

	chain := s.getChainMetaFunc()
	s.logger.WithFields(logrus.Fields{
		"target":       seqNo,
		"target_hash":  digest,
		"current":      chain.Height,
		"current_hash": chain.BlockHash.String(),
	}).Info("State Update")
	get := func(peers []string, i int) (block *types.Block, err error) {
		for _, id := range peers {
			block, err = s.getBlock(id, i)
			if err != nil {
				s.logger.Error(err)
				continue
			}

			return block, nil
		}

		return nil, errors.New("can't get block from all peers")
	}

	blockCache := make([]*types.Block, seqNo-chain.Height)
	var block *types.Block
	for i := seqNo; i > chain.Height; i-- {
		if err := retry.Retry(func(attempt uint) (err error) {
			block, err = get(peers, int(i))
			if err != nil {
				s.logger.Info(err)
				return err
			}

			if digest != block.BlockHash.String() {
				s.logger.WithFields(logrus.Fields{
					"required": digest,
					"received": block.BlockHash.String(),
					"height":   i,
				}).Error("block hash is inconsistent in state update state")
				return err
			}

			digest = block.BlockHeader.ParentHash.String()
			blockCache[i-chain.Height-1] = block

			return nil
		}, strategy.Wait(200*time.Millisecond)); err != nil {
			s.logger.Error(err)
		}
	}

	for _, block := range blockCache {
		if block == nil {
			s.logger.Error("Receive a nil block")
			return
		}
		localList := make([]bool, len(block.Transactions))
		for i := 0; i < len(block.Transactions); i++ {
			localList[i] = false
		}
		commitEvent := &common.CommitEvent{
			Block:     block,
			LocalList: localList,
		}
		s.BlockC <- commitEvent
	}
}

func (s *RBFTAdaptor) SendFilterEvent(informType rbfttypes.InformType, message ...any) {
	// TODO: add implement
}
