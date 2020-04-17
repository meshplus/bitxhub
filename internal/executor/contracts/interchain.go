package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/meshplus/bitxhub/internal/constant"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
	"github.com/sirupsen/logrus"
)

const (
	prefix = "appchain-"

	registered = 0
	approved   = 1
)

type Interchain struct {
	boltvm.Stub
}

type appchain struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Validators    string `json:"validators"`
	ConsensusType int32  `json:"consensus_type"`
	// 0 => registered, 1 => approved, -1 => rejected
	Status               int32             `json:"status"`
	ChainType            string            `json:"chain_type"`
	Desc                 string            `json:"desc"`
	Version              string            `json:"version"`
	InterchainCounter    map[string]uint64 `json:"interchain_counter,omitempty"`
	ReceiptCounter       map[string]uint64 `json:"receipt_counter,omitempty"`
	SourceReceiptCounter map[string]uint64 `json:"source_receipt_counter,omitempty"`
}

type auditRecord struct {
	Appchain   *appchain `json:"appchain"`
	IsApproved bool      `json:"is_approved"`
	Desc       string    `json:"desc"`
}

func (chain *appchain) UnmarshalJSON(data []byte) error {
	type alias appchain
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

	*chain = appchain(*t)
	return nil
}

// Register appchain manager registers appchain info caller is the appchain
// manager address return appchain id and error
func (x *Interchain) Register(validators string, consensusType int32, chainType, name, desc, version string) *boltvm.Response {
	chain := &appchain{
		ID:            x.Caller(),
		Name:          name,
		Validators:    validators,
		ConsensusType: consensusType,
		ChainType:     chainType,
		Desc:          desc,
		Version:       version,
	}

	ok := x.Has(x.appchainKey(x.Caller()))
	if ok {
		x.Stub.Logger().WithFields(logrus.Fields{
			"id": x.Caller(),
		}).Debug("Appchain has registered")
		x.GetObject(x.appchainKey(x.Caller()), chain)
	} else {
		// logger.Info(x.Caller())
		x.SetObject(x.appchainKey(x.Caller()), chain)
		x.Logger().WithFields(logrus.Fields{
			"id": x.Caller(),
		}).Info("Appchain register successfully")
	}
	body, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(body)
}

func (x *Interchain) UpdateAppchain(validators string, consensusType int32, chainType, name, desc, version string) *boltvm.Response {
	ok := x.Has(x.appchainKey(x.Caller()))
	if !ok {
		return boltvm.Error("register appchain firstly")
	}

	chain := &appchain{}
	x.GetObject(x.appchainKey(x.Caller()), chain)

	if chain.Status == registered {
		return boltvm.Error("this appchain is being audited")
	}

	chain = &appchain{
		ID:            x.Caller(),
		Name:          name,
		Validators:    validators,
		ConsensusType: consensusType,
		ChainType:     chainType,
		Desc:          desc,
		Version:       version,
	}

	x.SetObject(x.appchainKey(x.Caller()), chain)

	return boltvm.Success(nil)
}

// Audit bitxhub manager audit appchain register info
// caller is the bitxhub manager address
// proposer is the appchain manager address
func (x *Interchain) Audit(proposer string, isApproved int32, desc string) *boltvm.Response {
	ret := x.CrossInvoke(constant.RoleContractAddr.String(), "IsAdmin", pb.String(x.Caller()))
	is, err := strconv.ParseBool(string(ret.Result))
	if err != nil {
		return boltvm.Error(fmt.Errorf("judge caller type: %w", err).Error())
	}

	if !is {
		return boltvm.Error("caller is not an admin account")
	}

	chain := &appchain{}
	ok := x.GetObject(x.appchainKey(proposer), chain)
	if !ok {
		return boltvm.Error(fmt.Errorf("this appchain does not exist").Error())
	}

	chain.Status = isApproved

	record := &auditRecord{
		Appchain:   chain,
		IsApproved: isApproved == approved,
		Desc:       desc,
	}

	var records []*auditRecord
	x.GetObject(x.auditRecordKey(proposer), &records)
	records = append(records, record)

	x.SetObject(x.auditRecordKey(proposer), records)
	x.SetObject(x.appchainKey(proposer), chain)

	return boltvm.Success([]byte(fmt.Sprintf("audit %s successfully", proposer)))
}

func (x *Interchain) FetchAuditRecords(id string) *boltvm.Response {
	var records []*auditRecord
	x.GetObject(x.auditRecordKey(id), &records)

	body, err := json.Marshal(records)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(body)
}

