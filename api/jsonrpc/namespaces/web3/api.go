package web3

import (
	"fmt"

	"github.com/axiomesh/axiom"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// PublicWeb3API is the web3_ prefixed set of APIs in the Web3 JSON-RPC spec.
type PublicWeb3API struct{}

// NewAPI creates an instance of the Web3 API.
func NewAPI() *PublicWeb3API {
	return &PublicWeb3API{}
}

// ClientVersion returns the client version in the Web3 user agent format.
func (PublicWeb3API) ClientVersion() string {
	return fmt.Sprintf("%s-%s-%s", axiom.CurrentVersion, axiom.CurrentBranch, axiom.CurrentCommit)
}

// Sha3 returns the keccak-256 hash of the passed-in input.
func (PublicWeb3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	return crypto.Keccak256(input)
}
