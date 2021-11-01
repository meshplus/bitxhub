package contracts

import (
	"encoding/json"
	"fmt"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/pb"
)

type TrustChain struct {
	boltvm.Stub
}

type TrustMeta struct {
	ChainId           string `json:"chain_id"`
	TrustContractAddr string `json:"trust_contract_addr"`
	Method            string `json:"method"`
	Data              []byte `json:"data"`
}

func trustKey(key string) string {
	return fmt.Sprintf("trust-%s", key)
}

func (t *TrustChain) AddTrustMeta(data []byte) *boltvm.Response {
	var trustMeta *TrustMeta
	if err := json.Unmarshal(data, &trustMeta); err != nil {
		return boltvm.Error(boltvm.TrustInternalErrCode, fmt.Sprintf(string(boltvm.TrustInternalErrMsg), err.Error()))
	}
	if len(trustMeta.TrustContractAddr) != 0 {
		return t.CrossInvoke(trustMeta.TrustContractAddr, trustMeta.Method, &pb.Arg{Type: pb.Arg_Bytes, Value: trustMeta.Data})
	}
	t.Set(trustKey(trustMeta.ChainId), trustMeta.Data)
	return boltvm.Success(nil)
}

func (t *TrustChain) GetTrustMeta(key []byte) *boltvm.Response {
	var trustMeta *TrustMeta
	if err := json.Unmarshal(key, &trustMeta); err != nil {
		return boltvm.Error(boltvm.TrustInternalErrCode, fmt.Sprintf(string(boltvm.TrustInternalErrMsg), err.Error()))
	}
	if len(trustMeta.TrustContractAddr) != 0 {
		return t.CrossInvoke(trustMeta.TrustContractAddr, trustMeta.Method, &pb.Arg{Type: pb.Arg_Bytes, Value: trustMeta.Data})
	}
	ok, data := t.Get(trustKey(trustMeta.ChainId))
	if !ok {
		return boltvm.Error(boltvm.TrustNonexistentTrustDataCode, fmt.Sprintf(string(boltvm.TrustNonexistentTrustDataMsg), trustMeta.ChainId))
	}
	return boltvm.Success(data)
}