// CountApprovedAppchains counts all approved appchains
func (x *Interchain) CountApprovedAppchains() *boltvm.Response {
	ok, value := x.Query(prefix)
	if !ok {
		return boltvm.Success([]byte("0"))
	}

	count := 0
	for _, v := range value {
		a := &appchain{}
		if err := json.Unmarshal(v, a); err != nil {
			return boltvm.Error(fmt.Sprintf("unmarshal json error: %v", err))
		}
		if a.Status == approved {
			count++
		}
	}

	return boltvm.Success([]byte(strconv.Itoa(count)))
}

// CountAppchains counts all appchains including approved, rejected or registered
func (x *Interchain) CountAppchains() *boltvm.Response {
	ok, value := x.Query(prefix)
	if !ok {
		return boltvm.Success([]byte("0"))
	}

	return boltvm.Success([]byte(strconv.Itoa(len(value))))
}

// Appchains returns all appchains
func (x *Interchain) Appchains() *boltvm.Response {
	ok, value := x.Query(prefix)
	if !ok {
		return boltvm.Success(nil)
	}

	ret := make([]*appchain, 0)
	for _, data := range value {
		chain := &appchain{}
		if err := json.Unmarshal(data, chain); err != nil {
			return boltvm.Error(err.Error())
		}
		ret = append(ret, chain)
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(data)
}

func (x *Interchain) DeleteAppchain(cid string) *boltvm.Response {
	ret := x.CrossInvoke(constant.RoleContractAddr.String(), "IsAdmin", pb.String(x.Caller()))
	is, err := strconv.ParseBool(string(ret.Result))
	if err != nil {
		return boltvm.Error(fmt.Errorf("judge caller type: %w", err).Error())
	}

	if !is {
		return boltvm.Error("caller is not an admin account")
	}

	x.Delete(prefix + cid)
	x.Logger().Infof("delete appchain:%s", cid)

	return boltvm.Success(nil)
}

func (x *Interchain) Appchain() *boltvm.Response {
	ok, data := x.Get(x.appchainKey(x.Caller()))
	if !ok {
		return boltvm.Error(fmt.Errorf("this appchain does not exist").Error())
	}

	return boltvm.Success(data)
}

func (x *Interchain) HandleIBTP(data []byte) *boltvm.Response {
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

	app := &appchain{}
	x.GetObject(x.appchainKey(ibtp.From), &app)

	// get validation rule contract address
	res := x.CrossInvoke(constant.RuleManagerContractAddr.String(), "GetRuleAddress", pb.String(ibtp.From), pb.String(app.ChainType))
	if !res.Ok {
		return boltvm.Error("this appchain don't register rule")
	}

	// handle validation
	isValid, err := x.ValidationEngine().Validate(string(res.Result), ibtp.From, ibtp.Proof, app.Validators)
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

		idx := app.InterchainCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			return boltvm.Error(fmt.Sprintf("wrong index, required %d, but %d", idx+1, ibtp.Index))
		}

		app.InterchainCounter[ibtp.To]++
		x.SetObject(x.appchainKey(ibtp.From), app)
		x.SetObject(x.indexMapKey(ibtp.ID()), x.GetTxHash())
		m := make(map[string]uint64)
		m[ibtp.To] = x.GetTxIndex()
		x.PostInterchainEvent(m)
	case pb.IBTP_RECEIPT:
		if ibtp.To != x.Caller() {
			return boltvm.Error("ibtp from != caller")
		}

		idx := app.ReceiptCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			if app.SourceReceiptCounter[ibtp.To]+1 != ibtp.Index {
				return boltvm.Error(fmt.Sprintf("wrong receipt index, required %d, but %d", idx+1, ibtp.Index))
			}
		}

		app.ReceiptCounter[ibtp.To] = ibtp.Index
		x.SetObject(x.appchainKey(ibtp.From), app)
		m := make(map[string]uint64)
		m[ibtp.From] = x.GetTxIndex()

		ac := &appchain{}
		x.GetObject(x.appchainKey(ibtp.To), &ac)
		ac.SourceReceiptCounter[ibtp.From] = ibtp.Index
		x.SetObject(x.appchainKey(ibtp.To), ac)

		x.PostInterchainEvent(m)
	}

	return boltvm.Success(nil)
}

func (x *Interchain) GetIBTPByID(id string) *boltvm.Response {
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

func (x *Interchain) appchainKey(id string) string {
	return prefix + id
}

func (x *Interchain) auditRecordKey(id string) string {
	return "audit-record-" + id
}

func (x *Interchain) indexMapKey(id string) string {
	return fmt.Sprintf("index-tx-%s", id)
}
