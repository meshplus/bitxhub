package contracts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meshplus/bitxhub-kit/crypto"

	"github.com/meshplus/bitxhub-kit/crypto/asym"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
)

type InterchainManager struct {
	boltvm.Stub
}

type Interchain struct {
	ID                   string            `json:"id"`
	InterchainCounter    map[string]uint64 `json:"interchain_counter,omitempty"`
	ReceiptCounter       map[string]uint64 `json:"receipt_counter,omitempty"`
	SourceReceiptCounter map[string]uint64 `json:"source_receipt_counter,omitempty"`
}

type BxhValidators struct {
	Addresses []string `json:"addresses"`
}

func (i *Interchain) UnmarshalJSON(data []byte) error {
	type alias Interchain
	t := &alias{}
	if err := json.Unmarshal(data, t); err != nil {
		return err
	}

	if t.InterchainCounter == nil {
		t.InterchainCounter = make(map[string]uint64)
	}

	if t.ReceiptCounter == nil {
		t.ReceiptCounter = make(map[string]uint64)
	}

	if t.SourceReceiptCounter == nil {
		t.SourceReceiptCounter = make(map[string]uint64)
	}

	*i = Interchain(*t)
	return nil
}

func (x *InterchainManager) Register() *boltvm.Response {
	interchain := &Interchain{ID: x.Caller()}
	ok := x.Has(x.appchainKey(x.Caller()))
	if ok {
		x.GetObject(x.appchainKey(x.Caller()), interchain)
	} else {
		x.SetObject(x.appchainKey(x.Caller()), interchain)
	}
	body, err := json.Marshal(interchain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(body)
}

func (x *InterchainManager) DeleteInterchain(id string) *boltvm.Response {
	x.Delete(x.appchainKey(id))
	return boltvm.Success(nil)
}

// Interchain returns information of the interchain count, Receipt count and SourceReceipt count
func (x *InterchainManager) Interchain() *boltvm.Response {
	ok, data := x.Get(x.appchainKey(x.Caller()))
	if !ok {
		return boltvm.Error(fmt.Errorf("this appchain does not exist").Error())
	}
	return boltvm.Success(data)
}

// GetInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
func (x *InterchainManager) GetInterchain(id string) *boltvm.Response {
	ok, data := x.Get(x.appchainKey(id))
	if !ok {
		return boltvm.Error(fmt.Errorf("this appchain does not exist").Error())
	}
	return boltvm.Success(data)
}

func (x *InterchainManager) HandleIBTP(data []byte) *boltvm.Response {
	ibtp := &pb.IBTP{}
	if err := ibtp.Unmarshal(data); err != nil {
		return boltvm.Error(err.Error())
	}

	if len(strings.Split(ibtp.From, "-")) == 2 {
		return x.handleUnionIBTP(ibtp)
	}

	ok := x.Has(x.appchainKey(x.Caller()))
	if !ok {
		return boltvm.Error("this appchain does not exist")
	}

	interchain := &Interchain{}
	x.GetObject(x.appchainKey(ibtp.From), &interchain)

	if err := x.checkIBTP(ibtp, interchain); err != nil {
		return boltvm.Error(err.Error())
	}

	res := boltvm.Success(nil)

	if pb.IBTP_INTERCHAIN == ibtp.Type {
		res = x.beginTransaction(ibtp)
	} else if pb.IBTP_RECEIPT_SUCCESS == ibtp.Type || pb.IBTP_RECEIPT_FAILURE == ibtp.Type {
		res = x.reportTransaction(ibtp)
	} else if pb.IBTP_ASSET_EXCHANGE_INIT == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REDEEM == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REFUND == ibtp.Type {
		res = x.handleAssetExchange(ibtp)
	}

	if !res.Ok {
		return res
	}

	x.ProcessIBTP(ibtp, interchain)

	return res
}

func (x *InterchainManager) HandleIBTPs(data []byte) *boltvm.Response {
	ok := x.Has(x.appchainKey(x.Caller()))
	if !ok {
		return boltvm.Error("this appchain does not exist")
	}

	ibtps := &pb.IBTPs{}
	if err := ibtps.Unmarshal(data); err != nil {
		return boltvm.Error(err.Error())
	}

	interchain := &Interchain{}
	x.GetObject(x.appchainKey(x.Caller()), &interchain)

	for _, ibtp := range ibtps.Iptp {
		if err := x.checkIBTP(ibtp, interchain); err != nil {
			return boltvm.Error(err.Error())
		}
	}

	if res := x.beginMultiTargetsTransaction(ibtps); !res.Ok {
		return res
	}

	for _, ibtp := range ibtps.Iptp {
		x.ProcessIBTP(ibtp, interchain)
	}

	return boltvm.Success(nil)
}

func (x *InterchainManager) checkIBTP(ibtp *pb.IBTP, interchain *Interchain) error {
	if ibtp.To == "" {
		return fmt.Errorf("empty destination chain id")
	}
	if ok := x.Has(x.appchainKey(ibtp.To)); !ok {
		x.Logger().WithField("chain_id", ibtp.To).Debug("target appchain does not exist")
	}

	app := &appchainMgr.Appchain{}
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(ibtp.From))
	if err := json.Unmarshal(res.Result, app); err != nil {
		return err
	}

	// get validation rule contract address
	res = x.CrossInvoke(constant.RuleManagerContractAddr.String(), "GetRuleAddress", pb.String(ibtp.From), pb.String(app.ChainType))
	if !res.Ok {
		return fmt.Errorf("this appchain does not register rule")
	}

	// handle validation
	isValid, err := x.ValidationEngine().Validate(string(res.Result), ibtp.From, ibtp.Proof, ibtp.Payload, app.Validators)
	if err != nil {
		return err
	}

	if !isValid {
		return fmt.Errorf("invalid interchain transaction")
	}

	if pb.IBTP_INTERCHAIN == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_INIT == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REDEEM == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REFUND == ibtp.Type {
		if ibtp.From != x.Caller() {
			return fmt.Errorf("ibtp from != caller")
		}

		idx := interchain.InterchainCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			return fmt.Errorf(fmt.Sprintf("wrong index, required %d, but %d", idx+1, ibtp.Index))
		}
	} else {
		if ibtp.To != x.Caller() {
			return fmt.Errorf("ibtp to != caller")
		}

		idx := interchain.ReceiptCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			if interchain.SourceReceiptCounter[ibtp.To]+1 != ibtp.Index {
				return fmt.Errorf("wrong receipt index, required %d, but %d", idx+1, ibtp.Index)
			}
		}
	}

	return nil
}

