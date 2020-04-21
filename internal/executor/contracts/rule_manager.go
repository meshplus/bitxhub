package contracts

import (
	"fmt"

	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
)

const (
	rulePrefix = "rule-"
)

// RuleManager is the contract manage validation rules
type RuleManager struct {
	boltvm.Stub
}

type rule struct {
	Address string `json:"address"`
	Status  int32  `json:"status"` // 0 => registered, 1 => approved, -1 => rejected
}

type ruleRecord struct {
	Rule       *rule  `json:"rule"`
	IsApproved bool   `json:"is_approved"`
	Desc       string `json:"desc"`
}

// SetRule can map the validation rule address with the chain id
func (r *RuleManager) RegisterRule(id string, address string) *boltvm.Response {
	rl := &rule{
		Address: address,
		Status:  0,
	}
	r.SetObject(r.ruleKey(id), rl)
	return boltvm.Success(nil)
}

func (r *RuleManager) GetRuleAddress(id, chainType string) *boltvm.Response {
	rl := &rule{}

	ok := r.GetObject(r.ruleKey(id), rl)
	if ok {
		return boltvm.Success([]byte(rl.Address))
	}

	if chainType == "fabric" {
		return boltvm.Success([]byte(validator.FabricRuleAddr))
	}

	return boltvm.Error("")
}

func (r *RuleManager) Audit(id string, isApproved int32, desc string) *boltvm.Response {
	rl := &rule{}
	ok := r.GetObject(r.ruleKey(id), rl)
	if !ok {
		return boltvm.Error(fmt.Errorf("this rule does not exist").Error())
	}
	rl.Status = isApproved

	record := &ruleRecord{
		Rule:       rl,
		IsApproved: isApproved == approved,
		Desc:       desc,
	}

	var records []*ruleRecord
	r.GetObject(r.ruleRecordKey(id), &records)
	records = append(records, record)

	r.SetObject(r.ruleRecordKey(id), records)
	r.SetObject(r.ruleKey(id), rl)

	return boltvm.Success(nil)
}

func (r *RuleManager) ruleKey(id string) string {
	return rulePrefix + id
}

func (r *RuleManager) ruleRecordKey(id string) string {
	return "audit-record-" + id
}
