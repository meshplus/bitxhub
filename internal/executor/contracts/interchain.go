package contracts

import (
	"encoding/json"
	"fmt"
	"strings"

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

func (x *InterchainManager) HandleIBTP(data []byte) *boltvm.Response {
	ok := x.Has(x.appchainKey(x.Caller()))
	if !ok {
		return boltvm.Error("this appchain does not exist")
	}

	ibtp := &pb.IBTP{}
	if err := ibtp.Unmarshal(data); err != nil {
		return boltvm.Error(err.Error())
	}

	interchain := &Interchain{}
	x.GetObject(x.appchainKey(ibtp.From), &interchain)

	if err := x.checkIBTP(ibtp, interchain); err != nil {
		return boltvm.Error(err.Error())
	}

	res := boltvm.Success(nil)

	if pb.IBTP_INTERCHAIN == ibtp.Type {
		res = x.beginTransaction(ibtp)
	} else {
		res = x.reportTransaction(ibtp)
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
		x.Logger().WithField("chain_id", ibtp.To).Warn("target appchain does not exist")
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

	if pb.IBTP_INTERCHAIN == ibtp.Type {
		if ibtp.From != x.Caller() {
			return fmt.Errorf("ibtp from != caller")
		}

		idx := interchain.InterchainCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			return fmt.Errorf(fmt.Sprintf("wrong index, required %d, but %d", idx+1, ibtp.Index))
		}
	} else {
		if ibtp.To != x.Caller() {
			return fmt.Errorf("ibtp from != caller")
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

	if pb.IBTP_INTERCHAIN == ibtp.Type {
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

	return x.CrossInvoke(constant.TransactionMgrContractAddr.String(), "Begin", args...)
}

func (x *InterchainManager) beginTransaction(ibtp *pb.IBTP) *boltvm.Response {
	txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
	return x.CrossInvoke(constant.TransactionMgrContractAddr.String(), "Begin", pb.String(txId))
}

func (x *InterchainManager) reportTransaction(ibtp *pb.IBTP) *boltvm.Response {
	txId := fmt.Sprintf("%s-%s-%d", ibtp.To, ibtp.From, ibtp.Index)
	result := int32(0)
	if ibtp.Type == pb.IBTP_RECEIPT_FAILURE {
		result = 1
	}
	return x.CrossInvoke(constant.TransactionMgrContractAddr.String(), "Report", pb.String(txId), pb.Int32(result))
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

func (x *InterchainManager) appchainKey(id string) string {
	return appchainMgr.PREFIX + id
}

func (x *InterchainManager) indexMapKey(id string) string {
	return fmt.Sprintf("index-tx-%s", id)
}
