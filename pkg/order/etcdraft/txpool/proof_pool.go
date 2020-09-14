package txpool

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/meshplus/bitxhub-kit/types"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
)

type ProofPool struct {
	proofs sync.Map //ibtp proof cache
	ledger ledger.Ledger
	ve     *validator.ValidationEngine
	logger logrus.FieldLogger
}

func (pl *ProofPool) extractIBTP(tx *pb.Transaction) *pb.IBTP {
	if strings.ToLower(tx.To.String()) != constant.InterchainContractAddr.String() {
		return nil
	}
	if tx.Data.VmType != pb.TransactionData_BVM {
		return nil
	}
	ip := &pb.InvokePayload{}
	if err := ip.Unmarshal(tx.Data.Payload); err != nil {
		return nil
	}
	if ip.Method != "HandleIBTP" {
		return nil
	}
	if len(ip.Args) != 1 {
		return nil
	}

	ibtp := &pb.IBTP{}
	if err := ibtp.Unmarshal(ip.Args[0].Value); err != nil {
		pl.logger.Error(err)
		return nil
	}
	return ibtp
}

func (pl *ProofPool) verifyProof(txHash types.Hash, ibtp *pb.IBTP, proof []byte) (bool, error) {
	if proof == nil {
		return false, nil
	}
	proofHash := sha256.Sum256(proof)
	if !bytes.Equal(proofHash[:], ibtp.Proof) {
		return false, nil
	}

	app := &appchainMgr.Appchain{}
	ok, data := pl.getAccountState(constant.AppchainMgrContractAddr, contracts.AppchainKey(ibtp.From))
	if !ok {
		return false, nil
	}
	err := json.Unmarshal(data, app)
	if err != nil {
		return false, err
	}

	validateAddr := validator.FabricRuleAddr
	rl := &contracts.Rule{}
	ok, data = pl.getAccountState(constant.RuleManagerContractAddr, contracts.RuleKey(ibtp.From))
	if ok {
		if err := json.Unmarshal(data, rl); err != nil {
			return false, err
		}
		validateAddr = rl.Address
	} else {
		if app.ChainType != "fabric" {
			return false, nil
		}
	}

	ok, err = pl.ve.Validate(validateAddr, ibtp.From, proof, ibtp.Payload, app.Validators)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (pl *ProofPool) getAccountState(address constant.BoltContractAddress, key string) (bool, []byte) {
	return pl.ledger.GetAccount(address.Address()).GetState([]byte(key))
}

func (pl *ProofPool) putProofHash(txHash types.Hash, proofHash types.Hash) {
	pl.proofs.Store(txHash, proofHash)
}

func (pl *ProofPool) getProofHash(txHash types.Hash) types.Hash {
	proofHash, ok := pl.proofs.Load(txHash)
	if !ok {
		return types.Hash{}
	}
	return proofHash.(types.Hash)
}

func (pl *ProofPool) deleteProofHash(txHash types.Hash) {
	pl.proofs.Delete(txHash)
}
