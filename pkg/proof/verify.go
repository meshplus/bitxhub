package proof

import (
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

//go:generate mockgen -destination mock_proof/mock_proof.go -package mock_proof -source verify.go
type Verify interface {
	// CheckProof verifies ibtp proof in interchain transaction
	CheckProof(tx pb.Transaction) (bool, uint64, error)

	// ValidationEngine returns validation engine
	ValidationEngine() validator.Engine

	// GetProof gets proof by transaction hash
	GetProof(txHash types.Hash) ([]byte, bool)

	// DeleteProof deletes proof in verify pool
	DeleteProof(txHash types.Hash)
}
