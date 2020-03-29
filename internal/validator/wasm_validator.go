package validator

import (
	"strconv"

	"github.com/gogo/protobuf/proto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm"
	"github.com/sirupsen/logrus"
)

// Validator is the instance that can use wasm to verify transaction validity
type WasmValidator struct {
	wasm   *wasm.Wasm
	tx     *pb.Transaction
	ledger ledger.Ledger
	logger logrus.FieldLogger
}

// New a validator instance
func NewWasmValidator(ledger ledger.Ledger, logger logrus.FieldLogger) *WasmValidator {
	return &WasmValidator{
		ledger: ledger,
		logger: logger,
	}
}

// Verify will check whether the transaction info is valid
func (vlt *WasmValidator) Verify(address, from string, proof []byte, validators string) (bool, error) {
	err := vlt.initRule(address, from, proof, validators)
	if err != nil {
		return false, err
	}

	ret, err := vlt.wasm.Run(vlt.tx.Data.Payload)
	if err != nil {
		return false, err
	}

	result, err := strconv.Atoi(string(ret))
	if err != nil {
		return false, err
	}

	if result == 0 {
		return false, nil
	}

	return true, nil
}

// InitRule can import a specific rule for validator to verify the transaction
func (vlt *WasmValidator) initRule(address, from string, proof []byte, validators string) error {
	err := vlt.setTransaction(address, from, proof, validators)
	if err != nil {
		return err
	}

	wasmCtx := vm.NewContext(vlt.tx, 0, vlt.tx.Data, vlt.ledger, vlt.logger)
	wasm, err := wasm.New(wasmCtx)
	if err != nil {
		return err
	}
	vlt.wasm = wasm

	return nil
}

func (vlt *WasmValidator) setTransaction(address, from string, proof []byte, validators string) error {
	payload := &pb.InvokePayload{
		Method: "start_verify",
		Args: []*pb.Arg{
			{Type: pb.Arg_Bytes, Value: proof},
			{Type: pb.Arg_Bytes, Value: []byte(validators)},
		},
	}
	input, _ := proto.Marshal(payload)

	txData := &pb.TransactionData{
		Type:    pb.TransactionData_INVOKE,
		Payload: input,
	}

	vlt.tx = &pb.Transaction{
		From: types.String2Address(from),
		To:   types.String2Address(address),
		Data: txData,
	}
	return nil
}
