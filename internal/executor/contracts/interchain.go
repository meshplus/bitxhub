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

	if ibtp.To == "" {
		return boltvm.Error("empty destination chain id")
	}
	ok = x.Has(x.appchainKey(ibtp.To))
	if !ok {
		x.Logger().WithField("chain_id", ibtp.To).Warn("target appchain does not exist")
	}

	interchain := &Interchain{}
	x.GetObject(x.appchainKey(ibtp.From), &interchain)

	app := &appchainMgr.Appchain{}
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(ibtp.From))
	err := json.Unmarshal(res.Result, app)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	// get validation rule contract address
	res = x.CrossInvoke(constant.RuleManagerContractAddr.String(), "GetRuleAddress", pb.String(ibtp.From), pb.String(app.ChainType))
	if !res.Ok {
		return boltvm.Error("this appchain don't register rule")
	}

	// handle validation
	isValid, err := x.ValidationEngine().Validate(string(res.Result), ibtp.From, ibtp.Proof, ibtp.Payload, app.Validators)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	if !isValid {
		return boltvm.Error("invalid interchain transaction")
	}

	switch ibtp.Type {
	case pb.IBTP_INTERCHAIN:
		if ibtp.From != x.Caller() {
			return boltvm.Error("ibtp from != caller")
		}

		idx := interchain.InterchainCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			return boltvm.Error(fmt.Sprintf("wrong index, required %d, but %d", idx+1, ibtp.Index))
		}

		interchain.InterchainCounter[ibtp.To]++
		x.SetObject(x.appchainKey(ibtp.From), interchain)
		x.SetObject(x.indexMapKey(ibtp.ID()), x.GetTxHash())
		m := make(map[string]uint64)
		m[ibtp.To] = x.GetTxIndex()
		x.PostInterchainEvent(m)
	case pb.IBTP_RECEIPT:
		if ibtp.To != x.Caller() {
			return boltvm.Error("ibtp from != caller")
		}

		idx := interchain.ReceiptCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			if interchain.SourceReceiptCounter[ibtp.To]+1 != ibtp.Index {
				return boltvm.Error(fmt.Sprintf("wrong receipt index, required %d, but %d", idx+1, ibtp.Index))
			}
		}

		interchain.ReceiptCounter[ibtp.To] = ibtp.Index
		x.SetObject(x.appchainKey(ibtp.From), interchain)
		m := make(map[string]uint64)
		m[ibtp.From] = x.GetTxIndex()

		ic := &Interchain{}
		x.GetObject(x.appchainKey(ibtp.To), &ic)
		ic.SourceReceiptCounter[ibtp.From] = ibtp.Index
		x.SetObject(x.appchainKey(ibtp.To), ic)

		x.PostInterchainEvent(m)
	}

	return boltvm.Success(nil)
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
