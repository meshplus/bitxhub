package adaptor

import (
	"fmt"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/hyperchain/go-hpc-rbft/common/consensus"
	rbfttypes "github.com/hyperchain/go-hpc-rbft/types"
	"github.com/meshplus/bitxhub-model/pb"
	ethtypes "github.com/meshplus/eth-kit/types"
	"github.com/sirupsen/logrus"
)

func (s *RBFTAdaptor) Execute(requests [][]byte, localList []bool, seqNo uint64, timestamp int64) {
	var txs []pb.Transaction
	for _, request := range requests {
		var tx ethtypes.EthTransaction
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

// TODO: process epoch update and verify checkpoints
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
	get := func(peers []uint64, i int) (block *pb.Block, err error) {
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

	blockCache := make([]*pb.Block, seqNo-chain.Height)
	var block *pb.Block
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
		localList := make([]bool, len(block.Transactions.Transactions))
		for i := 0; i < len(block.Transactions.Transactions); i++ {
			localList[i] = false
		}
		commitEvent := &pb.CommitEvent{
			Block:     block,
			LocalList: localList,
		}
		s.BlockC <- commitEvent
	}
}

func (s *RBFTAdaptor) SendFilterEvent(informType rbfttypes.InformType, message ...interface{}) {
	// TODO: add implement
}
