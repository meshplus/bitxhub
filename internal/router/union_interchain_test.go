package router

//import (
//	"encoding/json"
//	"fmt"
//	"testing"
//
//	"github.com/golang/mock/gomock"
//	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
//	"github.com/meshplus/bitxhub-core/governance"
//	"github.com/meshplus/bitxhub-kit/log"
//	"github.com/meshplus/bitxhub-kit/types"
//	"github.com/meshplus/bitxhub-model/constant"
//	"github.com/meshplus/bitxhub-model/pb"
//	"github.com/meshplus/bitxhub/internal/ledger"
//	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
//	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//)
//
//const (
//	appchainID = "appchain1"
//)
//
//func Test(t *testing.T) {
//	var chainsData [][]byte
//	chainStatus := []string{string(governance.GovernanceAvailable), string(governance.GovernanceUpdating), string(governance.GovernanceFrozen), string(governance.GovernanceFreezing), string(governance.GovernanceActivating), string(governance.GovernanceLogouting)}
//	var chains []*appchainMgr.Appchain
//	for i := 0; i < len(chainStatus); i++ {
//		chain := &appchainMgr.Appchain{
//			ChainName: fmt.Sprintf("应用链%d", i),
//			ChainType: appchainMgr.ChainTypeFabric1_4_3,
//			Status:    governance.GovernanceStatus(chainStatus[i]),
//			ID:        fmt.Sprintf(appchainID, types.NewAddress([]byte{byte(i)}).String()),
//			TrustRoot: []byte(""),
//			Desc:      "",
//			Version:   0,
//		}
//
//		data, err := json.Marshal(chain)
//		assert.Nil(t, err)
//
//		chainsData = append(chainsData, data)
//		chains = append(chains, chain)
//	}
//
//	mockCtl := gomock.NewController(t)
//	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
//	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
//	mockLedger := &ledger.Ledger{
//		ChainLedger: chainLedger,
//		StateLedger: stateLedger,
//	}
//	stateLedger.EXPECT().Copy().Return(stateLedger).AnyTimes()
//	stateLedger.EXPECT().QueryByPrefix(constant.AppchainMgrContractAddr.Address(), "appchain").Return(true, chainsData)
//	mockPeerMgr := mock_peermgr.NewMockPeerManager(mockCtl)
//	router, err := New(log.NewWithModule("router"), nil, mockLedger, mockPeerMgr, 1)
//	require.Nil(t, err)
//
//	var txs []pb.Transaction
//	ibtp1 := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
//	BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp1)
//	BVMTx := mockTx(BVMData)
//	txs = append(txs, BVMTx)
//	m := make(map[string]*pb.VerifiedIndexSlice, 0)
//	m[otherChainID] = &pb.VerifiedIndexSlice{
//		Slice: []*pb.VerifiedIndex{{0, true, false}},
//	}
//	meta := &pb.InterchainMeta{
//		Counter: m,
//		L2Roots: nil,
//	}
//	block := mockBlock(1, txs)
//	ret := router.classify(block, meta)
//	router.generateUnionInterchainTxWrappers(ret, block, meta)
//	stateLedger.EXPECT().QueryByPrefix(constant.AppchainMgrContractAddr.Address(), "appchain").Return(false, nil)
//	router.queryAllAppchains()
//
//	chainsData = append(chainsData, []byte{'a', 'b'})
//	stateLedger.EXPECT().QueryByPrefix(constant.AppchainMgrContractAddr.Address(), "appchain").Return(true, chainsData).AnyTimes()
//	router.queryAllAppchains()
//	router.generateUnionInterchainTxWrappers(ret, block, meta)
//
//}
