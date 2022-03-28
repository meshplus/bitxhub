package contracts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxid"
)

const (
	SUPERVISE_PREFIX = "supervise"
)

type SuperviseManager struct {
	boltvm.Stub
}

type SuperviseInfo struct {
	SubmitChainId  string   `json:"submit_chain_id"`
	IssuePath      []string `json:"issue_path"`
	CurrentChainId string   `json:"current_chain_id"`
}

func (sr *SuperviseManager) Record(from, to string, extra []byte) *boltvm.Response {
	//check current caller
	if !strings.EqualFold(sr.CurrentCaller(), constant.InterchainContractAddr.String()) {
		return boltvm.Error(boltvm.SuperviseNoPermissionCode, fmt.Sprintf(string(boltvm.SuperviseNoPermissionMsg), sr.CurrentCaller()))
	}

	//if extra == nil {
	//	return boltvm.Error(boltvm.SuperviseNoExtraInfoCode, fmt.Sprintf(string(boltvm.SuperviseNoExtraInfoMsg)))
	//}
	index := string(extra)
	if err := checkIndexFormat(index); err != nil {
		return boltvm.Error(boltvm.SuperviseIllegalIndexFormatCode, fmt.Sprintf(string(boltvm.SuperviseIllegalIndexFormatMsg), index, err.Error()))
	}
	info := SuperviseInfo{}
	if sr.Has(SuperviseKey(index)) {
		if !sr.GetObject(SuperviseKey(index), &info) {
			return boltvm.Error(boltvm.SuperviseGetSuperviseInfoErrCode, fmt.Sprintf(string(boltvm.SuperviseGetSuperviseInfoErrMsg)))
		}
		toChainId, err := sr.getChainId(to)
		if err != nil {
			return boltvm.Error(boltvm.SuperviseGetChainIdErrCode, fmt.Sprintf(string(boltvm.SuperviseGetChainIdErrMsg), to, err.Error()))
		}
		// check if recorded
		for _, chainId := range info.IssuePath {
			if chainId == toChainId {
				// issue repeatedly
				return boltvm.Success(nil)
			}
		}
		info.IssuePath = append(info.IssuePath, toChainId)
		info.CurrentChainId = toChainId
	} else {
		fromChainId, err := sr.getChainId(from)
		if err != nil {
			return boltvm.Error(boltvm.SuperviseGetChainIdErrCode, fmt.Sprintf(string(boltvm.SuperviseGetChainIdErrMsg), from, err.Error()))
		}
		toChainId, err := sr.getChainId(to)
		if err != nil {
			return boltvm.Error(boltvm.SuperviseGetChainIdErrCode, fmt.Sprintf(string(boltvm.SuperviseGetChainIdErrMsg), to, err.Error()))
		}
		info.SubmitChainId = fromChainId
		info.IssuePath = append(info.IssuePath, toChainId)
		info.CurrentChainId = toChainId
	}
	sr.SetObject(SuperviseKey(index), info)
	return boltvm.Success(nil)
}

func (sr *SuperviseManager) GetIssuePath(index string) *boltvm.Response {
	if err := checkIndexFormat(index); err != nil {
		return boltvm.Error(boltvm.SuperviseIllegalIndexFormatCode, fmt.Sprintf(string(boltvm.SuperviseIllegalIndexFormatMsg), index, err.Error()))
	}

	if !sr.Has(SuperviseKey(index)) {
		return boltvm.Error(boltvm.SuperviseNonexistentSuperviseInfoCode, fmt.Sprintf(string(boltvm.SuperviseNonexistentSuperviseInfoMsg), index))
	}

	info := SuperviseInfo{}
	if !sr.GetObject(SuperviseKey(index), &info) {
		return boltvm.Error(boltvm.SuperviseGetSuperviseInfoErrCode, fmt.Sprintf(string(boltvm.SuperviseGetSuperviseInfoErrMsg)))
	}

	issuePathBytes, err := json.Marshal(info.IssuePath)
	if err != nil {
		msg := "marshal issuePath error: %s" + err.Error()
		return boltvm.Error(boltvm.SuperviseInternalErrCode, fmt.Sprintf(string(boltvm.SuperviseInternalErrMsg), msg))
	}

	return boltvm.Success(issuePathBytes)
}

func (sr *SuperviseManager) GetCurrentChainId(index string) *boltvm.Response {
	if err := checkIndexFormat(index); err != nil {
		return boltvm.Error(boltvm.SuperviseIllegalIndexFormatCode, fmt.Sprintf(string(boltvm.SuperviseIllegalIndexFormatMsg), index, err.Error()))
	}

	if !sr.Has(SuperviseKey(index)) {
		return boltvm.Error(boltvm.SuperviseNonexistentSuperviseInfoCode, fmt.Sprintf(string(boltvm.SuperviseNonexistentSuperviseInfoMsg), index))
	}

	info := SuperviseInfo{}
	if !sr.GetObject(SuperviseKey(index), &info) {
		return boltvm.Error(boltvm.SuperviseGetSuperviseInfoErrCode, fmt.Sprintf(string(boltvm.SuperviseGetSuperviseInfoErrMsg)))
	}

	return boltvm.Success([]byte(info.CurrentChainId))
}

func (sr *SuperviseManager) getChainId(did string) (string, error) {
	splits := strings.Split(did, ":")
	methodDID := "did:bitxhub:" + splits[2] + ":."
	resp := sr.CrossInvoke(constant.MethodRegistryContractAddr.String(), "ResolveWithDoc", pb.String(methodDID))
	if !resp.Ok {
		return "", fmt.Errorf("get method doc from %s failed: %s", methodDID, string(resp.Result))
	}
	var methodDoc *bitxid.MethodDoc
	if err := json.Unmarshal(resp.Result, &methodDoc); err != nil {
		return "", fmt.Errorf("unmarshal method doc error: %w", err)
	}
	addr, err := pb.GetAddrFromDoc(methodDoc)
	if err != nil {
		return "", err
	}
	resp = sr.CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetChainIdByAdmin", pb.String(addr))
	if !resp.Ok {
		return "", fmt.Errorf(string(resp.Result))
	}
	return string(resp.Result), nil
}

func checkIndexFormat(index string) error {
	splits := strings.Split(index, ":")
	if len(splits) != 4 || splits[0] != "did" || splits[1] != "bitxhub" || splits[2] == "" || splits[3] == "" {
		return fmt.Errorf("invalid index format")
	}
	return nil
}

func SuperviseKey(index string) string {
	return fmt.Sprintf("%s-%s", SUPERVISE_PREFIX, index)
}
