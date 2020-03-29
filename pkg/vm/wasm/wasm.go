package wasm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/gogo/protobuf/proto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm/wasmlib"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

var (
	errorLackOfMethod = fmt.Errorf("wasm execute: lack of method name")
)

var _ vm.VM = (*Wasm)(nil)

var instances = make(map[string]wasmer.Instance)

func getInstance(code []byte) (wasmer.Instance, error) {
	ret := sha256.Sum256(code)
	v, ok := instances[string(ret[:])]
	if ok {
		return v, nil
	}

	imports, err := wasmlib.New()
	if err != nil {
		return wasmer.Instance{}, err
	}

	instance, err := wasmer.NewInstanceWithImports(code, imports)
	if err != nil {
		return wasmer.Instance{}, err
	}

	instances[string(ret[:])] = instance

	return instance, nil
}

// Wasm represents the wasm vm in BitXHub
type Wasm struct {
	// contract context
	ctx *vm.Context

	// wasm instance
	Instance wasmer.Instance

	argMap map[int]int
}

// Contract represents the smart contract structure used in the wasm vm
type Contract struct {
	// contract byte
	Code []byte

	// contract hash
	Hash types.Hash
}

// New creates a wasm vm instance
func New(ctx *vm.Context) (*Wasm, error) {
	wasm := &Wasm{
		ctx: ctx,
	}

	if ctx.Callee == (types.Address{}) {
		return wasm, nil
	}

	contractByte := ctx.Ledger.GetCode(ctx.Callee)

	if contractByte == nil {
		return nil, fmt.Errorf("this contract address does not exist")
	}

	contract := &Contract{}
	if err := json.Unmarshal(contractByte, contract); err != nil {
		return wasm, fmt.Errorf("contract byte not correct")
	}

	if len(contract.Code) == 0 {
		return wasm, fmt.Errorf("contract byte is empty")
	}

	instance, err := getInstance(contract.Code)
	if err != nil {
		return nil, err
	}

	wasm.Instance = instance
	wasm.argMap = make(map[int]int)

	return wasm, nil
}

// Run let the wasm vm excute or deploy the smart contract which depends on whether the callee is empty
func (w *Wasm) Run(input []byte) (ret []byte, err error) {
	if w.ctx.Callee == (types.Address{}) {
		return w.deploy()
	}

	return w.execute(input)
}

func (w *Wasm) deploy() ([]byte, error) {
	if len(w.ctx.TransactionData.Payload) == 0 {
		return nil, fmt.Errorf("contract cannot be empty")
	}
	contractNonce := w.ctx.Ledger.GetNonce(w.ctx.Caller)

	contractAddr := createAddress(w.ctx.Caller, contractNonce)
	wasmStruct := &Contract{
		Code: w.ctx.TransactionData.Payload,
		Hash: types.Bytes2Hash(w.ctx.TransactionData.Payload),
	}
	wasmByte, err := json.Marshal(wasmStruct)
	if err != nil {
		return nil, err
	}
	w.ctx.Ledger.SetCode(contractAddr, wasmByte)

	w.ctx.Ledger.SetNonce(w.ctx.Caller, contractNonce+1)

	return contractAddr.Bytes(), nil
}

func (w *Wasm) execute(input []byte) ([]byte, error) {
	payload := &pb.InvokePayload{}
	if err := proto.Unmarshal(input, payload); err != nil {
		return nil, err
	}

	if payload.Method == "" {
		return nil, errorLackOfMethod
	}

	methodName, ok := w.Instance.Exports[payload.Method]
	if !ok {
		return nil, fmt.Errorf("wrong rule contract")
	}
	slice := make([]interface{}, len(payload.Args))
	for i := range slice {
		arg := payload.Args[i]
		switch arg.Type {
		case pb.Arg_I32:
			temp, err := strconv.Atoi(string(arg.Value))
			if err != nil {
				return nil, err
			}
			slice[i] = temp
		case pb.Arg_I64:
			temp, err := strconv.ParseInt(string(arg.Value), 10, 64)
			if err != nil {
				return nil, err
			}
			slice[i] = temp
		case pb.Arg_F32:
			temp, err := strconv.ParseFloat(string(arg.Value), 32)
			if err != nil {
				return nil, err
			}
			slice[i] = temp
		case pb.Arg_F64:
			temp, err := strconv.ParseFloat(string(arg.Value), 64)
			if err != nil {
				return nil, err
			}
			slice[i] = temp
		case pb.Arg_String:
			inputPointer, err := w.SetString(string(arg.Value))
			if err != nil {
				return nil, err
			}
			slice[i] = inputPointer
		case pb.Arg_Bytes:
			inputPointer, err := w.SetBytes(arg.Value)
			if err != nil {
				return nil, err
			}
			slice[i] = inputPointer
		case pb.Arg_Bool:
			inputPointer, err := strconv.Atoi(string(arg.Value))
			if err != nil {
				return nil, err
			}
			slice[i] = inputPointer
		default:
			return nil, fmt.Errorf("input type not support")
		}
	}

	w.Instance.SetContextData(w.argMap)

	result, err := methodName(slice...)
	if err != nil {
		return nil, err
	}

	return []byte(result.String()), err
}

func createAddress(b types.Address, nonce uint64) types.Address {
	data, _ := rlp.EncodeToBytes([]interface{}{b, nonce})
	hashBytes := sha256.Sum256(data)

	return types.Bytes2Address(hashBytes[12:])
}
