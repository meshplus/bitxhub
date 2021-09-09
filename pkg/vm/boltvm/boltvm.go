package boltvm

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/pkg/vm"
	evm "github.com/meshplus/eth-kit/evm"
)

var _ vm.VM = (*BoltVM)(nil)

type BoltVM struct {
	ctx       *vm.Context
	evm       *evm.EVM
	ve        validator.Engine
	contracts map[string]agency.Contract
}

// New creates a blot vm object
func New(ctx *vm.Context, ve validator.Engine, evm *evm.EVM, contracts map[string]agency.Contract) *BoltVM {
	return &BoltVM{
		ctx:       ctx,
		ve:        ve,
		evm:       evm,
		contracts: contracts,
	}
}

func (bvm *BoltVM) Run(input []byte, _ uint64) (ret []byte, gasUsed uint64, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()

	return bvm.InvokeBVM(bvm.ctx.Callee.String(), input)
}

func (bvm *BoltVM) HandleIBTP(ibtp *pb.IBTP) (ret []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()

	con := &contracts.InterchainManager{}
	con.Stub = &BoltStubImpl{
		bvm: bvm,
		ctx: bvm.ctx,
		ve:  bvm.ve,
	}

	_, err = GetBoltContract(bvm.ctx.Callee.String(), bvm.contracts)
	if err != nil {
		return nil, fmt.Errorf("get bolt contract: %w", err)
	}

	res := con.HandleIBTP(ibtp)
	if !res.Ok {
		return nil, fmt.Errorf("call error: %s", res.Result)
	}

	return res.Result, err
}

func (bvm *BoltVM) InvokeBVM(address string, input []byte) (ret []byte, _ uint64, err error) {
	payload := &pb.InvokePayload{}
	if err := payload.Unmarshal(input); err != nil {
		return nil, 0, fmt.Errorf("unmarshal invoke payload: %w", err)
	}

	method, ins := payload.Method, payload.Args
	contract, err := GetBoltContract(address, bvm.contracts)
	if err != nil {
		return nil, 0, fmt.Errorf("get bolt contract: %w", err)
	}

	rc := reflect.ValueOf(contract)
	stubField := rc.Elem().Field(0)
	stub := &BoltStubImpl{
		bvm: bvm,
		ctx: bvm.ctx,
		ve:  bvm.ve,
	}

	if stubField.CanSet() {
		stubField.Set(reflect.ValueOf(stub))
	} else {
		return nil, 0, fmt.Errorf("stub filed can`t set")
	}

	// judge whether method is valid
	m := rc.MethodByName(method)
	if !m.IsValid() {
		return nil, 0, fmt.Errorf("not such method `%s`", method)
	}

	fnArgs, err := parseArgs(ins)
	if err != nil {
		return nil, 0, fmt.Errorf("parse args: %w", err)
	}

	res := m.Call(fnArgs)[0].Interface().(*boltvm.Response)
	if !res.Ok {
		return nil, 0, fmt.Errorf("call error: %s", res.Result)
	}
	return res.Result, 0, err
}

func parseArgs(in []*pb.Arg) ([]reflect.Value, error) {
	args := make([]reflect.Value, len(in))
	for i := 0; i < len(in); i++ {
		switch in[i].Type {
		case pb.Arg_F64:
			ret, err := strconv.ParseFloat(string(in[i].Value), 64)
			if err != nil {
				return nil, err
			}

			args[i] = reflect.ValueOf(float64(ret))
		case pb.Arg_I32:
			ret, err := strconv.Atoi(string(in[i].Value))
			if err != nil {
				return nil, err
			}

			args[i] = reflect.ValueOf(int32(ret))
		case pb.Arg_I64:
			ret, err := strconv.Atoi(string(in[i].Value))
			if err != nil {
				return nil, err
			}

			args[i] = reflect.ValueOf(int64(ret))
		case pb.Arg_U64:
			ret, err := strconv.ParseUint(string(in[i].Value), 10, 64)
			if err != nil {
				return nil, err
			}
			args[i] = reflect.ValueOf(ret)
		case pb.Arg_String:
			args[i] = reflect.ValueOf(string(in[i].Value))
		case pb.Arg_Bytes:
			args[i] = reflect.ValueOf(in[i].Value)
		case pb.Arg_Bool:
			ret, err := strconv.ParseBool(string(in[i].Value))
			if err != nil {
				return nil, err
			}
			args[i] = reflect.ValueOf(ret)
		default:
			args[i] = reflect.ValueOf(string(in[i].Value))
		}
	}
	return args, nil
}
