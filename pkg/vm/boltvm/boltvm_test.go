package boltvm

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/validator/mock_validator"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
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
			Name:     "store service",
			Address:  constant.StoreContractAddr.Address().String(),
			Contract: &contracts.Store{},
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
	require.Equal(t, len(registers), 1)

	contract, err := GetBoltContract(constant.StoreContractAddr.Address().String(), registers)
	require.Nil(t, err)

	require.NotNil(t, contract)

}

func TestNewContext(t *testing.T) {
	tx := &pb.BxhTransaction{
		From: types.NewAddressByStr(from),
		To:   types.NewAddressByStr(to),
	}
	tx.TransactionHash = tx.GetHash()
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
	chainLedger := mock_ledger.NewMockChainLedger(ctr)
	stateLedger := mock_ledger.NewMockStateLedger(ctr)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	cons := GetBoltContracts()
	data := make([][]byte, 0)
	data = append(data, []byte("1"))
	proposalData, err := json.Marshal(&contracts.Proposal{
		Id: from + "-0",
	})
	require.Nil(t, err)
	proposals := make([][]byte, 0)
	proposals = append(proposals, proposalData)

	// create Interchain boltVM
	txInterchain := &pb.BxhTransaction{
		From: types.NewAddressByStr(from),
		To:   constant.InterchainContractAddr.Address(),
	}
	txInterchain.TransactionHash = txInterchain.Hash()
	ctxInterchain := vm.NewContext(txInterchain, 1, nil, 100, mockLedger, log.NewWithModule("vm"))
	boltVMInterchain := New(ctxInterchain, mockEngine, nil, cons)
	ibtp := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
	_, err = boltVMInterchain.HandleIBTP(ibtp)
	require.NotNil(t, err)
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
		From:    from,
		To:      from,
		Payload: ibtppd,
		Index:   index,
		Type:    typ,
	}
}
