package validator

import (
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/sirupsen/logrus"
)

const (
	FabricRuleAddr = "0x00000000000000000000000000000000000000a0"
)

// Validator is the instance that can use wasm to verify transaction validity
type ValidationEngine struct {
	ledger ledger.Ledger
	logger logrus.FieldLogger
}

// New a validator instance
func NewValidationEngine(ledger ledger.Ledger, logger logrus.FieldLogger) *ValidationEngine {
	return &ValidationEngine{
		ledger: ledger,
		logger: logger,
	}
}

// Verify will check whether the transaction info is valid
func (ve *ValidationEngine) Validate(address, from string, proof []byte, validators string) (bool, error) {
	vlt := ve.getValidator(address)

	return vlt.Verify(address, from, proof, validators)
}

func (ve *ValidationEngine) getValidator(address string) Validator {
	if address == FabricRuleAddr {
		return NewFabV14Validator(ve.ledger, ve.logger)
	}

	return NewWasmValidator(ve.ledger, ve.logger)
}
