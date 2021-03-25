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
)

var _ vm.VM = (*BoltVM)(nil)

type BoltVM struct {
	ctx       *vm.Context
	ve        validator.Engine
	contracts map[string]agency.Contract
}

// New creates a blot vm object
func New(ctx *vm.Context, ve validator.Engine, contracts map[string]agency.Contract) *BoltVM {
	return &BoltVM{
		ctx:       ctx,
		ve:        ve,
		contracts: contracts,
	}
}

func (bvm *BoltVM) Run(input []byte) (ret []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()

	payload := &pb.InvokePayload{}
	if err := payload.Unmarshal(input); err != nil {
		return nil, fmt.Errorf("unmarshal invoke payload: %w", err)
	}

	contract, err := GetBoltContract(bvm.ctx.Callee.String(), bvm.contracts)
	if err != nil {
		return nil, fmt.Errorf("get bolt contract: %w", err)
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
		return nil, fmt.Errorf("stub filed can`t set")
	}

	// judge whether method is valid
	m := rc.MethodByName(payload.Method)
	if !m.IsValid() {
		return nil, fmt.Errorf("not such method `%s`", payload.Method)
	}

	fnArgs, err := parseArgs(payload.Args)
	if err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	res := m.Call(fnArgs)[0].Interface().(*boltvm.Response)
	if !res.Ok {
		return nil, fmt.Errorf("call error: %s", res.Result)
	}

	return res.Result, err
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

func parseArgs(in []*pb.Arg) ([]reflect.Value, error) {
	args := make([]reflect.Value, len(in))
	for i := 0; i < len(in); i++ {
		switch in[i].Type {
		case pb.Arg_I32:
			ret, err := strconv.Atoi(string(in[i].Value))
			if err != nil {
				return nil, err
			}

			args[i] = reflect.ValueOf(int32(ret))
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
		default:
			args[i] = reflect.ValueOf(string(in[i].Value))
		}
	}
	return args, nil
}
