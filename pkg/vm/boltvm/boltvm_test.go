package boltvm

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/agency"
	appchain_mgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/validator/mock_validator"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	from = "0x3f9d18f7C3a6E5E4C0B877FE3E688aB08840b997"
	to   = "0x000018f7C3A6E5E4c0b877fe3E688ab08840b997"
)

func GetBoltContracts() map[string]agency.Contract {
	boltContracts := []*BoltContract{
		{
			Enabled:  true,
			Name:     "interchain manager contract",
			Address:  constant.InterchainContractAddr.Address().String(),
			Contract: &contracts.InterchainManager{},
		},
		{
			Enabled:  true,
			Name:     "store service",
			Address:  constant.StoreContractAddr.Address().String(),
			Contract: &contracts.Store{},
		},
		{
			Enabled:  true,
			Name:     "rule manager service",
			Address:  constant.RuleManagerContractAddr.Address().String(),
			Contract: &contracts.RuleManager{},
		},
		{
			Enabled:  true,
			Name:     "role manager service",
			Address:  constant.RoleContractAddr.Address().String(),
			Contract: &contracts.Role{},
		},
		{
			Enabled:  true,
			Name:     "appchain manager service",
			Address:  constant.AppchainMgrContractAddr.Address().String(),
			Contract: &contracts.AppchainManager{},
		},
		{
			Enabled:  true,
			Name:     "transaction manager service",
			Address:  constant.TransactionMgrContractAddr.Address().String(),
			Contract: &contracts.TransactionManager{},
		},
		{
			Enabled:  true,
			Name:     "asset exchange service",
			Address:  constant.AssetExchangeContractAddr.Address().String(),
			Contract: &contracts.AssetExchange{},
		},
		{
			Enabled:  true,
			Name:     "governance service",
			Address:  constant.GovernanceContractAddr.Address().String(),
			Contract: &contracts.Governance{},
		},
	}

	ContractsInfo := agency.GetRegisteredContractInfo()
	for addr, info := range ContractsInfo {
		boltContracts = append(boltContracts, &BoltContract{
			Enabled:  true,
			Name:     info.Name,
			Address:  addr,
			Contract: info.Constructor(),
		})
	}
	return Register(boltContracts)
}

func TestRegister(t *testing.T) {
	registers := GetBoltContracts()
	require.Equal(t, len(registers), 8)

	contract, err := GetBoltContract(constant.StoreContractAddr.Address().String(), registers)
	require.Nil(t, err)

	require.NotNil(t, contract)

}

func TestNewContext(t *testing.T) {
	tx := &pb.Transaction{
		From: types.NewAddressByStr(from),
		To:   types.NewAddressByStr(to),
	}
	tx.TransactionHash = tx.Hash()
	ctx := NewContext(tx, 1, nil, nil, nil)
	require.Equal(t, from, ctx.Caller())
	require.Equal(t, to, ctx.Callee())
	require.Equal(t, uint64(1), ctx.TransactionIndex())
	require.Equal(t, tx.TransactionHash.String(), ctx.TransactionHash().String())
	require.Nil(t, ctx.Logger())
}

func TestBoltVM_Run(t *testing.T) {
	ctr := gomock.NewController(t)
	mockEngine := mock_validator.NewMockEngine(ctr)
	mockLedger := mock_ledger.NewMockLedger(ctr)

	data := make([][]byte, 0)
	data = append(data, []byte("1"))
	proposalData, err := json.Marshal(&contracts.Proposal{
		Id: from + "-0",
	})
	require.Nil(t, err)
	proposals := make([][]byte, 0)
	proposals = append(proposals, proposalData)
	mockLedger.EXPECT().QueryByPrefix(gomock.Any(), contracts.PROPOSAL_PREFIX).Return(true, proposals).AnyTimes()
	mockLedger.EXPECT().QueryByPrefix(gomock.Any(), appchain_mgr.PREFIX).Return(true, data).AnyTimes()
	mockLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).DoAndReturn(func(addr *types.Address, key []byte) (bool, []byte) {
		switch addr.String() {
		case constant.AppchainMgrContractAddr.String():
			return false, nil
		case constant.InterchainContractAddr.String():
			return false, nil
		}
		return false, nil
	}).Times(1)
	mockLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).DoAndReturn(func(addr *types.Address, key []byte) (bool, []byte) {
		switch addr.String() {
		case constant.AppchainMgrContractAddr.String():
			return true, nil
		case constant.InterchainContractAddr.String():
			return false, nil
		}
		return true, nil
	}).AnyTimes()
	mockLedger.EXPECT().AddState(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLedger.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLedger.EXPECT().AddEvent(gomock.Any()).AnyTimes()

	tx := &pb.Transaction{
		From: types.NewAddressByStr(from),
		To:   constant.AppchainMgrContractAddr.Address(),
	}
	tx.TransactionHash = tx.Hash()
	ctx := vm.NewContext(tx, 1, nil, mockLedger, log.NewWithModule("vm"))
	boltVM := New(ctx, mockEngine, GetBoltContracts())

	ip := &pb.InvokePayload{
		Method: "CountAppchains",
	}
	input, err := ip.Marshal()
	require.Nil(t, err)

	ret, err := boltVM.Run(input)
	require.Nil(t, err)
	require.Equal(t, "1", string(ret))

	ip = &pb.InvokePayload{
		Method: "GetAppchain",
		Args: []*pb.Arg{
			{
				Type:  pb.Arg_U64,
				Value: []byte(strconv.Itoa(1)),
			},
			{
				Type:  pb.Arg_Bytes,
				Value: []byte(strconv.Itoa(1)),
			},
		},
	}
	input, err = ip.Marshal()
	require.Nil(t, err)
	ret, err = boltVM.Run(input)
	require.NotNil(t, err)

	//validators string, consensusType int32, chainType, name, desc, version, pubkey string
	ip = &pb.InvokePayload{
		Method: "Register",
		Args: []*pb.Arg{
			{
				Type:  pb.Arg_String,
				Value: []byte(from),
			},
			{
				Type:  pb.Arg_I32,
				Value: []byte(strconv.Itoa(1)),
			},
			{
				Type:  pb.Arg_String,
				Value: []byte(from),
			},
			{
				Type:  pb.Arg_String,
				Value: []byte(from),
			},
			{
				Type:  pb.Arg_String,
				Value: []byte(from),
			},
			{
				Type:  pb.Arg_String,
				Value: []byte(from),
			},
			{
				Type:  pb.Arg_String,
				Value: []byte(from),
			},
		},
	}
	input, err = ip.Marshal()
	require.Nil(t, err)
	ret, err = boltVM.Run(input)
	require.Nil(t, err)

	ip = &pb.InvokePayload{
		Method: "DeleteAppchain",
		Args: []*pb.Arg{
			{
				Type:    pb.Arg_String,
				IsArray: false,
				Value:   []byte(from),
			},
		},
	}
	input, err = ip.Marshal()
	require.Nil(t, err)
	ret, err = boltVM.Run(input)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "caller is not an admin account")

	ibtp := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
	_, err = boltVM.HandleIBTP(ibtp)
	require.Nil(t, err)

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
		To:        from,
		Payload:   ibtppd,
		Index:     index,
		Type:      typ,
		Timestamp: time.Now().UnixNano(),
	}
}
