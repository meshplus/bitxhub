package proof

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/sirupsen/logrus"
)

const (
	InvalidIBTP          = "invalid ibtp"
	AppchainNotAvailable = "appchain not available"
	NoBindRule           = "appchain didn't register rule"
	internalError        = "internal server error"
)

type VerifyPool struct {
	proofs sync.Map //ibtp proof cache
	ledger *ledger.Ledger
	ve     validator.Engine
	logger logrus.FieldLogger
}

var _ Verify = (*VerifyPool)(nil)

func New(ledger *ledger.Ledger, logger logrus.FieldLogger, wasmGasLimit uint64) Verify {
	ve := validator.NewValidationEngine(ledger, &sync.Map{}, log.NewWithModule("validator"), wasmGasLimit)
	proofPool := &VerifyPool{
		ledger: ledger,
		logger: logger,
		ve:     ve,
	}
	return proofPool
}

func (pl *VerifyPool) ValidationEngine() validator.Engine {
	return pl.ve
}

func (pl *VerifyPool) CheckProof(tx pb.Transaction) (ok bool, gasUsed uint64, err error) {
	ibtp := tx.GetIBTP()
	if ibtp != nil {
		ok, gasUsed, err = pl.verifyProof(ibtp, tx.GetExtra())
		if err != nil {
			pl.logger.WithFields(logrus.Fields{
				"hash":  tx.GetHash().String(),
				"id":    ibtp.ID(),
				"error": err}).Warn("ibtp verify got error")
			return false, gasUsed, err
		}
		if !ok {
			pl.logger.WithFields(logrus.Fields{"hash": tx.GetHash().String(), "id": ibtp.ID()}).Warn("ibtp verify failed")
			return false, gasUsed, nil
		}

		//TODO(jz): need to remove the proof
		//tx.Extra = nil
	}
	return true, gasUsed, nil
}

type bxhValidators struct {
	Addresses []string `json:"addresses"`
}

// verifyMultiSign .
func verifyMultiSign(app *appchainMgr.Appchain, ibtp *pb.IBTP, proof []byte) (bool, uint64, error) {
	if app.TrustRoot == nil {
		return false, 0, fmt.Errorf("%s: empty validators in relay chain:%s", internalError, app.ID)
	}
	var validators bxhValidators
	if err := json.Unmarshal(app.TrustRoot, &validators); err != nil {
		return false, 0, fmt.Errorf("%s: %w", InvalidIBTP, err)
	}

	m := make(map[string]struct{}, 0)
	for _, validator := range validators.Addresses {
		m[validator] = struct{}{}
	}

	var signs pb.SignResponse
	if err := signs.Unmarshal(proof); err != nil {
		return false, 0, err
	}

	threshold := (len(validators.Addresses) - 1) / 3 // TODO be dynamic
	counter := 0

	ibtpHash := ibtp.Hash()
	hash := sha256.Sum256([]byte(ibtpHash.String()))
	for v, sign := range signs.Sign {
		if _, ok := m[v]; !ok {
			return false, 0, fmt.Errorf("%s: wrong validator: %s", InvalidIBTP, v)
		}
		delete(m, v)
		addr := types.NewAddressByStr(v)
		ok, _ := asym.VerifyWithType(sign, hash[:], *addr)
		if ok {
			counter++
		}
		if counter > threshold {
			return true, 0, nil
		}
	}
	return false, 0, fmt.Errorf("%s: multi signs verify fail, counter: %d", InvalidIBTP, counter)
}

func (pl *VerifyPool) verifyProof(ibtp *pb.IBTP, proof []byte) (bool, uint64, error) {
	if proof == nil {
		return false, 0, fmt.Errorf("%s:, empty proof", InvalidIBTP)
	}
	proofHash := sha256.Sum256(proof)
	if !bytes.Equal(proofHash[:], ibtp.Proof) {
		return false, 0, fmt.Errorf("%s: proof hash is not correct", InvalidIBTP)
	}

	// get real appchain id for union ibtp
	if err := ibtp.CheckServiceID(); err != nil {
		return false, 0, err
	}

	from := ibtp.SrcChainID()
	app := &appchainMgr.Appchain{}
	ok, data := pl.getAccountState(constant.AppchainMgrContractAddr, contracts.AppchainKey(from)) // ibtp.From
	if !ok {
		return false, 0, fmt.Errorf("%s: cannot get registered appchain", AppchainNotAvailable)
	}
	err := json.Unmarshal(data, app)
	if err != nil {
		return false, 0, fmt.Errorf("%s: unmarshal appchain data fail: %w", internalError, err)
	}

	if len(strings.Split(ibtp.From, "-")) == 2 {
		return verifyMultiSign(app, ibtp, proof)
	}

	validateAddr := validator.SimFabricRuleAddr
	rl := &ruleMgr.Rule{}
	ok, data = pl.getRule(from)
	if ok {
		if err := json.Unmarshal(data, rl); err != nil {
			return false, 0, fmt.Errorf("%s: unmarshal rule data error: %w", internalError, err)
		}
		validateAddr = rl.Address
	} else {
		return false, 0, fmt.Errorf(NoBindRule)
	}

	ok, gasUsed, err := pl.ve.Validate(validateAddr, from, proof, ibtp.Payload, string(app.TrustRoot)) // ibtp.From
	if err != nil {
		return false, gasUsed, fmt.Errorf("%s: %w", InvalidIBTP, err)
	}
	return true, gasUsed, nil
}

func (pl *VerifyPool) getRule(chainId string) (bool, []byte) {
	ok, data := pl.ledger.Copy().GetState(constant.RuleManagerContractAddr.Address(), []byte(ruleMgr.RuleKey(chainId)))
	if !ok {
		return ok, data
	}

	rules := make([]*ruleMgr.Rule, 0)
	if err := json.Unmarshal(data, &rules); err != nil {
		return false, []byte("unmarshal rules error: " + err.Error())
	}

	for _, r := range rules {
		if governance.GovernanceAvailable == r.Status {
			resData, err := json.Marshal(r)
			if err != nil {
				return false, []byte("marshal rule error: " + err.Error())
			}
			return true, resData
		}
	}

	return false, []byte(fmt.Errorf("the available rule does not exist").Error())
}

func (pl *VerifyPool) getAccountState(address constant.BoltContractAddress, key string) (bool, []byte) {
	return pl.ledger.Copy().GetState(address.Address(), []byte(key))
}

func (pl *VerifyPool) putProof(proofHash types.Hash, proof []byte) {
	pl.proofs.Store(proofHash, proof)
}

func (pl *VerifyPool) GetProof(txHash types.Hash) ([]byte, bool) {
	proof, ok := pl.proofs.Load(txHash)
	if !ok {
		return nil, ok
	}
	return proof.([]byte), ok
}

func (pl *VerifyPool) DeleteProof(txHash types.Hash) {
	pl.proofs.Delete(txHash)
}
