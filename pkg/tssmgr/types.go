package tssmgr

import (
	"crypto/ecdsa"

	"github.com/meshplus/bitxhub-model/pb"
)

//go:generate mockgen -destination mock_tss/mock_tss.go -package mock_tss -source types.go
type TssManager interface {
	Start(t uint64)

	Stop()

	Keygen(isKeygenReq bool) error

	Keysign(signers []string, msgs []string, randomN string) ([]byte, []string, error)

	PutTssMsg(msg *pb.Message, msgID string)

	// GetTssPubkey returns tss pool pubkey addr and pubkey
	GetTssPubkey() (string, *ecdsa.PublicKey, error)

	// GetTssInfo returns tss pubkey and participants pubkey info
	GetTssInfo() (*pb.TssInfo, error)

	DeleteTssNodes(nodes []string) error

	UpdateThreshold(threshold uint64)

	GetThreshold() uint64
}
