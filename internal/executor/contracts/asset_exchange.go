package contracts

import (
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
)

type AssetExchange struct {
	boltvm.Stub
}

type AssetExchangeStatus uint64
type AssetExchangeType uint64

type AssetExchangeRecord struct {
	Chain0 string
	Chain1 string
	Status AssetExchangeStatus
	Req    pb.AssetExchangeInfo
	Resp   pb.AssetExchangeInfo
}

const (
	ASSET_PREFIX                            = "asset-"
	AssetExchangeInit   AssetExchangeStatus = 0
	AssetExchangeRedeem AssetExchangeStatus = 1
	AssetExchangeRefund AssetExchangeStatus = 2
)

func (t *AssetExchange) Init(from, to string, info []byte) *boltvm.Response {
	aei := pb.AssetExchangeInfo{}
	if err := aei.Unmarshal(info); err != nil {
		return boltvm.Error(err.Error())
	}

	if t.Has(AssetExchangeKey(aei.Id)) {
		return boltvm.Error(fmt.Sprintf("asset exhcange id already exists"))
	}

	if err := checkAssetExchangeInfo(&aei); err != nil {
		return boltvm.Error(err.Error())
	}

	aer := AssetExchangeRecord{
		Chain0: from,
		Chain1: to,
		Status: AssetExchangeInit,
		Req:    aei,
	}
	t.SetObject(AssetExchangeKey(aei.Id), aer)

	return boltvm.Success(nil)
}

func (t *AssetExchange) Redeem(from, to string, info []byte) *boltvm.Response {
	aei := pb.AssetExchangeInfo{}
	if err := aei.Unmarshal(info); err != nil {
		return boltvm.Error(err.Error())
	}

	aer := AssetExchangeRecord{}
	ok := t.GetObject(AssetExchangeKey(aei.Id), &aer)
	if !ok {
		return boltvm.Error(fmt.Sprintf("counterparty's asset exchange info does not exist"))
	}

	if aer.Chain0 != to || aer.Chain1 != from {
		return boltvm.Error(fmt.Sprintf("invalid chain info for current asset exchange id"))
	}

	if aer.Status != AssetExchangeInit {
		return boltvm.Error(fmt.Sprintf("asset exchange status for this id is not 'Init'"))
	}

	if err := checkAssetExchangeInfo(&aei); err != nil {
		return boltvm.Error(err.Error())
	}

	if err := checkAssetExchangeInfoPair(&aer.Req, &aei); err != nil {
		return boltvm.Error(err.Error())
	}

	aer.Resp = aei
	aer.Status = AssetExchangeRedeem
	t.SetObject(AssetExchangeKey(aei.Id), aer)

	return boltvm.Success(nil)
}

func (t *AssetExchange) Refund(from, to string, payload []byte) *boltvm.Response {
	id := string(payload)
	aer := AssetExchangeRecord{}
	ok := t.GetObject(AssetExchangeKey(id), &aer)
	if !ok {
		return boltvm.Error(fmt.Sprintf("asset exchange record does not exist"))
	}

	if aer.Status != AssetExchangeInit {
		return boltvm.Error(fmt.Sprintf("asset exchange status for this id is not 'Init'"))
	}

	if !(aer.Chain0 == from && aer.Chain1 == to) && !(aer.Chain0 == to && aer.Chain1 == from) {
		return boltvm.Error(fmt.Sprintf("invalid participator of asset exchange id %s", id))
	}

	aer.Status = AssetExchangeRefund
	t.SetObject(AssetExchangeKey(id), aer)

	return boltvm.Success(nil)
}

func (t *AssetExchange) GetStatus(id string) *boltvm.Response {
	aer := AssetExchangeRecord{}
	ok := t.GetObject(AssetExchangeKey(id), &aer)
	if !ok {
		return boltvm.Error(fmt.Sprintf("asset exchange record does not exist"))
	}

	return boltvm.Success([]byte(fmt.Sprintf("%s-%d", id, aer.Status)))
}

func AssetExchangeKey(id string) string {
	return fmt.Sprintf("%s-%s", ASSET_PREFIX, id)
}

func checkAssetExchangeInfo(aei *pb.AssetExchangeInfo) error {
	if aei.SenderOnDst == "" ||
		aei.ReceiverOnSrc == "" ||
		aei.SenderOnSrc == "" ||
		aei.ReceiverOnDst == "" ||
		aei.AssetOnSrc == 0 ||
		aei.AssetOnDst == 0 {
		return fmt.Errorf("illegal asset exchange info")
	}

	return nil
}

func checkAssetExchangeInfoPair(aei0, aei1 *pb.AssetExchangeInfo) error {
	if aei0.SenderOnSrc != aei1.ReceiverOnDst ||
		aei0.ReceiverOnSrc != aei1.SenderOnDst ||
		aei0.SenderOnDst != aei1.ReceiverOnSrc ||
		aei0.ReceiverOnDst != aei1.SenderOnSrc ||
		aei0.AssetOnSrc != aei1.AssetOnDst ||
		aei0.AssetOnDst != aei1.AssetOnSrc {
		return fmt.Errorf("unmatched exchange info pair")
	}

	return nil
}
