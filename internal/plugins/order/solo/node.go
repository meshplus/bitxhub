package main

import (
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/order/solo"
)

func NewNode(opts ...order.Option) (order.Order, error) {
	return solo.NewNode(opts...)
}