func (x *InterchainManager) ProcessIBTP(ibtp *pb.IBTP, interchain *Interchain) {
	m := make(map[string]uint64)

	if pb.IBTP_INTERCHAIN == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_INIT == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REDEEM == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REFUND == ibtp.Type {
		interchain.InterchainCounter[ibtp.To]++
		x.SetObject(x.appchainKey(ibtp.From), interchain)
		x.SetObject(x.indexMapKey(ibtp.ID()), x.GetTxHash())
		m[ibtp.To] = x.GetTxIndex()
	} else {
		interchain.ReceiptCounter[ibtp.To] = ibtp.Index
		x.SetObject(x.appchainKey(ibtp.From), interchain)
		m[ibtp.From] = x.GetTxIndex()

		ic := &Interchain{}
		x.GetObject(x.appchainKey(ibtp.To), &ic)
		ic.SourceReceiptCounter[ibtp.From] = ibtp.Index
		x.SetObject(x.appchainKey(ibtp.To), ic)
	}

	x.PostInterchainEvent(m)
}

func (x *InterchainManager) beginMultiTargetsTransaction(ibtps *pb.IBTPs) *boltvm.Response {
	args := make([]*pb.Arg, 0)
	globalId := fmt.Sprintf("%s-%s", x.Caller(), x.GetTxHash())
	args = append(args, pb.String(globalId))

	for _, ibtp := range ibtps.Iptp {
		if ibtp.Type != pb.IBTP_INTERCHAIN {
			return boltvm.Error("ibtp type != IBTP_INTERCHAIN")
		}

		childTxId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
		args = append(args, pb.String(childTxId))
	}

	return x.CrossInvoke(constant.TransactionMgrContractAddr.String(), "BeginMultiTXs", args...)
}

func (x *InterchainManager) beginTransaction(ibtp *pb.IBTP) *boltvm.Response {
	txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
	return x.CrossInvoke(constant.TransactionMgrContractAddr.String(), "Begin", pb.String(txId))
}

func (x *InterchainManager) reportTransaction(ibtp *pb.IBTP) *boltvm.Response {
	txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
	result := int32(0)
	if ibtp.Type == pb.IBTP_RECEIPT_FAILURE {
		result = 1
	}
	return x.CrossInvoke(constant.TransactionMgrContractAddr.String(), "Report", pb.String(txId), pb.Int32(result))
}

