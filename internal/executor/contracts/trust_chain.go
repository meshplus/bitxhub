package contracts

import (
	"fmt"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/pb"
)

type TrustChain struct {
	boltvm.Stub
}

func trustKey(key string) string {
	return fmt.Sprintf("trust-%s", key)
}

func (t *TrustChain) AddTrustMeta(data []byte) *boltvm.Response {
	trustMeta := &pb.TrustMeta{}
	if err := trustMeta.Unmarshal(data); err != nil {
		return boltvm.Error(boltvm.TrustInternalErrCode, err.Error())
	}
	if len(trustMeta.TrustContractAddr) != 0 {
		return t.CrossInvoke(trustMeta.TrustContractAddr, trustMeta.Method, &pb.Arg{Type: pb.Arg_Bytes, Value: trustMeta.Data})
	}
	t.Set(trustKey(trustMeta.ChainId), trustMeta.Data)
	return boltvm.Success(nil)
}

func (t *TrustChain) GetTrustMeta(key []byte) *boltvm.Response {
	trustMeta := &pb.TrustMeta{}
	if err := trustMeta.Unmarshal(key); err != nil {
		return boltvm.Error(boltvm.TrustInternalErrCode, err.Error())
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
