package contracts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
)

// InterBroker manages all interchain ibtp meta data produced on
// relaychain, similar to broker contract on appchain.
type InterBroker struct {
	boltvm.Stub
}

type CallFunc struct {
	CallF string   `json:"call_f"`
	Args  []string `json:"args"`
}

type InterchainInvoke struct {
	CallFunc CallFunc `json:"call_func"`
	Callback CallFunc `json:"callback"`
	Rollback CallFunc `json:"rollback"`
}

func compositeKeys(key1, key2 string) string {
	return key1 + "-" + key2
}

func outMessageKey(id string) string {
	return fmt.Sprintf("out-%s", id)
}

const (
	InCounterKey       = "InCounter"
	OutCounterKey      = "OutCounter"
	CallbackCounterKey = "CallbackCounter"
)

func (ib *InterBroker) incCounter(id, key string) uint64 {
	counterMap := make(map[string]uint64)
	ib.GetObject(key, &counterMap)
	counterMap[id]++
	ib.SetObject(key, &counterMap)
	return counterMap[id]
}

// GetInMeta returns map[string]map[string]uint64
func (ib *InterBroker) GetInMeta() *boltvm.Response {
	data, err := json.Marshal(ib.getMeta(InCounterKey))
	if err != nil {
		return boltvm.Error(boltvm.BrokerInternalErrCode, fmt.Sprintf(string(boltvm.BrokerInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

// GetOutMeta returns map[string]map[string]uint64
func (ib *InterBroker) GetOutMeta() *boltvm.Response {
	data, err := json.Marshal(ib.getMeta(OutCounterKey))
	if err != nil {
		return boltvm.Error(boltvm.BrokerInternalErrCode, fmt.Sprintf(string(boltvm.BrokerInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

// GetCallbackMeta returns map[string]map[string]uint64
func (ib *InterBroker) GetCallbackMeta() *boltvm.Response {
	data, err := json.Marshal(ib.getMeta(CallbackCounterKey))
	if err != nil {
		return boltvm.Error(boltvm.BrokerInternalErrCode, fmt.Sprintf(string(boltvm.BrokerInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

func (ib *InterBroker) getMeta(key string) map[string]uint64 {
	meta := make(map[string]uint64)
	ok, data := ib.Get(key)
	if !ok {
		return meta
	}
	_ = json.Unmarshal(data, &meta)
	return meta
}

func (ib *InterBroker) EmitInterchain(fromServiceId, toServiceId, funcs, args, argsCb, argsRb string) *boltvm.Response {
	index := ib.incCounter(compositeKeys(toServiceId, fromServiceId), OutCounterKey)
	splitFuncs := strings.Split(funcs, ",")
	if len(splitFuncs) != 3 {
		return boltvm.Error(boltvm.BrokerIllegalFunctionCode, fmt.Sprintf(string(boltvm.BrokerIllegalFunctionMsg), funcs))
	}

	interInvoke := &InterchainInvoke{
		CallFunc: CallFunc{CallF: splitFuncs[0], Args: strings.Split(args, ",")},
		Callback: CallFunc{CallF: splitFuncs[1], Args: strings.Split(argsCb, ",")},
		Rollback: CallFunc{CallF: splitFuncs[2], Args: strings.Split(argsRb, ",")},
	}

	content := &pb.Content{
		Func: interInvoke.CallFunc.CallF,
		Args: argsToByteArray(interInvoke.CallFunc.Args),
	}
	contData, err := content.Marshal()
	if err != nil {
		return boltvm.Error(boltvm.BrokerInternalErrCode, fmt.Sprintf(string(boltvm.BrokerInternalErrMsg), err.Error()))
	}
	ibtp := &pb.IBTP{
		From:    fromServiceId,
		To:      toServiceId,
		Index:   index,
		Type:    pb.IBTP_INTERCHAIN,
		Payload: contData,
	}

	ib.SetObject(outMessageKey(ibtp.ID()), interInvoke)

	data, err := ibtp.Marshal()
	if err != nil {
		return boltvm.Error(boltvm.BrokerInternalErrCode, fmt.Sprintf(string(boltvm.BrokerInternalErrMsg), err.Error()))
	}
	return ib.CrossInvoke(constant.InterchainContractAddr.String(), "HandleIBTPData", pb.Bytes(data))
}

func (ib *InterBroker) InvokeReceipt(input []byte) *boltvm.Response {
	ibtp := &pb.IBTP{}
	err := ibtp.Unmarshal(input)
	if err != nil {
		return boltvm.Error(boltvm.BrokerInternalErrCode, fmt.Sprintf(string(boltvm.BrokerInternalErrMsg), err.Error()))
	}
	ib.incCounter(compositeKeys(ibtp.From, ibtp.To), CallbackCounterKey)
	chainService := strings.Split(ibtp.To, ":")
	if len(chainService) != 3 {
		return boltvm.Error(boltvm.BrokerIllegalIBTPToCode, fmt.Sprintf(string(boltvm.BrokerIllegalIBTPToMsg), ibtp.To))
	}
	interInvoke := &InterchainInvoke{}
	if ok := ib.GetObject(outMessageKey(ibtp.ID()), &interInvoke); !ok {
		return boltvm.Error(boltvm.BrokerNonexistentInterchainInvokeCode, fmt.Sprintf(string(boltvm.BrokerNonexistentInterchainInvokeMsg), ibtp.ID()))
	}

	if ibtp.Category() == pb.IBTP_RESPONSE && interInvoke.CallFunc.CallF == "" {
		return boltvm.Success(nil)
	}
	evmInput := []byte(interInvoke.Callback.Args[0])
	if ibtp.Type == pb.IBTP_RECEIPT_ROLLBACK {
		evmInput = []byte(interInvoke.Rollback.Args[0])
	}
	_ = ib.CrossInvokeEVM(chainService[2], evmInput)
	return boltvm.Success(nil)
}

func (ib *InterBroker) InvokeInterchain(input []byte) *boltvm.Response {
	ibtp := &pb.IBTP{}
	err := ibtp.Unmarshal(input)
	if err != nil {
		return boltvm.Error(boltvm.BrokerInternalErrCode, fmt.Sprintf(string(boltvm.BrokerInternalErrMsg), err.Error()))
	}
	ib.incCounter(compositeKeys(ibtp.From, ibtp.To), InCounterKey)
	chainService := strings.Split(ibtp.To, ":")
	if len(chainService) != 3 {
		return boltvm.Error(boltvm.BrokerIllegalIBTPToCode, fmt.Sprintf(string(boltvm.BrokerIllegalIBTPToMsg), ibtp.To))
	}

	content := &pb.Content{}
	if err := content.Unmarshal(ibtp.GetPayload()); err != nil {
		return boltvm.Error(boltvm.BrokerInternalErrCode, fmt.Sprintf(string(boltvm.BrokerInternalErrMsg), err.Error()))
	}
	newIbtp := &pb.IBTP{
		From:          ibtp.From,
		To:            ibtp.To,
		Index:         ibtp.Index,
		Type:          pb.IBTP_RECEIPT_SUCCESS,
		TimeoutHeight: ibtp.TimeoutHeight,
		Proof:         ibtp.Proof,
		Payload:       ibtp.Payload,
		Group:         ibtp.Group,
		Version:       ibtp.Version,
		Extra:         ibtp.Extra,
	}
	// content args[0] that includes method and args is unpacked by the client,
	// if the result is fail that the client should roll back the ibtp,
	// if the result is successful that the client should call back the ibtp.
	res := ib.CrossInvokeEVM(chainService[2], content.Args[0])
	if !res.Ok {
		if ibtp.Type == pb.IBTP_INTERCHAIN {
			newIbtp.Type = pb.IBTP_RECEIPT_FAILURE
		}
	} else {
		content.Args[0] = res.Result
	}
	data, _ := newIbtp.Marshal()
	return boltvm.Success(data)
}

func (ib *InterBroker) GetInMessage(from, to string, index uint64) *boltvm.Response {
	key := fmt.Sprintf("%s-%s-%d", from, to, index)
	return ib.CrossInvoke(constant.InterchainContractAddr.String(), "GetIBTPById", pb.String(key), pb.Bool(true))
}

func (ib *InterBroker) GetOutMessage(from, to string, index uint64) *boltvm.Response {
	key := fmt.Sprintf("%s-%s-%d", from, to, index)
	interInvoke := &InterchainInvoke{}
	ok := ib.GetObject(outMessageKey(key), &interInvoke)
	if !ok {
		return boltvm.Error(boltvm.BrokerNonexistentOutMsgCode, fmt.Sprintf(string(boltvm.BrokerNonexistentOutMsgMsg), key))
	}
	data, err := json.Marshal(interInvoke.CallFunc)
	if err != nil {
		return boltvm.Error(boltvm.BrokerInternalErrCode, fmt.Sprintf(string(boltvm.BrokerInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

func argsToByteArray(args []string) [][]byte {
	var contentArgs [][]byte
	for _, arg := range args {
		contentArgs = append(contentArgs, []byte(arg))
	}
	return contentArgs
}
