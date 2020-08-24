package cert

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/libp2p/go-libp2p-core/crypto"
	peer2 "github.com/libp2p/go-libp2p-core/peer"
	crypto2 "github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePrivateKey(t *testing.T) {
	data, err := ioutil.ReadFile(filepath.Join("testdata", "ca.priv"))
	assert.Nil(t, err)
	privKey, err := ParsePrivateKey(data, crypto2.ECDSA_P256)
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

func TestParsePrivateKey2(t *testing.T) {
	privKey, err := asym.GenerateKeyPair(crypto2.ECDSA_P256)
	require.Nil(t, err)

	priKeyEncode, err := privKey.Bytes()
	require.Nil(t, err)

	f, err := os.Create("./key.priv")
	require.Nil(t, err)

	err = pem.Encode(f, &pem.Block{Type: "EC PRIVATE KEY", Bytes: priKeyEncode})
	require.Nil(t, err)

	data, err := ioutil.ReadFile("./key.priv")
	assert.Nil(t, err)

	privKey1, err := ParsePrivateKey(data, crypto2.ECDSA_P256)
	assert.Nil(t, err)

	_, pk, err := crypto.KeyPairFromStdKey(privKey1.K)
	assert.Nil(t, err)

	pid, err := peer2.IDFromPublicKey(pk)
	assert.Nil(t, err)

	fmt.Println(pid.String())
}
