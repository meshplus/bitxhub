package adaptor

import (
	rbft "github.com/axiomesh/axiom-bft"
)

func (a *RBFTAdaptor) GetCurrentEpochInfo() (*rbft.EpochInfo, error) {
	return a.config.GetCurrentEpochInfoFromEpochMgrContractFunc()
}

func (a *RBFTAdaptor) GetEpochInfo(epoch uint64) (*rbft.EpochInfo, error) {
	return a.config.GetEpochInfoFromEpochMgrContractFunc(epoch)
}
