package proof

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/meshplus/bitxhub-kit/crypto"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
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

type VerifyPool struct {
	proofs sync.Map //ibtp proof cache
	ledger ledger.Ledger
	ve     validator.Engine
	logger logrus.FieldLogger
}

var _ Verify = (*VerifyPool)(nil)

func New(ledger ledger.Ledger, logger logrus.FieldLogger) Verify {
	ve := validator.NewValidationEngine(ledger, &sync.Map{}, log.NewWithModule("validator"))
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

func (pl *VerifyPool) CheckProof(tx *pb.Transaction) (bool, error) {
	ibtp := tx.IBTP
	if ibtp != nil {
		ok, err := pl.verifyProof(ibtp, tx.Extra)
		if err != nil {
			pl.logger.WithFields(logrus.Fields{
				"hash":  tx.TransactionHash.String(),
				"id":    ibtp.ID(),
				"error": err}).Warn("ibtp verify got error")
			return false, err
		}
		if !ok {
			pl.logger.WithFields(logrus.Fields{"hash": tx.TransactionHash.String(), "id": ibtp.ID()}).Warn("ibtp verify failed")
			return false, nil
		}

		//TODO(jz): need to remove the proof
		//tx.Extra = nil
	}
	return true, nil
}

type bxhValidators struct {
	Addresses []string `json:"addresses"`
}

// verifyMultiSign .
func verifyMultiSign(app *appchainMgr.Appchain, ibtp *pb.IBTP, proof []byte) (bool, error) {
	if "" == app.Validators {
		return false, fmt.Errorf("empty validators in relay chain:%s", app.ID)
	}
	var validators bxhValidators
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

func (pl *VerifyPool) verifyProof(ibtp *pb.IBTP, proof []byte) (bool, error) {
	if proof == nil {
		return false, fmt.Errorf("empty proof")
	}
	proofHash := sha256.Sum256(proof)
	if !bytes.Equal(proofHash[:], ibtp.Proof) {
		return false, fmt.Errorf("proof hash is not correct")
	}

	// get real appchain id for union ibtp
	from := ibtp.From
	if len(strings.Split(ibtp.From, "-")) == 2 {
		from = strings.Split(ibtp.From, "-")[1]
		return true, nil
	}

	app := &appchainMgr.Appchain{}
	ok, data := pl.getAccountState(constant.AppchainMgrContractAddr, contracts.AppchainKey(from)) // ibtp.From
	if !ok {
		return false, fmt.Errorf("cannot get registered appchain")
	}
	err := json.Unmarshal(data, app)
	if err != nil {
		return false, fmt.Errorf("unmarshal appchain data fail: %w", err)
	}

	if len(strings.Split(ibtp.From, "-")) == 2 {
		return verifyMultiSign(app, ibtp, proof)
	}

	validateAddr := validator.FabricRuleAddr
	rl := &contracts.Rule{}
	ok, data = pl.getAccountState(constant.RuleManagerContractAddr, contracts.RuleKey(from))
	if ok {
		if err := json.Unmarshal(data, rl); err != nil {
			return false, fmt.Errorf("unmarshal rule data error: %w", err)
		}
		validateAddr = rl.Address
	} else {
		if app.ChainType != appchainMgr.FabricType {
			return false, fmt.Errorf("appchain didn't register rule")
		}
	}

	ok, err = pl.ve.Validate(validateAddr, from, proof, ibtp.Payload, app.Validators) // ibtp.From
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (pl *VerifyPool) getAccountState(address constant.BoltContractAddress, key string) (bool, []byte) {
	return pl.ledger.GetState(address.Address(), []byte(key))
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
