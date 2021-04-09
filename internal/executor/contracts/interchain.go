package contracts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
)

type InterchainManager struct {
	boltvm.Stub
}

type BxhValidators struct {
	Addresses []string `json:"addresses"`
}

func (x *InterchainManager) Register(chainId string) *boltvm.Response {
	interchain, ok := x.getInterchain(chainId)
	if !ok {
		interchain = &pb.Interchain{
			ID:                   chainId,
			InterchainCounter:    make(map[string]uint64),
			ReceiptCounter:       make(map[string]uint64),
			SourceReceiptCounter: make(map[string]uint64),
		}
		x.setInterchain(chainId, interchain)
	}
	body, err := interchain.Marshal()
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(body)
}

func (x *InterchainManager) DeleteInterchain(id string) *boltvm.Response {
	x.Delete(AppchainKey(id))
	return boltvm.Success(nil)
}

// GetInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
func (x *InterchainManager) getInterchain(id string) (*pb.Interchain, bool) {
	interchain := &pb.Interchain{}
	ok, data := x.Get(AppchainKey(id))
	if !ok {
		return nil, false
	}

	if err := interchain.Unmarshal(data); err != nil {
		panic(err)
	}

	if interchain.InterchainCounter == nil {
		interchain.InterchainCounter = make(map[string]uint64)
	}

	if interchain.ReceiptCounter == nil {
		interchain.ReceiptCounter = make(map[string]uint64)
	}

	if interchain.SourceReceiptCounter == nil {
		interchain.SourceReceiptCounter = make(map[string]uint64)
	}

	return interchain, true
}

// GetInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
func (x *InterchainManager) setInterchain(id string, interchain *pb.Interchain) {
	data, err := interchain.Marshal()
	if err != nil {
		panic(err)
	}

	x.Set(AppchainKey(id), data)
}

// Interchain returns information of the interchain count, Receipt count and SourceReceipt count
func (x *InterchainManager) Interchain() *boltvm.Response {
	ok, data := x.Get(AppchainKey(x.Caller()))
	if !ok {
		return boltvm.Error(fmt.Errorf("this appchain does not exist").Error())
	}
	return boltvm.Success(data)
}

// GetInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
func (x *InterchainManager) GetInterchain(id string) *boltvm.Response {
	ok, data := x.Get(AppchainKey(id))
	if !ok {
		return boltvm.Error(fmt.Errorf("this appchain does not exist").Error())
	}
	return boltvm.Success(data)
}

func (x *InterchainManager) HandleIBTP(ibtp *pb.IBTP) *boltvm.Response {

	if len(strings.Split(ibtp.From, "-")) == 2 {
		return x.handleUnionIBTP(ibtp)
	}

	interchain, _, err := x.checkAppchain(ibtp.From)
	if err != nil {
		return boltvm.Error(err.Error())
	}

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
	interchain, _, err := x.checkAppchain(x.Caller())
	if err != nil {
		return boltvm.Error(err.Error())
	}

	ibtps := &pb.IBTPs{}
	if err := ibtps.Unmarshal(data); err != nil {
		return boltvm.Error(err.Error())
	}

	for _, ibtp := range ibtps.Ibtps {
		if err := x.checkIBTP(ibtp, interchain); err != nil {
			return boltvm.Error(err.Error())
		}
	}

	if res := x.beginMultiTargetsTransaction(ibtps); !res.Ok {
		return res
	}

	for _, ibtp := range ibtps.Ibtps {
		x.ProcessIBTP(ibtp, interchain)
	}

	return boltvm.Success(nil)
}

func (x *InterchainManager) checkIBTP(ibtp *pb.IBTP, interchain *pb.Interchain) error {
	isRelayIBTP := x.isRelayIBTP(ibtp.From)

	if ibtp.To == "" {
		return fmt.Errorf("empty destination chain id")
	}

	if _, ok := x.getInterchain(ibtp.To); !ok {
		x.Logger().WithField("chain_id", ibtp.To).Debug("target appchain does not exist")
	}

	if pb.IBTP_INTERCHAIN == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_INIT == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REDEEM == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REFUND == ibtp.Type {
		if !isRelayIBTP {
			if ibtp.From != x.Caller() {
				return fmt.Errorf("ibtp from != caller")
			}
		}

		idx := interchain.InterchainCounter[ibtp.To]
		if ibtp.Index <= idx {
			return fmt.Errorf(fmt.Sprintf("index already exists, required %d, but %d", idx+1, ibtp.Index))
		}
		if ibtp.Index > idx+1 {
			return fmt.Errorf(fmt.Sprintf("wrong index, required %d, but %d", idx+1, ibtp.Index))
		}
	} else {
		if !isRelayIBTP {
			if ibtp.To != x.Caller() {
				return fmt.Errorf("ibtp to != caller")
			}
		}

		idx := interchain.ReceiptCounter[ibtp.To]
		if ibtp.Index <= idx {
			return fmt.Errorf(fmt.Sprintf("receipt index already exists, required %d, but %d", idx+1, ibtp.Index))
		}

		if ibtp.Index > idx+1 {
			return fmt.Errorf(fmt.Sprintf("wrong receipt index, required %d, but %d", idx+1, ibtp.Index))
		}
	}

	return nil
}

