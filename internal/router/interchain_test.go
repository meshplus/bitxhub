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
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	from  = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b991"
	to    = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b992"
	other = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b993"
)

func TestInterchainRouter_AddPier(t *testing.T) {
	appchains := make([]*appchain_mgr.Appchain, 0)

	var ret [][]byte

	app := &appchain_mgr.Appchain{
		ID:   from,
		Name: "app",
	}
	bxh := &appchain_mgr.Appchain{
		ID:   to,
		Name: "bxh",
	}

	appchains = append(appchains, app, bxh)
	for _, appchain := range appchains {
		data, err := json.Marshal(appchain)
		require.Nil(t, err)
		ret = append(ret, data)
	}

	mockCtl := gomock.NewController(t)
	mockLedger := mock_ledger.NewMockLedger(mockCtl)
	mockLedger.EXPECT().QueryByPrefix(constant.AppchainMgrContractAddr.Address(), appchain_mgr.PREFIX).Return(true, ret)

	mockPeerMgr := mock_peermgr.NewMockPeerManager(mockCtl)

	router, err := New(log.NewWithModule("router"), nil, mockLedger, mockPeerMgr, 1)
	require.Nil(t, err)

	require.Nil(t, router.Start())

	interchainWrappersC, err := router.AddPier(to, true)
	require.Nil(t, err)

	var txs []*pb.Transaction
	// set tx of TransactionData_BVM type
	ibtp1 := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
	BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp1)
	BVMTx := mockTx(BVMData)
	txs = append(txs, BVMTx)

	m := make(map[string]*pb.Uint64Slice, 0)

	m[other] = &pb.Uint64Slice{
		Slice: []uint64{0},
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
		require.Equal(t, iw.InterchainTxWrappers[0].Transactions[0].Hash().String(), BVMTx.Hash().String())
	default:
		require.Errorf(t, fmt.Errorf("not found interchainWrappers"), "")
	}

	router.RemovePier(to, true)

	require.Nil(t, router.Stop())
}

func mockBlock(blockNumber uint64, txs []*pb.Transaction) *pb.Block {
	header := &pb.BlockHeader{
		Number:    blockNumber,
		Timestamp: time.Now().UnixNano(),
	}
	return &pb.Block{
		BlockHeader:  header,
		Transactions: txs,
	}
}

func mockTx(data *pb.TransactionData) *pb.Transaction {
	return &pb.Transaction{
		Data:  data,
		Nonce: rand.Int63(),
	}
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
		Amount:  10,
		Payload: pd,
	}
}

func mockIBTP(t *testing.T, index uint64, typ pb.IBTP_Type) *pb.IBTP {
	content := pb.Content{
		SrcContractId: from,
		DstContractId: from,
		Func:          "set",
	}

	bytes, err := content.Marshal()
	assert.Nil(t, err)

	ibtppd, err := json.Marshal(pb.Payload{
		Encrypted: false,
		Content:   bytes,
	})
	assert.Nil(t, err)

	return &pb.IBTP{
		From:      from,
		To:        other,
		Payload:   ibtppd,
		Index:     index,
		Type:      typ,
		Timestamp: time.Now().UnixNano(),
	}
}
