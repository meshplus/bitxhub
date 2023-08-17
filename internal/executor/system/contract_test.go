package system

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
)

var systemContractAddrs = []string{
	common.NodeManagerContractAddr,
}

var notSystemContractAddrs = []string{
	"0x1000000000000000000000000000000000000000",
	"0x0340000000000000000000000000000000000000",
	"0x0200000000000000000000000000000000000000",
	"0xffddd00000000000000000000000000000000000",
}

func TestContract_GetSystemContract(t *testing.T) {
	Initialize(logrus.New())

	for _, addr := range systemContractAddrs {
		contract, ok := GetSystemContract(types.NewAddressByStr(addr))
		assert.True(t, ok)
		assert.NotNil(t, contract)
	}

	for _, addr := range notSystemContractAddrs {
		contract, ok := GetSystemContract(types.NewAddressByStr(addr))
		assert.False(t, ok)
		assert.Nil(t, contract)
	}

	// test nil address
	contract, ok := GetSystemContract(nil)
	assert.False(t, ok)
	assert.Nil(t, contract)

	// test empty address
	contract, ok = GetSystemContract(&types.Address{})
	assert.False(t, ok)
	assert.Nil(t, contract)
}
