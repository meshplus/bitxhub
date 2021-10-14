package proof

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/sirupsen/logrus"
)

const (
	ProofError           = "proof verify failed"
	AppchainNotAvailable = "appchain not available"
	NoBindRule           = "appchain didn't register rule"
	internalError        = "internal server error"
)

type VerifyPool struct {
	proofs    sync.Map //ibtp proof cache
	ledger    *ledger.Ledger
	ve        validator.Engine
	logger    logrus.FieldLogger
	bitxhubID string
}

var _ Verify = (*VerifyPool)(nil)

func New(ledger *ledger.Ledger, logger logrus.FieldLogger, bxhID, wasmGasLimit uint64) Verify {
	ve := validator.NewValidationEngine(ledger, &sync.Map{}, log.NewWithModule("validator"), wasmGasLimit)

	proofPool := &VerifyPool{
		ledger:    ledger,
		logger:    logger,
		ve:        ve,
		bitxhubID: fmt.Sprintf("%d", bxhID),
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
		return false, 0, fmt.Errorf("%s: %w", ProofError, err)
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
			return false, 0, fmt.Errorf("%s: wrong validator: %s", ProofError, v)
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
	return false, 0, fmt.Errorf("%s: multi signs verify fail, counter: %d", ProofError, counter)
}

func (pl *VerifyPool) verifyProof(ibtp *pb.IBTP, proof []byte) (bool, uint64, error) {
	if proof == nil {
		return false, 0, fmt.Errorf("%s:, empty proof", ProofError)
	}
	proofHash := sha256.Sum256(proof)
	if !bytes.Equal(proofHash[:], ibtp.Proof) {
		return false, 0, fmt.Errorf("%s: proof hash is not correct", ProofError)
	}

	// get real appchain id for union ibtp
	if err := ibtp.CheckServiceID(); err != nil {
		return false, 0, err
	}

	var (
		bxhID   string
		chainID string
	)

	if ibtp.Category() == pb.IBTP_REQUEST {
		bxhID, chainID, _ = ibtp.ParseFrom()
	} else {
		bxhID, chainID, _ = ibtp.ParseTo()
	}

	if bxhID != pl.bitxhubID {
		app, err := pl.getAppchain(bxhID)
		if err != nil {
			return false, 0, err
		}
		return verifyMultiSign(app, ibtp, proof)
	}

	app, err := pl.getAppchain(chainID)
	if err != nil {
		return false, 0, err
	}

	validateAddr, err := pl.getValidateAddress(chainID)
	if err != nil {
		return false, 0, err
	}

	ok, gasUsed, err := pl.ve.Validate(validateAddr, chainID, proof, ibtp.Payload, string(app.TrustRoot))
	if err != nil {
		return false, gasUsed, fmt.Errorf("%s: %w", ProofError, err)
	}
	return ok, gasUsed, nil
}

func (pl *VerifyPool) getValidateAddress(chainID string) (string, error) {
	getRuleFunc := func(chainID string) (*ruleMgr.Rule, error) {
		ok, data := pl.ledger.Copy().GetState(constant.RuleManagerContractAddr.Address(), []byte(ruleMgr.RuleKey(chainID)))
		if !ok {
			return nil, nil
		}

		rules := make([]*ruleMgr.Rule, 0)
		if err := json.Unmarshal(data, &rules); err != nil {
			return nil, fmt.Errorf("unmarshal rules error: %w", err)
		}

		for _, r := range rules {
			if governance.GovernanceAvailable == r.Status {
				return r, nil
			}
		}

		return nil, nil
	}

	rl, err := getRuleFunc(chainID)
	if err != nil {
		return "", err
	}

	if rl != nil {
		return rl.Address, nil
	}

	return "", fmt.Errorf("%s for chainID %s", NoBindRule, chainID)
}

func (pl *VerifyPool) getAccountState(address constant.BoltContractAddress, key string) (bool, []byte) {
	return pl.ledger.Copy().GetState(address.Address(), []byte(key))
}

func (pl *VerifyPool) getAppchain(chainID string) (*appchainMgr.Appchain, error) {
	app := &appchainMgr.Appchain{}
	ok, data := pl.getAccountState(constant.AppchainMgrContractAddr, appchainMgr.AppchainKey(chainID))
	if !ok {
		return nil, fmt.Errorf("%s: cannot get registered appchain", AppchainNotAvailable)
	}

	err := json.Unmarshal(data, app)
	if err != nil {
		return nil, fmt.Errorf("%s: unmarshal appchain data fail: %w", internalError, err)
	}

	return app, nil
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
