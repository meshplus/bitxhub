package network

import (
	"github.com/meshplus/bitxhub/pkg/network/pb"
)

func Message(data []byte) *pb.Message {
	return &pb.Message{
		Data: data,
	}
}
