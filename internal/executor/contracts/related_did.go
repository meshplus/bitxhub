package contracts

import (
	"encoding/json"
	"fmt"
	"github.com/meshplus/bitxhub-core/boltvm"
	"strings"
)

type DIDManager struct {
	boltvm.Stub
}

type RelatedDID struct {
	XDID DID   `json:"x_did"`
	DIDs []DID `json:"dids"`
}

type DID struct {
	DID     string  `json:"did"`
	Account Account `json:"account"`
}

type Account struct {
	Address   string `json:"address"`
	PublicKey string `json:"publicKey"`
	Version   string `json:"version"`
	Algo      string `json:"algo"`
}

func (r *DIDManager) AddRelatedDIDs(relatedDID []byte) *boltvm.Response {
	var rd *RelatedDID
	if err := json.Unmarshal(relatedDID, &rd); err != nil {
		return boltvm.Error(err.Error())
	}
	if _, err := checkDID(rd.XDID.DID); err != nil {
		return boltvm.Error(err.Error())
	}
	for _, did := range rd.DIDs {
		if _, err := checkDID(did.DID); err != nil {
			return boltvm.Error(err.Error())
		}
	}
	r.Set(rd.XDID.DID, relatedDID)
	for _, did := range rd.DIDs {
		r.Set(did.DID, []byte(rd.XDID.DID))
	}
	return boltvm.Success(nil)
}

func (r *DIDManager) QueryDIDByTargetChainId(didCaller string, targetChainID string) *boltvm.Response {
	ok, xDID := r.Get(didCaller)
	if !ok {
		return boltvm.Error(fmt.Sprintf("not found xDID by:%s", didCaller))
	}
	ok, relatedDID := r.Get(string(xDID))
	if !ok {
		return boltvm.Error(fmt.Sprintf("not found relatedDID by:%s", xDID))
	}
	var rd *RelatedDID
	if err := json.Unmarshal(relatedDID, &rd); err != nil {
		return boltvm.Error(err.Error())
	}
	for _, did := range rd.DIDs {
		chainID, _ := checkDID(did.DID)
		if strings.EqualFold(chainID, targetChainID) {
			return boltvm.Success([]byte(did.DID))
		}
	}
	return boltvm.Error(fmt.Sprintf("not found did by %s and %s", didCaller, targetChainID))
}

func checkDID(did string) (string, error) {
	didSplits := strings.Split(did, ":")
	if len(didSplits) != 4 {
		return "", fmt.Errorf("no standard did format:%s", did)
	}
	return didSplits[2], nil
}
