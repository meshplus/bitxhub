package boltvm

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/validator/mock_validator"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	ctr := gomock.NewController(t)
	mockEngine := mock_validator.NewMockEngine(ctr)
	chainLedger := mock_ledger.NewMockChainLedger(ctr)
	stateLedger := mock_ledger.NewMockStateLedger(ctr)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}
	txInterchain := &pb.BxhTransaction{
		From: types.NewAddressByStr(from),
		To:   constant.InterchainContractAddr.Address(),
	}
	cons := GetBoltContracts()
	ctxInterchain := vm.NewContext(txInterchain, 1, nil, 100, mockLedger, log.NewWithModule("vm"), true, nil)
	boltVMInterchain := New(ctxInterchain, mockEngine, nil, cons)

	stateLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
	stateLedger.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().AddState(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().QueryByPrefix(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().AddEvent(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().PrepareEVM(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Snapshot().AnyTimes()

	boltStub := BoltStubImpl{
		bvm: boltVMInterchain,
		ctx: ctxInterchain,
		ve:  mockEngine,
	}
	caller := boltStub.Caller()
	assert.NotNil(t, caller)

	callee := boltStub.Callee()
	assert.NotNil(t, callee)

	currentCaller := boltStub.CurrentCaller()
	assert.NotNil(t, currentCaller)

	log := boltStub.Logger()
	assert.NotNil(t, log)

	hash := boltStub.GetTxHash()
	assert.NotNil(t, hash)

	timeStamp := boltStub.GetTxTimeStamp()
	assert.NotNil(t, timeStamp)

	txIndex := boltStub.GetTxIndex()
	assert.NotNil(t, txIndex)

	height := boltStub.GetCurrentHeight()
	assert.NotNil(t, height)

	judgeKey := boltStub.Has("key")
	assert.Equal(t, judgeKey, true)

	getRes, data := boltStub.Get("key")
	assert.Equal(t, getRes, true)
	assert.Nil(t, data)

	boltStub.Delete("key")

	var record pb.TransactionRecord
	boltStub.GetObject("key", record)

	value := []byte{'v', 'a', 'l', 'u', 'e'}
	boltStub.Set("key", value)
	boltStub.Add("key", value)

	boltStub.SetObject("key", record)
	boltStub.AddObject("key", record)

	boltStub.Query("key")

	m := make(map[string]uint64)
	boltStub.PostInterchainEvent(m)

	engine := boltStub.ValidationEngine()
	assert.NotNil(t, engine)

	boltStub.GetAccount("0x3f9d18f7C3a6E5E4C0B877FE3E688aB08840b997")

	args := &pb.Arg{
		Type:    6,
		IsArray: false,
		Value:   []byte{'a'},
	}
	boltStub.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "ClearRule", args)

	args = &pb.Arg{
		Type:    0,
		IsArray: false,
		Value:   []byte{'0'},
	}
	boltStub.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "ClearRule", args)

	args.Type = 1
	boltStub.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "ClearRule", args)

	args.Type = 2
	boltStub.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "ClearRule", args)

	args.Type = 3
	boltStub.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "ClearRule", args)

	args.Type = 4
	boltStub.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "ClearRule", args)

	args.Type = 5
	boltStub.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "ClearRule", args)

	args.Type = 7
	boltStub.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "ClearRule", args)

	args.Type = 8
	boltStub.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "ClearRule", args)

}
