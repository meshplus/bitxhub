package contracts

import (
	"fmt"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
)

const (
	rulePrefix = "rule-"
)

// RuleManager is the contract manage validation rules
type RuleManager struct {
	boltvm.Stub
}

type Rule struct {
	Address string `json:"address"`
	Status  int32  `json:"status"` // 0 => registered, 1 => approved, -1 => rejected
}

type ruleRecord struct {
	Rule       *Rule  `json:"rule"`
	IsApproved bool   `json:"is_approved"`
	Desc       string `json:"desc"`
}

// SetRule can map the validation rule address with the chain id
func (r *RuleManager) RegisterRule(id string, address string) *boltvm.Response {
	if res := r.CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(id)); !res.Ok {
		return boltvm.Error(string(res.Result))
	}

	rl := &Rule{
		Address: address,
		Status:  0,
	}
	r.SetObject(RuleKey(id), rl)
	return boltvm.Success(nil)
}

func (r *RuleManager) GetRuleAddress(id, chainType string) *boltvm.Response {
	rl := &Rule{}

	ok := r.GetObject(RuleKey(id), rl)
	if ok {
		return boltvm.Success([]byte(rl.Address))
	}

	if chainType == "fabric" {
		return boltvm.Success([]byte(validator.FabricRuleAddr))
	}

	return boltvm.Error("")
}

func (r *RuleManager) Audit(id string, isApproved int32, desc string) *boltvm.Response {
	rl := &Rule{}
	ok := r.GetObject(RuleKey(id), rl)
	if !ok {
		return boltvm.Error(fmt.Errorf("this rule does not exist").Error())
	}
	rl.Status = isApproved

	record := &ruleRecord{
		Rule:       rl,
		IsApproved: isApproved == appchainMgr.APPROVED,
		Desc:       desc,
	}

	var records []*ruleRecord
	r.GetObject(r.ruleRecordKey(id), &records)
	records = append(records, record)

	r.SetObject(r.ruleRecordKey(id), records)
	r.SetObject(RuleKey(id), rl)

	return boltvm.Success(nil)
}

func RuleKey(id string) string {
	return rulePrefix + id
}

func (r *RuleManager) ruleRecordKey(id string) string {
	return "audit-record-" + id
}
