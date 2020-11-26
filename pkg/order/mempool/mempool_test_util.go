package mempool

import (
	"errors"
	"math/rand"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	network "github.com/meshplus/go-lightp2p"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/mock"
)

var (
	InterchainContractAddr = types.NewAddressByStr("000000000000000000000000000000000000000a")
)

const (
	DefaultTestChainHeight = uint64(1)
	DefaultTestBatchSize   = uint64(4)
	DefaultTestTxSetSize   = uint64(1)
	LevelDBDir             = "test-db"
)

func mockMempoolImpl() (*mempoolImpl, chan *raftproto.Ready) {
	config := &Config{
		ID:                 1,
		ChainHeight:        DefaultTestChainHeight,
		BatchSize:          DefaultTestBatchSize,
		PoolSize:           DefaultPoolSize,
		TxSliceSize:        DefaultTestTxSetSize,
		BatchTick:          DefaultBatchTick,
		FetchTimeout:       DefaultFetchTxnTimeout,
		TxSliceTimeout:     DefaultTxSetTick,
		Logger:             log.NewWithModule("consensus"),
		GetTransactionFunc: getTransactionFunc,
	}
	config.PeerMgr = newMockPeerMgr()
	db, _ := leveldb.New(LevelDBDir)
	proposalC := make(chan *raftproto.Ready, 2)
	mempool := newMempoolImpl(config, db, proposalC)
	return mempool, proposalC
}

func getTransactionFunc(hash *types.Hash) (*pb.Transaction, error) {
	return nil, errors.New("can't find transaction")
}

func genPrivKey() crypto.PrivateKey {
	privKey, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	return privKey
}

func constructTx(nonce uint64, privKey *crypto.PrivateKey) *pb.Transaction {
	var privK crypto.PrivateKey
	if privKey == nil {
		privK = genPrivKey()
	}
	privK = *privKey
	pubKey := privK.PublicKey()
	addr, _ := pubKey.Address()
	tx := &pb.Transaction{Nonce: nonce}
	tx.Timestamp = time.Now().UnixNano()
	tx.From = addr
	sig, _ := privK.Sign(tx.SignHash().Bytes())
	tx.Signature = sig
	tx.TransactionHash = tx.Hash()
	return tx
}

func cleanTestData() bool {
	err := os.RemoveAll(LevelDBDir)
	if err != nil {
		return false
	}
	return true
}

func newHash() *types.Hash {
	hashBytes := make([]byte, types.HashLength)
	rand.Read(hashBytes)
	return types.NewHash(hashBytes)
}

type mockPeerMgr struct {
	mock.Mock
	EventChan chan *pb.Message
}

func newMockPeerMgr() *mockPeerMgr {
	return &mockPeerMgr{}
}

func (mpm *mockPeerMgr) Broadcast(msg *pb.Message) error {
	return nil
}

func (mpm *mockPeerMgr) AsyncSend(id uint64, msg *pb.Message) error {
	mpm.EventChan <- msg
	return nil
}

func (mpm *mockPeerMgr) Start() error {
	return nil
}

func (mpm *mockPeerMgr) Stop() error {
	return nil
}

func (mpm *mockPeerMgr) SendWithStream(network.Stream, *pb.Message) error {
	return nil
}

func (mpm *mockPeerMgr) Send(uint64, *pb.Message) (*pb.Message, error) {
	return nil, nil
}

func (mpm *mockPeerMgr) Peers() map[uint64]*peer.AddrInfo {
	peers := make(map[uint64]*peer.AddrInfo, 3)
	var id1 peer.ID
	id1 = "peer1"
	peers[0] = &peer.AddrInfo{ID: id1}
	id1 = "peer2"
	peers[1] = &peer.AddrInfo{ID: id1}
	id1 = "peer3"
	peers[2] = &peer.AddrInfo{ID: id1}
	return peers
}

func (mpm *mockPeerMgr) OtherPeers() map[uint64]*peer.AddrInfo {
	return nil
}

func (mpm *mockPeerMgr) SubscribeOrderMessage(ch chan<- events.OrderMessageEvent) event.Subscription {
	return nil
}
