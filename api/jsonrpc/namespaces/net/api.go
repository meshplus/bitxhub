package net

import (
	"fmt"

	"github.com/meshplus/bitxhub/internal/repo"
)

// PublicNetAPI is the eth_ prefixed set of APIs in the Web3 JSON-RPC spec.
type PublicNetAPI struct {
	networkVersion uint64
}

// NewAPI creates an instance of the public Net Web3 API.
func NewAPI(config *repo.Config) *PublicNetAPI {
	// parse the chainID from a integer string
	return &PublicNetAPI{
		networkVersion: config.ChainID,
	}
}

// Version returns the current ethereum protocol version.
func (api *PublicNetAPI) Version() string {
	return fmt.Sprintf("%d", api.networkVersion)
}
