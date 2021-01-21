package contracts

import (
	"encoding/json"
	"fmt"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
)

// InterRelayBroker manages all interchain ibtp meta data produced on
// relaychain, similar to broker contract on appchain.
type InterRelayBroker struct {
	boltvm.Stub
}

func combineKey(key1 string, key2 uint64) string {
	return fmt.Sprintf("ibroker-%s-%d", key1, key2)
}

const (
	InCounterKey  = "InCounter"
	OutCounterKey = "OutCounter"
	InMessageKey  = "InMessage"
	OutMessageKey = "OutMessage"
	Locked        = true
)

// IncInCounter increases InCounter[from] by once
func (ibroker *InterRelayBroker) IncInCounter(from string) *boltvm.Response {
	ibroker.incInCounter(from)
	return boltvm.Success(nil)
}

// IncInCounter increases InCounter[from] by once
func (ibroker *InterRelayBroker) incInCounter(from string) {
	InCounterMap := make(map[string]uint64)
	ibroker.GetObject(InCounterKey, &InCounterMap)
	InCounterMap[from]++
	ibroker.SetObject(InCounterKey, InCounterMap)
	// ibroker.Logger().Info("InCounterMap: ", InCounterMap)
}

// GetInCouterMap .
func (ibroker *InterRelayBroker) GetInCouterMap() *boltvm.Response {
	InCounterMap := make(map[string]uint64)
	ibroker.GetObject(InCounterKey, &InCounterMap)
	data, err := json.Marshal(InCounterMap)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

// GetOutCouterMap gets an index map, which implicates the greatest index of
func (ibroker *InterRelayBroker) GetOutCouterMap() *boltvm.Response {
	OutCounterMap := make(map[string]uint64)
	ibroker.GetObject(OutCounterKey, &OutCounterMap)
	data, err := json.Marshal(OutCounterMap)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

// GetOutMessageMap returns interchain-message(ibtp) map
func (ibroker *InterRelayBroker) GetOutMessageMap() *boltvm.Response {
	OutMessage := make(map[string]*pb.IBTP)
	ibroker.GetObject(OutMessageKey, &OutMessage)
	data, err := json.Marshal(OutMessage)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

// GetOutMessage returns interchain ibtp by index and target chain_id
func (ibroker *InterRelayBroker) GetOutMessage(destChain string, index uint64) *boltvm.Response {
	OutMessage := make(map[string]*pb.IBTP)
	ibroker.GetObject(OutMessageKey, &OutMessage)
	data, err := json.Marshal(OutMessage[combineKey(destChain, index)])
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

// RecordIBTPs receives ibtps, adds index for them, and stores in counter and message maps,
// called by inter-relaychain ibtp producer contracts
func (ibroker *InterRelayBroker) RecordIBTPs(ibtpsBytes []byte) *boltvm.Response {
	OutCounterMap := make(map[string]uint64)
	OutMessageMap := make(map[string]*pb.IBTP)
	ibroker.GetObject(OutCounterKey, &OutCounterMap)
	ibroker.GetObject(OutMessageKey, &OutMessageMap)

	ibtps := &pb.IBTPs{}
	err := ibtps.Unmarshal(ibtpsBytes)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	for _, ibtp := range ibtps.Ibtps {
		OutCounterMap[ibtp.To] = OutCounterMap[ibtp.To] + 1
		ibtp.Index = OutCounterMap[ibtp.To]
		OutMessageMap[combineKey(ibtp.To, ibtp.Index)] = ibtp
	}

	ibroker.SetObject(OutCounterKey, OutCounterMap)
	ibroker.SetObject(OutMessageKey, OutMessageMap)

	newIbtps, err := ibtps.Marshal()
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(newIbtps)
}

// InvokeInterRelayContract receives inter-relaychain execution call and invokes
func (ibroker *InterRelayBroker) InvokeInterRelayContract(addr string, fun string, args []byte) *boltvm.Response {
	// ibroker.Logger().Info("In InvokeInterRelayContract....")
	realArgs := [][]byte{}
	invokeArgs := []*pb.Arg{}
	err := json.Unmarshal(args, &realArgs)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	switch addr {
	case constant.MethodRegistryContractAddr.String():
		switch fun {
		case "Synchronize":
			invokeArgs = append(invokeArgs, pb.String(string(realArgs[0])))
			invokeArgs = append(invokeArgs, pb.Bytes(realArgs[1]))
			res := ibroker.CrossInvoke(addr, fun, invokeArgs...)
			if res.Ok {
				ibroker.incInCounter(string(realArgs[0]))
			}
			return res
		}
	}
	return boltvm.Error("Invoke " + addr + "." + fun + " is no supported")
}
