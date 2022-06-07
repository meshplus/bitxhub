package appchain

import (
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
)

func TestNewAppchainClient(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestRopstenLightClient")
	require.Nil(t, err)
	clientRes, err := NewAppchainClient("../../../../config/appchain/eth_header1.json", repoRoot, log.NewWithModule("test"))
	require.Nil(t, err)
	require.NotNil(t, clientRes)
	clientRes2, err := NewAppchainClient("", repoRoot, log.NewWithModule("test"))
	require.Nil(t, clientRes2)
}
