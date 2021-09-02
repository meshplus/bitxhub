package router

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	appchain_mgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	srcChainID   = "appchain1"
	dstChainID   = "appchain2"
	otherChainID = "appchain3"
	srcServiceID = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b991"
	dstServiceID = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b992"
	other        = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b993"
)

func TestInterchainRouter_GetInterchainTxWrappers(t *testing.T) {
	var txs []pb.Transaction
	// set tx of TransactionData_BVM type
	ibtp1 := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
	BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp1)
	BVMTx := mockTx(BVMData)
	txs = append(txs, BVMTx)

	m := make(map[string]*pb.VerifiedIndexSlice, 0)

	m[dstChainID] = &pb.VerifiedIndexSlice{
		Slice: []*pb.VerifiedIndex{{0, true}},
	}
	im := &pb.InterchainMeta{
		Counter: m,
		L2Roots: nil,
	}

	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}
	chainLedger.EXPECT().GetBlock(uint64(1)).Return(mockBlock(1, txs), nil).AnyTimes()
	chainLedger.EXPECT().GetBlock(uint64(2)).Return(nil, fmt.Errorf("get block error")).AnyTimes()
	chainLedger.EXPECT().GetBlock(uint64(3)).Return(mockBlock(1, txs), nil).AnyTimes()
	chainLedger.EXPECT().GetInterchainMeta(uint64(1)).Return(im, nil).AnyTimes()
	chainLedger.EXPECT().GetInterchainMeta(uint64(2)).Return(im, nil).AnyTimes()
	chainLedger.EXPECT().GetInterchainMeta(uint64(3)).Return(nil, fmt.Errorf("get interchain meta error")).AnyTimes()

	mockPeerMgr := mock_peermgr.NewMockPeerManager(mockCtl)

	router, err := New(log.NewWithModule("router"), nil, mockLedger, mockPeerMgr, 1)
	require.Nil(t, err)

	wrappersCh1 := make(chan *pb.InterchainTxWrappers, 1)
	wrappersCh2 := make(chan *pb.InterchainTxWrappers, 1)
	wrappersCh3 := make(chan *pb.InterchainTxWrappers, 1)
	wrappersCh4 := make(chan *pb.InterchainTxWrappers, 1)

	err = router.GetInterchainTxWrappers(dstChainID, 1, 1, wrappersCh1)
	require.Nil(t, err)
	err = router.GetInterchainTxWrappers(dstChainID, 2, 2, wrappersCh2)
	require.NotNil(t, err)
	err = router.GetInterchainTxWrappers(dstChainID, 3, 3, wrappersCh3)
	require.NotNil(t, err)
	err = router.GetInterchainTxWrappers(otherChainID, 1, 1, wrappersCh4)
	require.Nil(t, err)

	select {
	case iw1 := <-wrappersCh1:
		require.Equal(t, len(iw1.InterchainTxWrappers), 1)
		require.Equal(t, len(iw1.InterchainTxWrappers[0].Transactions), 1)
		require.Equal(t, iw1.InterchainTxWrappers[0].Transactions[0].Tx.Hash().String(), BVMTx.GetHash().String())
	case iw4 := <-wrappersCh4:
		require.Equal(t, len(iw4.InterchainTxWrappers), 1)
		require.Equal(t, len(iw4.InterchainTxWrappers[0].Transactions), 0)
	default:
		require.Errorf(t, fmt.Errorf("not found interchainWrappers"), "")
	}
}