func (x *InterchainManager) handleAssetExchange(ibtp *pb.IBTP) *boltvm.Response {
	var method string

	switch ibtp.Type {
	case pb.IBTP_ASSET_EXCHANGE_INIT:
		method = "Init"
	case pb.IBTP_ASSET_EXCHANGE_REDEEM:
		method = "Redeem"
	case pb.IBTP_ASSET_EXCHANGE_REFUND:
		method = "Refund"
	default:
		return boltvm.Error("unsupported asset exchange type")
	}

	return x.CrossInvoke(constant.AssetExchangeContractAddr.String(), method, pb.String(ibtp.From),
		pb.String(ibtp.To), pb.Bytes(ibtp.Extra))
}

func (x *InterchainManager) GetIBTPByID(id string) *boltvm.Response {
	arr := strings.Split(id, "-")
	if len(arr) != 3 {
		return boltvm.Error("wrong ibtp id")
	}

	caller := x.Caller()

	if caller != arr[0] && caller != arr[1] {
		return boltvm.Error("The caller does not have access to this ibtp")
	}

	var hash types.Hash
	exist := x.GetObject(x.indexMapKey(id), &hash)
	if !exist {
		return boltvm.Error("this id is not existed")
	}

	return boltvm.Success(hash.Bytes())
}

func (x *InterchainManager) handleUnionIBTP(ibtp *pb.IBTP) *boltvm.Response {
	srcRelayChainID := strings.Split(ibtp.From, "-")[0]
	ok := x.Has(x.appchainKey(srcRelayChainID))
	if !ok {
		return boltvm.Error("this relay chain does not exist")
	}

	if ibtp.To == "" {
		return boltvm.Error("empty destination chain id")
	}
	if ok := x.Has(x.appchainKey(ibtp.To)); !ok {
		return boltvm.Error(fmt.Sprintf("target appchain does not exist: %s", ibtp.To))
	}

	app := &appchainMgr.Appchain{}
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(srcRelayChainID))
	if err := json.Unmarshal(res.Result, app); err != nil {
		return boltvm.Error(err.Error())
	}

	interchain := &Interchain{
		ID: ibtp.From,
	}
	ok = x.Has(x.appchainKey(ibtp.From))
	if !ok {
		x.SetObject(x.appchainKey(ibtp.From), interchain)
	}
	x.GetObject(x.appchainKey(ibtp.From), &interchain)

	if err := x.checkUnionIBTP(app, ibtp, interchain); err != nil {
		return boltvm.Error(err.Error())
	}

	x.ProcessIBTP(ibtp, interchain)
	return boltvm.Success(nil)
}

func (x *InterchainManager) checkUnionIBTP(app *appchainMgr.Appchain, ibtp *pb.IBTP, interchain *Interchain) error {
	if pb.IBTP_INTERCHAIN == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_INIT == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REDEEM == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REFUND == ibtp.Type {

		idx := interchain.InterchainCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			return fmt.Errorf(fmt.Sprintf("wrong index, required %d, but %d", idx+1, ibtp.Index))
		}
	} else {
		idx := interchain.ReceiptCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			if interchain.SourceReceiptCounter[ibtp.To]+1 != ibtp.Index {
				return fmt.Errorf("wrong receipt index, required %d, but %d", idx+1, ibtp.Index)
			}
		}
	}

	if "" == app.Validators {
		return fmt.Errorf("empty validators in relay chain:%s", app.ID)
	}
	var validators BxhValidators
	if err := json.Unmarshal([]byte(app.Validators), &validators); err != nil {
		return err
	}

	m := make(map[string]struct{}, 0)
	for _, validator := range validators.Addresses {
		m[validator] = struct{}{}
	}

	var signs pb.SignResponse
	if err := signs.Unmarshal(ibtp.Proof); err != nil {
		return err
	}

	threshold := (len(validators.Addresses) - 1) / 3
	counter := 0

	hash := sha256.Sum256([]byte(ibtp.Hash().String()))
	for v, sign := range signs.Sign {
		if _, ok := m[v]; !ok {
			return fmt.Errorf("wrong validator: %s", v)
		}
		delete(m, v)
		addr := types.String2Address(v)
		ok, _ := asym.Verify(crypto.Secp256k1, sign, hash[:], addr)
		if ok {
			counter++
		}
		if counter > threshold {
			return nil
		}
	}
	return fmt.Errorf("multi signs verify fail, counter:%d", counter)

}

func (x *InterchainManager) appchainKey(id string) string {
	return appchainMgr.PREFIX + id
}

func (x *InterchainManager) indexMapKey(id string) string {
	return fmt.Sprintf("index-tx-%s", id)
}
