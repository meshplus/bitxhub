package adaptor

import (
	"fmt"

	"github.com/hyperchain/go-hpc-rbft/v2/common/consensus"
)

// TODO: implement it

// Reconfiguration is used to update router info of consensus, return updated epoch number.
func (s *RBFTAdaptor) Reconfiguration() uint64 {
	return 0
}

// GetNodeInfos returns the full node info with public key.
func (s *RBFTAdaptor) GetNodeInfos() []*consensus.NodeInfo {
	var nodes []*consensus.NodeInfo
	for _, item := range s.Nodes {
		nodes = append(nodes, &consensus.NodeInfo{
			Hostname: item.Pid,
			PubKey:   []byte{},
		})
	}
	return nodes
}

// GetAlgorithmVersion returns current algorithm version.
func (s *RBFTAdaptor) GetAlgorithmVersion() string {
	return "RBFT"
}

// GetEpoch returns the current epoch.
func (s *RBFTAdaptor) GetEpoch() uint64 {
	return 1
}

// IsConfigBlock returns if the block at height is config block.
func (s *RBFTAdaptor) IsConfigBlock(height uint64) bool {
	return false
}

// GetLastCheckpoint return the last QuorumCheckpoint in ledger.
func (s *RBFTAdaptor) GetLastCheckpoint() *consensus.QuorumCheckpoint {
	return &consensus.QuorumCheckpoint{
		Checkpoint: &consensus.Checkpoint{
			Epoch:          0,
			ConsensusState: &consensus.Checkpoint_ConsensusState{},
			ExecuteState: &consensus.Checkpoint_ExecuteState{
				Height: 0,
				Digest: "",
			},
			NextEpochState: &consensus.Checkpoint_NextEpochState{
				ValidatorSet:     []*consensus.NodeInfo{},
				ConsensusVersion: "",
			},
		},
		Signatures: map[string][]byte{},
	}

}

// GetCheckpointOfEpoch gets checkpoint of given epoch.
func (s *RBFTAdaptor) GetCheckpointOfEpoch(epoch uint64) (*consensus.QuorumCheckpoint, error) {
	return nil, fmt.Errorf("unsupported GetCheckpointOfEpoch")
}

// VerifyEpochChangeProof verifies the proof is correctly chained with known validator verifier.
func (s *RBFTAdaptor) VerifyEpochChangeProof(proof *consensus.EpochChangeProof, validators consensus.Validators) error {
	return nil
}
