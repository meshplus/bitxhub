package mempool

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/storage/leveldb"
	network "github.com/meshplus/go-lightp2p"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/mock"
)

var (
	InterchainContractAddr = types.String2Address("000000000000000000000000000000000000000a")
)

const (
	DefaultTestChainHeight = uint64(1)
	DefaultTestBatchSize   = uint64(4)
	DefaultTestTxSetSize   = uint64(1)
	LevelDBDir             = "test-db"
)

func mockMempoolImpl() (*mempoolImpl, chan *raftproto.Ready) {
	config := &Config{
		ID:             1,
		ChainHeight:    DefaultTestChainHeight,
		BatchSize:      DefaultTestBatchSize,
		PoolSize:       DefaultPoolSize,
		TxSliceSize:    DefaultTestTxSetSize,
		BatchTick:      DefaultBatchTick,
		FetchTimeout:   DefaultFetchTxnTimeout,
		TxSliceTimeout: DefaultTxSetTick,
		Logger:         log.NewWithModule("consensus"),
	}
	config.PeerMgr = newMockPeerMgr()
	db, _ := leveldb.New(LevelDBDir)
	proposalC := make(chan *raftproto.Ready)
	mempool := newMempoolImpl(config, db, proposalC)
	return mempool, proposalC
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

func constructIBTPTx(nonce uint64, privKey *crypto.PrivateKey) *pb.Transaction {
	var privK crypto.PrivateKey
	if privKey == nil {
		privK = genPrivKey()
	}
	privK = *privKey
	pubKey := privK.PublicKey()
	from, _ := pubKey.Address()
	to := from.Hex()
	ibtp := mockIBTP(from.Hex(), to, nonce)
	tx := mockInterChainTx(ibtp)
	tx.Timestamp = time.Now().UnixNano()
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

func mockInterChainTx(ibtp *pb.IBTP) *pb.Transaction {
	ib, _ := ibtp.Marshal()
	ipd := &pb.InvokePayload{
		Method: "HandleIBTP",
		Args:   []*pb.Arg{{Value: ib}},
	}
	pd, _ := ipd.Marshal()
	data := &pb.TransactionData{
		VmType:  pb.TransactionData_BVM,
		Type:    pb.TransactionData_INVOKE,
		Payload: pd,
	}
	return &pb.Transaction{
		To:    InterchainContractAddr,
		Nonce: ibtp.Index,
		Data:  data,
		Extra: []byte(fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Type)),
	}
}

func mockIBTP(from, to string, nonce uint64) *pb.IBTP {
	content := pb.Content{
		SrcContractId: from,
		DstContractId: from,
		Func:          "interchainget",
		Args:          [][]byte{[]byte("Alice"), []byte("10")},
	}
	bytes, _ := content.Marshal()
	ibtppd, _ := json.Marshal(pb.Payload{
		Encrypted: false,
		Content:   bytes,
	})
	return &pb.IBTP{
		From:      from,
		To:        to,
		Payload:   ibtppd,
		Index:     nonce,
		Type:      pb.IBTP_INTERCHAIN,
		Timestamp: time.Now().UnixNano(),
	}
}

type mockPeerMgr struct {
	mock.Mock
	EventChan chan *pb.Message
}

func newMockPeerMgr() *mockPeerMgr {
	return &mockPeerMgr{}
}

func (mpm *mockPeerMgr) Broadcast(msg *pb.Message) error {
	mpm.EventChan <- msg
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
