package contracts

import (
	"fmt"
	"strconv"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/pb"
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
	Info   pb.AssetExchangeInfo
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
		Info:   aei,
	}
	t.SetObject(AssetExchangeKey(aei.Id), aer)

	return boltvm.Success(nil)
}

func (t *AssetExchange) Redeem(from, to string, info []byte) *boltvm.Response {
	if err := t.confirm(from, to, info, AssetExchangeRedeem); err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(nil)
}

func (t *AssetExchange) Refund(from, to string, info []byte) *boltvm.Response {
	if err := t.confirm(from, to, info, AssetExchangeRefund); err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(nil)
}

func (t *AssetExchange) confirm(from, to string, payload []byte, status AssetExchangeStatus) error {
	id := string(payload)
	aer := AssetExchangeRecord{}
	ok := t.GetObject(AssetExchangeKey(id), &aer)
	if !ok {
		return fmt.Errorf("asset exchange record does not exist")
	}

	if aer.Status != AssetExchangeInit {
		return fmt.Errorf("asset exchange status for this id is not 'Init'")
	}

	if !(aer.Chain0 == from && aer.Chain1 == to) && !(aer.Chain0 == to && aer.Chain1 == from) {
		return fmt.Errorf("invalid participator of asset exchange id %s", id)
	}

	// TODO: verify proof

	aer.Status = status
	t.SetObject(AssetExchangeKey(id), aer)

	return nil
}

func (t *AssetExchange) GetStatus(id string) *boltvm.Response {
	aer := AssetExchangeRecord{}
	ok := t.GetObject(AssetExchangeKey(id), &aer)
	if !ok {
		return boltvm.Error(fmt.Sprintf("asset exchange record does not exist"))
	}

	return boltvm.Success([]byte(strconv.Itoa(int(aer.Status))))
}

func AssetExchangeKey(id string) string {
	return fmt.Sprintf("%s-%s", ASSET_PREFIX, id)
}

func checkAssetExchangeInfo(aei *pb.AssetExchangeInfo) error {
	if aei.Id == "" ||
		aei.SenderOnDst == "" ||
		aei.ReceiverOnSrc == "" ||
		aei.SenderOnSrc == "" ||
		aei.ReceiverOnDst == "" ||
		aei.AssetOnSrc == 0 ||
		aei.AssetOnDst == 0 {
		return fmt.Errorf("illegal asset exchange info")
	}

	return nil
}
