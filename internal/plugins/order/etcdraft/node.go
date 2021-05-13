package main

import (
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/order/etcdraft"
)

func NewNode(opts ...order.Option) (order.Order, error) {
	return etcdraft.NewNode(opts...)
}
