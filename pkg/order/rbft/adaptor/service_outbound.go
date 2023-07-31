package adaptor

import (
	"fmt"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/hyperchain/go-hpc-rbft/common/consensus"
	rbfttypes "github.com/hyperchain/go-hpc-rbft/types"
	"github.com/meshplus/bitxhub-kit/types"
	ethtypes "github.com/meshplus/bitxhub-kit/types"
	"github.com/sirupsen/logrus"
)

func (s *RBFTAdaptor) Execute(requests [][]byte, localList []bool, seqNo uint64, timestamp int64) {
	var txs []*types.Transaction
	for _, request := range requests {
		var tx ethtypes.Transaction
		if err := tx.Unmarshal(request); err != nil {
			// TODO: fix
			panic(fmt.Sprintf("failed to unmarshal transaction from rbft: %v", err))
		}
		txs = append(txs, &tx)
	}

	s.ReadyC <- &Ready{
		TXs:       txs,
		LocalList: localList,
		Height:    seqNo,
		Timestamp: timestamp,
	}
}

// TODO: process epoch update and checkpoints
func (s *RBFTAdaptor) StateUpdate(seqNo uint64, digest string, checkpoints []*consensus.SignedCheckpoint, epochChanges ...*consensus.QuorumCheckpoint) {
	s.StateUpdating = true
	s.StateUpdateHeight = seqNo

	var peers []uint64
	for id := range s.Nodes {
		if id != s.localID {
			peers = append(peers, id)
		}
	}

	chain := s.getChainMetaFunc()
	s.logger.WithFields(logrus.Fields{
		"target":       seqNo,
		"target_hash":  digest,
		"current":      chain.Height,
		"current_hash": chain.BlockHash.String(),
	}).Info("State Update")
	get := func(peers []uint64, i int) (block *types.Block, err error) {
		for _, id := range peers {
			block, err = s.getBlock(id, i)
			if err != nil {
				s.logger.Error(err)
				continue
			}

			return block, nil
		}

		return nil, fmt.Errorf("can't get block from all peers")
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
		commitEvent := &types.CommitEvent{
			Block:     block,
			LocalList: localList,
		}
		s.BlockC <- commitEvent
	}
}

func (s *RBFTAdaptor) SendFilterEvent(informType rbfttypes.InformType, message ...interface{}) {
	// TODO: add implement
}
