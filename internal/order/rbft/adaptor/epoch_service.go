package adaptor

import (
	rbft "github.com/axiomesh/axiom-bft"
)

func (s *RBFTAdaptor) GetCurrentEpochInfo() (*rbft.EpochInfo, error) {
	return s.config.GetCurrentEpochInfoFromEpochMgrContractFunc()
}

func (s *RBFTAdaptor) GetEpochInfo(epoch uint64) (*rbft.EpochInfo, error) {
	return s.config.GetEpochInfoFromEpochMgrContractFunc(epoch)
}
