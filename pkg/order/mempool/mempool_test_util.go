package mempool

import (
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
)

var (
	InterchainContractAddr = types.NewAddressByStr("000000000000000000000000000000000000000a")
)

const (
	DefaultTestChainHeight = uint64(1)
	DefaultTestBatchSize   = uint64(4)
	DefaultTestTxSetSize   = uint64(1)
)

func mockGetAccountNonce(address *types.Address) uint64 {
	return 0
}

func mockMempoolImpl(path string) (*mempoolImpl, chan *raftproto.Ready) {
	config := &Config{
		ID:              1,
		ChainHeight:     DefaultTestChainHeight,
		BatchSize:       DefaultTestBatchSize,
		PoolSize:        DefaultPoolSize,
		TxSliceSize:     DefaultTestTxSetSize,
		TxSliceTimeout:  DefaultTxSetTick,
		Logger:          log.NewWithModule("consensus"),
		StoragePath:     path,
		GetAccountNonce: mockGetAccountNonce,
	}
	proposalC := make(chan *raftproto.Ready)
	mempool, _ := NewMempool(config)
	mempoolImpl, ok := mempool.(*mempoolImpl)
	if !ok {
		return nil, nil
	}
	return mempoolImpl, proposalC
}

func genPrivKey() crypto.PrivateKey {
	privKey, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	return privKey
}

func constructTx(nonce uint64, privKey *crypto.PrivateKey) pb.Transaction {
	var privK crypto.PrivateKey
	if privKey == nil {
		privK = genPrivKey()
	}
	privK = *privKey
	pubKey := privK.PublicKey()
	addr, _ := pubKey.Address()
	tx := &pb.BxhTransaction{Nonce: nonce}
	tx.Timestamp = time.Now().UnixNano()
	tx.From = addr
	sig, _ := privK.Sign(tx.SignHash().Bytes())
	tx.Signature = sig
	tx.TransactionHash = tx.Hash()
	return tx
}
