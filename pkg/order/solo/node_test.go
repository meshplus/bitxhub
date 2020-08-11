package solo

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const to = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"

func TestNode_Start(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "node")
	defer os.RemoveAll(repoRoot)
	assert.Nil(t, err)
	order, err := NewNode(
		order.WithRepoRoot(repoRoot),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithLogger(log.NewWithModule("consensus")),
		order.WithApplied(1),
	)
	require.Nil(t, err)

	err = order.Start()
	require.Nil(t, err)

	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	tx := &pb.Transaction{
		From: from,
		To:   types.String2Address(to),
		Data: &pb.TransactionData{
			Amount: 10,
		},
		Timestamp: time.Now().UnixNano(),
		Nonce:     rand.Int63(),
	}
	err = tx.Sign(privKey)
	require.Nil(t, err)

	for {
		time.Sleep(200 * time.Millisecond)
		if order.Ready() {
			break
		}
	}

	err = order.Prepare(tx)
	require.Nil(t, err)

	block := <-order.Commit()
	require.Equal(t, uint64(2), block.BlockHeader.Number)
	require.Equal(t, 1, len(block.Transactions))

	order.Stop()
}
