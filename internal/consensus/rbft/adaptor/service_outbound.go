package adaptor

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-bft/common/consensus"
	rbfttypes "github.com/axiomesh/axiom-bft/types"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/internal/consensus/common"
)

func (a *RBFTAdaptor) Execute(requests []*types.Transaction, localList []bool, seqNo uint64, timestamp int64, proposerAccount string) {
	a.ReadyC <- &Ready{
		Txs:             requests,
		LocalList:       localList,
		Height:          seqNo,
		Timestamp:       timestamp,
		ProposerAccount: proposerAccount,
	}
}

func (a *RBFTAdaptor) StateUpdate(lowWatermark, seqNo uint64, digest string, checkpoints []*consensus.SignedCheckpoint, epochChanges ...*consensus.EpochChange) {
	a.StateUpdating = true
	a.StateUpdateHeight = seqNo

	var peers []string

	// get the validator set of the remote latest epoch
	if len(epochChanges) != 0 {
		peers = epochChanges[len(epochChanges)-1].GetValidators()
	}

	for _, v := range a.EpochInfo.ValidatorSet {
		if v.AccountAddress != a.config.SelfAccountAddress {
			peers = append(peers, v.P2PNodeID)
		}
	}

	chain := a.getChainMetaFunc()

	startHeight := chain.Height + 1

	if chain.Height >= seqNo {
		localBlock, err := a.getBlockFunc(seqNo)
		if err != nil {
			panic("get local block failed")
		}
		if localBlock.BlockHash.String() != digest {
			a.logger.WithFields(logrus.Fields{
				"remote": digest,
				"local":  localBlock.BlockHash.String(),
				"height": seqNo,
			}).Warningf("Block hash is inconsistent in state update state, we need rollback")
			// rollback to the lowWatermark height
			startHeight = lowWatermark + 1
		} else {
			a.logger.WithFields(logrus.Fields{
				"remote": digest,
				"local":  localBlock.BlockHash.String(),
				"height": seqNo,
			}).Info("state update is ignored, because we have the same block")
			a.StateUpdating = false
			return
		}
	}

	a.logger.WithFields(logrus.Fields{
		"target":      a.StateUpdateHeight,
		"target_hash": digest,
		"start":       startHeight,
	}).Info("State update start")

	syncSize := a.StateUpdateHeight - startHeight + 1

	blockCache := a.getBlockFromOthers(peers, int(syncSize), a.StateUpdateHeight)
	lastBlock := blockCache[len(blockCache)-1]
	if lastBlock.Height() != a.StateUpdateHeight || lastBlock.BlockHash.String() != digest {
		panic(fmt.Errorf("sync block failed: require[height:%d, hash:%s], actual[height:%d, hash:%s]",
			a.StateUpdateHeight, digest, lastBlock.Height(), lastBlock.BlockHash.String()))
	}

	for _, block := range blockCache {
		if block == nil {
			a.logger.Error("Receive a nil block")
			return
		}
		localList := make([]bool, len(block.Transactions))
		for i := 0; i < len(block.Transactions); i++ {
			localList[i] = false
		}

		// todo(lrx): verify sign of each checkpoint?
		var stateUpdatedCheckpoint *consensus.Checkpoint
		if len(checkpoints) != 0 {
			stateUpdatedCheckpoint = checkpoints[0].GetCheckpoint()
		}

		commitEvent := &common.CommitEvent{
			Block:                  block,
			StateUpdatedCheckpoint: stateUpdatedCheckpoint,
		}
		a.BlockC <- commitEvent
	}

	a.logger.WithFields(logrus.Fields{
		"target":      seqNo,
		"target_hash": digest,
	}).Info("State update finished fetch blocks")
}

func (a *RBFTAdaptor) get(peers []string, i int) (block *types.Block, err error) {
	for _, id := range peers {
		block, err = a.getBlock(id, i)
		if err != nil {
			a.logger.Error(err)
			continue
		}
		return block, nil
	}

	return nil, errors.New("can't get block from all peers")
}

func (a *RBFTAdaptor) getBlockFromOthers(peers []string, size int, seqNo uint64) []*types.Block {
	blockCache := make([]*types.Block, size)
	wg := &sync.WaitGroup{}
	wg.Add(size)
	for i := 0; i < size; i++ {
		go func(i int) {
			defer wg.Done()
			if err := retry.Retry(func(attempt uint) (err error) {
				curHeight := int(seqNo) - i
				block, err := a.get(peers, curHeight)
				if err != nil {
					a.logger.Info(err)
					return err
				}
				a.lock.Lock()
				blockCache[size-i-1] = block
				a.lock.Unlock()
				return nil
			}, strategy.Wait(200*time.Millisecond)); err != nil {
				a.logger.Error(err)
			}
		}(i)
	}
	wg.Wait()
	return blockCache
}

func (a *RBFTAdaptor) SendFilterEvent(informType rbfttypes.InformType, message ...any) {
	// TODO: add implement
}