func TestInterchainRouter_GetBlockHeader(t *testing.T) {
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}
	chainLedger.EXPECT().GetBlock(uint64(1)).Return(&pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number: 1,
		},
	}, nil).AnyTimes()
	chainLedger.EXPECT().GetBlock(uint64(2)).Return(nil, fmt.Errorf("get block error")).AnyTimes()

	mockPeerMgr := mock_peermgr.NewMockPeerManager(mockCtl)

	router, err := New(log.NewWithModule("router"), nil, mockLedger, mockPeerMgr, 1)
	require.Nil(t, err)

	var txs []pb.Transaction
	// set tx of TransactionData_BVM type
	ibtp1 := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
	BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp1)
	BVMTx := mockTx(BVMData)
	txs = append(txs, BVMTx)

	blockCh := make(chan *pb.BlockHeader, 1)
	blockCh2 := make(chan *pb.BlockHeader, 1)
	err = router.GetBlockHeader(1, 1, blockCh)
	require.Nil(t, err)
	err = router.GetBlockHeader(2, 2, blockCh2)
	require.NotNil(t, err)

	select {
	case bh := <-blockCh:
		require.Equal(t, uint64(1), bh.Number)
	default:
		require.Errorf(t, fmt.Errorf("not found blockHeaders"), "")
	}
}

func TestInterchainRouter_AddPier(t *testing.T) {
	router := testStartRouter(t)

	interchainWrappersC, err := router.AddPier(dstChainID)
	require.Nil(t, err)

	var txs []pb.Transaction
	// set tx of TransactionData_BVM type
	ibtp1 := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
	BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp1)
	BVMTx := mockTx(BVMData)
	txs = append(txs, BVMTx)

	m := make(map[string]*pb.VerifiedIndexSlice, 0)

	m[dstChainID] = &pb.VerifiedIndexSlice{
		Slice: []*pb.VerifiedIndex{{0, true}},
	}
	im := &pb.InterchainMeta{
		Counter: m,
		L2Roots: nil,
	}

	router.PutBlockAndMeta(mockBlock(1, txs), im)

	select {
	case iw := <-interchainWrappersC:
		require.Equal(t, len(iw.InterchainTxWrappers), 1)
		require.Equal(t, len(iw.InterchainTxWrappers[0].Transactions), 1)
		require.Equal(t, iw.InterchainTxWrappers[0].Transactions[0].Tx.Hash().String(), BVMTx.GetHash().String())
	default:
		require.Errorf(t, fmt.Errorf("not found interchainWrappers"), "")
	}

	router.RemovePier(dstChainID)

	require.Nil(t, router.Stop())
}

func TestInterchainRouter_AddNonexistentPier(t *testing.T) {
	router := testStartRouter(t)

	interchainWrappersC, err := router.AddPier(dstChainID)
	require.Nil(t, err)

	var txs []pb.Transaction
	// set tx of TransactionData_BVM type
	ibtp1 := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
	BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp1)
	BVMTx := mockTx(BVMData)
	txs = append(txs, BVMTx)

	m := make(map[string]*pb.VerifiedIndexSlice, 0)

	// pier of other is not added
	m[otherChainID] = &pb.VerifiedIndexSlice{
		Slice: []*pb.VerifiedIndex{{0, true}},
	}
	im := &pb.InterchainMeta{
		Counter: m,
		L2Roots: nil,
	}

	router.PutBlockAndMeta(mockBlock(1, txs), im)

	select {
	case iw := <-interchainWrappersC:
		require.Equal(t, len(iw.InterchainTxWrappers), 1)
		require.Equal(t, len(iw.InterchainTxWrappers[0].Transactions), 0)
	default:
		require.Errorf(t, fmt.Errorf("not found interchainWrappers"), "")
	}

	router.RemovePier(dstChainID)

	require.Nil(t, router.Stop())
}

