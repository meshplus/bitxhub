package evm

import (
	"github.com/meshplus/bitxhub/pkg/vm"
)

type EtherVM struct {
	ctx *vm.Context
}

func New(ctx *vm.Context) (*EtherVM, error) {
	return &EtherVM{
		ctx: ctx,
	}, nil
}
