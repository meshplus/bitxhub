package proof

import (
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

type Verify interface {
	// CheckProof verifies ibtp proof in interchain transaction
	CheckProof(tx *pb.Transaction) (bool, error)

	// ValidationEngine returns validation engine
	ValidationEngine() *validator.ValidationEngine

	// GetProof gets proof by transaction hash
	GetProof(txHash types.Hash) ([]byte, bool)

	// DeleteProof deletes proof in verify pool
	DeleteProof(txHash types.Hash)
}
