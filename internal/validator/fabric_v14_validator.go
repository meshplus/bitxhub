package validator

import (
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/validator/validatorlib"
	"github.com/sirupsen/logrus"
)

// Validator is the instance that can use wasm to verify transaction validity
type FabV14Validator struct {
	ledger ledger.Ledger
	logger logrus.FieldLogger
}

// New a validator instance
func NewFabV14Validator(ledger ledger.Ledger, logger logrus.FieldLogger) *FabV14Validator {
	return &FabV14Validator{
		ledger: ledger,
		logger: logger,
	}
}

// Verify will check whether the transaction info is valid
func (vlt *FabV14Validator) Verify(address, from string, proof []byte, validators string) (bool, error) {
	vInfo, err := validatorlib.UnmarshalValidatorInfo([]byte(validators))
	if err != nil {
		return false, err
	}
	err = validatorlib.ValidateV14(proof, []byte(vInfo.Policy), vInfo.ConfByte, vInfo.Cid)
	if err != nil {
		return false, err
	}

	return true, nil
}