func (x *InterchainManager) checkAppchain(id string) (*pb.Interchain, *appchainMgr.Appchain, error) {
	interchain, ok := x.getInterchain(id)
	if !ok {
		return nil, nil, fmt.Errorf("this appchain does not exist")
	}

	app := &appchainMgr.Appchain{}
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(id))
	if !res.Ok {
		return nil, nil, fmt.Errorf("get appchain info error: " + string(res.Result))
	}

	if err := json.Unmarshal(res.Result, app); err != nil {
		return nil, nil, fmt.Errorf("unmarshal error: " + err.Error())
	}

	if app.Status != appchainMgr.AppchainAvailable {
		return nil, nil, fmt.Errorf("the appchain status is " + string(app.Status) + ", can not handle IBTP")
	}

	return interchain, app, nil
}

// isRelayIBTP returns whether ibtp.from is relaychain type
func (x *InterchainManager) isRelayIBTP(from string) bool {
	srcChain := &appchainMgr.Appchain{}
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(from))
	if err := json.Unmarshal(res.Result, srcChain); err != nil {
		return false
	}
	return srcChain.ChainType == appchainMgr.RelaychainType
}

func (x *InterchainManager) ProcessIBTP(ibtp *pb.IBTP, interchain *pb.Interchain) {
	m := make(map[string]uint64)
	if pb.IBTP_INTERCHAIN == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_INIT == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REDEEM == ibtp.Type ||
		pb.IBTP_ASSET_EXCHANGE_REFUND == ibtp.Type {
		if interchain.InterchainCounter == nil {
			x.Logger().Info("interchain counter is nil, make one")
			interchain.InterchainCounter = make(map[string]uint64)
		}
		interchain.InterchainCounter[ibtp.To]++
		x.setInterchain(ibtp.From, interchain)
		x.AddObject(x.indexMapKey(ibtp.ID()), x.GetTxHash())
		m[ibtp.To] = x.GetTxIndex()
	} else {
		interchain.ReceiptCounter[ibtp.To] = ibtp.Index
		x.setInterchain(ibtp.From, interchain)
		m[ibtp.From] = x.GetTxIndex()

		ic, _ := x.getInterchain(ibtp.To)
		ic.SourceReceiptCounter[ibtp.From] = ibtp.Index
		x.setInterchain(ibtp.To, ic)
		x.SetObject(x.indexReceiptMapKey(ibtp.ID()), x.GetTxHash())

	}

	x.PostInterchainEvent(m)
}

func (x *InterchainManager) beginMultiTargetsTransaction(ibtps *pb.IBTPs) *boltvm.Response {
	args := make([]*pb.Arg, 0)
	globalId := fmt.Sprintf("%s-%s", x.Caller(), x.GetTxHash())
	args = append(args, pb.String(globalId))

	for _, ibtp := range ibtps.Ibtps {
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

	_, app, err := x.checkAppchain(srcRelayChainID)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	if ibtp.To == "" {
		return boltvm.Error("empty destination chain id")
	}
	if ok := x.Has(AppchainKey(ibtp.To)); !ok {
		return boltvm.Error(fmt.Sprintf("target appchain does not exist: %s", ibtp.To))
	}

	interchain, ok := x.getInterchain(ibtp.From)
	if !ok {
		interchain = &pb.Interchain{
			ID: ibtp.From,
		}
		x.setInterchain(ibtp.From, interchain)
	}

	if err := x.checkUnionIBTP(app, ibtp, interchain); err != nil {
		return boltvm.Error(err.Error())
	}

	x.ProcessIBTP(ibtp, interchain)
	return boltvm.Success(nil)
}

func (x *InterchainManager) checkUnionIBTP(app *appchainMgr.Appchain, ibtp *pb.IBTP, interchain *pb.Interchain) error {
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

	_, err := verifyMultiSign(app, ibtp, ibtp.Proof)
	return err
}

// verifyMultiSign .
func verifyMultiSign(app *appchainMgr.Appchain, ibtp *pb.IBTP, proof []byte) (bool, error) {
	if "" == app.Validators {
		return false, fmt.Errorf("empty validators in relay chain:%s", app.ID)
	}
	var validators BxhValidators
	if err := json.Unmarshal([]byte(app.Validators), &validators); err != nil {
		return false, err
	}

	m := make(map[string]struct{}, 0)
	for _, validator := range validators.Addresses {
		m[validator] = struct{}{}
	}

	var signs pb.SignResponse
	if err := signs.Unmarshal(ibtp.Proof); err != nil {
		return false, err
	}

	threshold := (len(validators.Addresses) - 1) / 3 // TODO be dynamic
	counter := 0

	ibtpHash := ibtp.Hash()
	hash := sha256.Sum256([]byte(ibtpHash.String()))
	for v, sign := range signs.Sign {
		if _, ok := m[v]; !ok {
			return false, fmt.Errorf("wrong validator: %s", v)
		}
		delete(m, v)
		addr := types.NewAddressByStr(v)
		ok, _ := asym.Verify(crypto.Secp256k1, sign, hash[:], *addr)
		if ok {
			counter++
		}
		if counter > threshold {
			return true, nil
		}
	}
	return false, fmt.Errorf("multi signs verify fail, counter: %d", counter)
}

func AppchainKey(id string) string {
	return appchainMgr.PREFIX + id
}

func (x *InterchainManager) indexMapKey(id string) string {
	return fmt.Sprintf("index-tx-%s", id)
}

func (x *InterchainManager) indexReceiptMapKey(id string) string {
	return fmt.Sprintf("index-receipt-tx-%s", id)
}