//func TestInterchainRouter_AddUnionPier(t *testing.T) {
//	router := testStartRouter(t)
//
//	interchainWrappersC, err := router.AddPier(dstChainID)
//	require.Nil(t, err)
//
//	var txs []pb.Transaction
//	// set tx of TransactionData_BVM type
//	ibtp1 := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
//	BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp1)
//	BVMTx := mockTx(BVMData)
//	txs = append(txs, BVMTx)
//
//	m := make(map[string]*pb.VerifiedIndexSlice, 0)
//
//	m[otherChainID] = &pb.VerifiedIndexSlice{
//		Slice: []*pb.VerifiedIndex{{0, true}},
//	}
//	im := &pb.InterchainMeta{
//		Counter: m,
//		L2Roots: nil,
//	}
//
//	router.PutBlockAndMeta(mockBlock(1, txs), im)
//
//	select {
//	case iw := <-interchainWrappersC:
//		require.Equal(t, len(iw.InterchainTxWrappers), 1)
//		require.Equal(t, len(iw.InterchainTxWrappers[0].Transactions), 1)
//		require.Equal(t, iw.InterchainTxWrappers[0].Transactions[0].Tx.Hash().String(), BVMTx.GetHash().String())
//	default:
//		require.Errorf(t, fmt.Errorf("not found interchainWrappers"), "")
//	}
//
//	router.RemovePier(dstChainID)
//
//	require.Nil(t, router.Stop())
//}

func testStartRouter(t *testing.T) *InterchainRouter {
	appchains := make([]*appchain_mgr.Appchain, 0)

	var ret [][]byte

	app := &appchain_mgr.Appchain{
		ID:   srcChainID,
		Desc: "app",
	}
	bxh := &appchain_mgr.Appchain{
		ID:   dstChainID,
		Desc: "bxh",
	}

	appchains = append(appchains, app, bxh)
	for _, appchain := range appchains {
		data, err := json.Marshal(appchain)
		require.Nil(t, err)
		ret = append(ret, data)
	}

	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}
	stateLedger.EXPECT().Copy().Return(stateLedger).AnyTimes()
	//stateLedger.EXPECT().QueryByPrefix(constant.AppchainMgrContractAddr.Address(), appchain_mgr.PREFIX).Return(true, ret)

	mockPeerMgr := mock_peermgr.NewMockPeerManager(mockCtl)

	router, err := New(log.NewWithModule("router"), nil, mockLedger, mockPeerMgr, 1)
	require.Nil(t, err)

	require.Nil(t, router.Start())

	return router
}

func mockBlock(blockNumber uint64, txs []pb.Transaction) *pb.Block {
	header := &pb.BlockHeader{
		Number:    blockNumber,
		Timestamp: time.Now().Unix(),
	}
	return &pb.Block{
		BlockHeader:  header,
		Transactions: &pb.Transactions{Transactions: txs},
	}
}

func mockTx(data *pb.TransactionData) pb.Transaction {
	payload, err := data.Marshal()
	if err != nil {
		panic(err)
	}
	tx := &pb.BxhTransaction{
		Payload: payload,
		Nonce:   uint64(rand.Int63()),
	}
	tx.TransactionHash = tx.Hash()

	return tx
}

func mockTxData(t *testing.T, dataType pb.TransactionData_Type, vmType pb.TransactionData_VMType, ibtp proto.Marshaler) *pb.TransactionData {
	ib, err := ibtp.Marshal()
	assert.Nil(t, err)

	tmpIP := &pb.InvokePayload{
		Method: "set",
		Args:   []*pb.Arg{{Value: ib}},
	}
	pd, err := tmpIP.Marshal()
	assert.Nil(t, err)

	return &pb.TransactionData{
		VmType:  vmType,
		Type:    dataType,
		Amount:  "10",
		Payload: pd,
	}
}

func mockIBTP(t *testing.T, index uint64, typ pb.IBTP_Type) *pb.IBTP {
	content := pb.Content{
		Func: "set",
	}

	bytes, err := content.Marshal()
	assert.Nil(t, err)

	ibtppd, err := json.Marshal(pb.Payload{
		Encrypted: false,
		Content:   bytes,
	})
	assert.Nil(t, err)

	return &pb.IBTP{
		From:    fmt.Sprintf("%s:%s", srcChainID, srcServiceID),
		To:      fmt.Sprintf("%s:%s", dstChainID, dstServiceID),
		Payload: ibtppd,
		Index:   index,
		Type:    typ,
	}
}
