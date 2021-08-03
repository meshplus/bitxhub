package contracts

import (
	"encoding/json"
	"fmt"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"strings"
)

// InterBroker manages all interchain ibtp meta data produced on
// relaychain, similar to broker contract on appchain.
type InterBroker struct {
	boltvm.Stub
}

func compositeKeys(key1, key2 string) string {
	return key1 + "-" + key2
}

func ibtpReqKey(key1 string, key2 string, key3 uint64) string {
	return fmt.Sprintf("req-%s-%s-%d", key1, key2, key3)
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
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

// GetOutMeta returns map[string]map[string]uint64
func (ib *InterBroker) GetOutMeta() *boltvm.Response {
	data, err := json.Marshal(ib.getMeta(OutCounterKey))
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

// GetCallbackMeta returns map[string]map[string]uint64
func (ib *InterBroker) GetCallbackMeta() *boltvm.Response {
	data, err := json.Marshal(ib.getMeta(CallbackCounterKey))
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

func (ib *InterBroker) getMeta(key string) map[string]uint64 {
	meta := make(map[string]uint64)
	ok, data := ib.Get(key)
	if !ok {
		return meta
	}
	json.Unmarshal(data, &meta)
	return meta
}

func (ib *InterBroker) EmitInterchain(fromServiceId, toServiceId, funcs, args, argsCb, argsRb string) *boltvm.Response {
	index := ib.incCounter(compositeKeys(toServiceId, fromServiceId), OutCounterKey)
	splitFuncs := strings.Split(funcs, ",")
	if len(splitFuncs) != 3 {
		return boltvm.Error("funcs should be (func,funcCb,funcRb)")
	}
	content := &pb.Content{
		Func:     splitFuncs[0],
		Args:     argsToByteArray(strings.Split(args, ",")),
		Callback: splitFuncs[1],
		ArgsCb:   argsToByteArray(strings.Split(argsCb, ",")),
		Rollback: splitFuncs[2],
		ArgsRb:   argsToByteArray(strings.Split(argsRb, ",")),
	}
	contData, err := content.Marshal()
	if err != nil {
		return boltvm.Error(err.Error())
	}
	ibtp := &pb.IBTP{
		From:    fromServiceId,
		To:      toServiceId,
		Index:   index,
		Type:    pb.IBTP_INTERCHAIN,
		Payload: contData,
	}

	data, err := ibtp.Marshal()
	if err != nil {
		return boltvm.Error(err.Error())
	}
	ib.Set(ibtpReqKey(ibtp.From, ibtp.To, ibtp.Index), data)
	return ib.CrossInvoke(constant.InterchainContractAddr.String(), "HandleIBTPData", &pb.Arg{Type: pb.Arg_Bytes, Value: data})
}

func (ib *InterBroker) InvokeInterchain(input []byte) *boltvm.Response {
	ibtp := &pb.IBTP{}
	err := ibtp.Unmarshal(input)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	if ibtp.Category() == pb.IBTP_RESPONSE {
		ib.incCounter(compositeKeys(ibtp.From, ibtp.To), CallbackCounterKey)
		if len(ibtp.GetPayload()) == 0 {
			return boltvm.Success(nil)
		}
	} else {
		ib.incCounter(compositeKeys(ibtp.From, ibtp.To), InCounterKey)
	}
	chainService := strings.Split(ibtp.To, ":")
	if len(chainService) != 3 {
		return boltvm.Error("ibtp.To is not chain service")
	}

	content := &pb.Content{}
	if err := content.Unmarshal(ibtp.GetPayload()); err != nil {
		return boltvm.Error(err.Error())
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
	}
	data, _ := newIbtp.Marshal()
	return boltvm.Success(data)
}

func (ib *InterBroker) GetInMessage(from, to string, index uint64) *boltvm.Response {
	key := fmt.Sprintf("%s-%s-%d", from, to, index)
	return ib.CrossInvoke(constant.InterchainContractAddr.String(), "GetIBTPById", &pb.Arg{Type: pb.Arg_String, Value: []byte(key)}, &pb.Arg{Type: pb.Arg_Bool, Value: []byte("true")})
}

func (ib *InterBroker) GetOutMessage(from, to string, index uint64) *boltvm.Response {
	ok, data := ib.Get(ibtpReqKey(from, to, index))
	if !ok {
		return boltvm.Error("not found ibtp")
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
