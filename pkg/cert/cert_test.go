package cert

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestParsePrivateKey(t *testing.T) {
	data, err := ioutil.ReadFile(filepath.Join("testdata", "ca.priv"))
	assert.Nil(t, err)
	privKey, err := ParsePrivateKey(data)
	assert.Nil(t, err)
	assert.NotNil(t, privKey)
}

func TestVerifySign(t *testing.T) {
	data, err := ioutil.ReadFile(filepath.Join("testdata", "ca.cert"))
	require.Nil(t, err)
	caCert, err := ParseCert(data)
	require.Nil(t, err)

	subData, err := ioutil.ReadFile(filepath.Join("testdata", "agency.cert"))
	require.Nil(t, err)
	subCert, err := ParseCert(subData)
	require.Nil(t, err)
	err = VerifySign(subCert, caCert)
	require.Nil(t, err)

	nodeData, err := ioutil.ReadFile(filepath.Join("testdata", "node.cert"))
	require.Nil(t, err)
	nodeCert, err := ParseCert(nodeData)
	require.Nil(t, err)
	err = VerifySign(nodeCert, subCert)
	require.Nil(t, err)
}
