package etcdraft

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-core/order"
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-model/constant"

	"github.com/meshplus/bitxhub/internal/ledger"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/golang/mock/gomock"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/order/mempool"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const to = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"

func constructTx(nonce uint64) pb.Transaction {
	privK, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	pubKey := privK.PublicKey()
	addr, _ := pubKey.Address()
	tx := &pb.BxhTransaction{Nonce: nonce}
	tx.Timestamp = time.Now().UnixNano()
	tx.From = addr
	sig, _ := privK.Sign(tx.SignHash().Bytes())
	tx.Signature = sig
	tx.TransactionHash = tx.Hash()
	tx.Nonce = nonce
	return tx
}

func mockRaftNode(t *testing.T) (*Node, error) {
	logger := log.NewWithModule("consensus")
	batchTimerMgr := NewTimer(500*time.Millisecond, logger)
	txCache := mempool.NewTxCache(25*time.Millisecond, uint64(2), logger)
	walDir := filepath.Join("./testdata/storage", "wal")
	snapDir := filepath.Join("./testdata/storage", "snap")
	dbDir := filepath.Join("./testdata/storage", "state")
	raftStorage, dbStorage, _ := CreateStorage(logger, walDir, snapDir, dbDir, raft.NewMemoryStorage())
	repoRoot := "./testdata/"
	raftConfig, timedGenBlock, _ := generateRaftConfig(repoRoot)
	peerCnt := 4
	swarms, _ := newSwarms(t, peerCnt, false)
	mempoolConf := &mempool.Config{
		ID:             uint64(1),
		ChainHeight:    uint64(1),
		Logger:         logger,
		BatchSize:      raftConfig.RAFT.MempoolConfig.BatchSize,
		PoolSize:       raftConfig.RAFT.MempoolConfig.PoolSize,
		TxSliceSize:    raftConfig.RAFT.MempoolConfig.TxSliceSize,
		TxSliceTimeout: raftConfig.RAFT.MempoolConfig.TxSliceTimeout,
		StoragePath:    filepath.Join(repoRoot, "storage"),
		GetAccountNonce: func(address *types.Address) uint64 {
			return 0
		},

		IsTimed:      timedGenBlock.Enable,
		BlockTimeout: timedGenBlock.BlockTimeout,
	}
	mempoolInst, err := mempool.NewMempool(mempoolConf)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	node := &Node{
		id:               uint64(1),
		lastExec:         uint64(1),
		repoRoot:         repoRoot,
		logger:           logger,
		confChangeC:      make(chan raftpb.ConfChange),
		commitC:          make(chan *pb.CommitEvent, 1024),
		errorC:           make(chan<- error),
		msgC:             make(chan []byte),
		stateC:           make(chan *mempool.ChainState),
		proposeC:         make(chan *raftproto.RequestBatch),
		txCache:          txCache,
		batchTimerMgr:    batchTimerMgr,
		storage:          dbStorage,
		raftStorage:      raftStorage,
		mempool:          mempoolInst,
		tickTimeout:      500 * time.Millisecond,
		checkInterval:    3 * time.Minute,
		ctx:              ctx,
		cancel:           cancel,
		peerMgr:          swarms[0],
		getChainMetaFunc: getChainMetaFunc,
		isTimed:          mempoolConf.IsTimed,
		blockTimeout:     mempoolConf.BlockTimeout,
	}
	node.syncer = &mockSync{}
	return node, nil
}

type mockSync struct{}

// SyncCFTBlocks fetches the block list from other node, and just fetches but not verifies the block
func (sync *mockSync) SyncCFTBlocks(begin, end uint64, blockCh chan *pb.Block) error {
	for i := begin; i <= end; i++ {
		blockHeight := i
		header := &pb.BlockHeader{Number: blockHeight}
		blockHash := &types.Hash{
			RawHash: [types.HashLength]byte{0},
		}
		blockCh <- &pb.Block{BlockHeader: header, BlockHash: blockHash, Transactions: &pb.Transactions{}}
	}
	blockCh <- nil
	return nil
}

// SyncBFTBlocks fetches the block list from quorum nodes, and verifies all the block
func (sync *mockSync) SyncBFTBlocks(begin, end uint64, metaHash *types.Hash, blockCh chan *pb.Block) error {
	return nil
}

func getChainMetaFunc() *pb.ChainMeta {
	blockHash := &types.Hash{
		RawHash: [types.HashLength]byte{1},
	}
	return &pb.ChainMeta{
		Height:    uint64(1),
		BlockHash: blockHash,
	}
}
func listen(t *testing.T, order order.Order, swarm *peermgr.Swarm) {
	orderMsgCh := make(chan orderPeerMgr.OrderMessageEvent)
	sub := swarm.SubscribeOrderMessage(orderMsgCh)
	defer sub.Unsubscribe()
	for {
		select {
		case ev := <-orderMsgCh:
			err := order.Step(ev.Data)
			require.Nil(t, err)
		}
	}
}

func generateTx() pb.Transaction {
	privKey, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	from, _ := privKey.PublicKey().Address()
	tx := &pb.BxhTransaction{
		From:      from,
		To:        types.NewAddressByStr(to),
		Timestamp: time.Now().UnixNano(),
		Nonce:     0,
	}
	_ = tx.Sign(privKey)
	tx.TransactionHash = tx.Hash()
	return tx
}

func peers(id uint64, addrs []string, ids []string, accounts []string) []*repo.NetworkNodes {
	m := make([]*repo.NetworkNodes, 0, len(addrs))
	for i, addr := range addrs {
		m = append(m, &repo.NetworkNodes{
			ID:      uint64(i + 1),
			Account: accounts[i],
			Pid:     ids[i],
			Hosts:   []string{addr},
		})
	}
	return m
}

func genKeysAndConfig(t *testing.T, peerCnt int) ([]crypto2.PrivKey, []crypto.PrivateKey, []string, []string, []string) {
	var nodeKeys []crypto2.PrivKey
	var privKeys []crypto.PrivateKey
	var peers []string
	var pids []string
	var accounts []string

	port := 5001

	for i := 0; i < peerCnt; i++ {
		key, err := asym.GenerateKeyPair(crypto.ECDSA_P256)
		require.Nil(t, err)

		libp2pKey, err := convertToLibp2pPrivKey(key)
		require.Nil(t, err)
		nodeKeys = append(nodeKeys, libp2pKey)
		id, err := peer.IDFromPublicKey(libp2pKey.GetPublic())
		require.Nil(t, err)

		peer := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/", port)
		peers = append(peers, peer)
		pids = append(pids, id.String())
		port++

		privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
		require.Nil(t, err)

		privKeys = append(privKeys, privKey)

		account, err := privKey.PublicKey().Address()
		require.Nil(t, err)
		accounts = append(accounts, account.String())
	}

	return nodeKeys, privKeys, peers, pids, accounts
}

func convertToLibp2pPrivKey(privateKey crypto.PrivateKey) (crypto2.PrivKey, error) {
	ecdsaPrivKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("convert to libp2p private key: not ecdsa private key")
	}

	libp2pPrivKey, _, err := crypto2.ECDSAKeyPairFromKey(ecdsaPrivKey.K)
	if err != nil {
		return nil, err
	}

	return libp2pPrivKey, nil
}

func newSwarms(t *testing.T, peerCnt int, certVerify bool) ([]*peermgr.Swarm, map[uint64]*pb.VpInfo) {
	var swarms []*peermgr.Swarm
	nodes := make(map[uint64]*pb.VpInfo)
	nodeKeys, privKeys, addrs, pids, accounts := genKeysAndConfig(t, peerCnt)
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	for i := 0; i < peerCnt; i++ {
		node := &node_mgr.Node{
			Account:  accounts[i],
			VPNodeId: uint64(i),
			Pid:      pids[i],
			Status:   governance.GovernanceAvailable,
		}
		nodeData, err := json.Marshal(node)
		require.Nil(t, err)
		stateLedger.EXPECT().GetState(constant.NodeManagerContractAddr.Address(), []byte(node_mgr.NodeKey(accounts[i]))).Return(true, nodeData).AnyTimes()
		stateLedger.EXPECT().GetState(constant.NodeManagerContractAddr.Address(), []byte(node_mgr.VpNodePidKey(pids[i]))).Return(true, []byte(accounts[i])).AnyTimes()
	}
	stateLedger.EXPECT().Copy().Return(stateLedger).AnyTimes()

	agencyData, err := ioutil.ReadFile("testdata/agency.cert")
	require.Nil(t, err)

	nodeData, err := ioutil.ReadFile("testdata/node.cert")
	require.Nil(t, err)

	caData, err := ioutil.ReadFile("testdata/ca.cert")
	require.Nil(t, err)

	cert, err := libp2pcert.ParseCert(caData)
	require.Nil(t, err)

	for i := 0; i < peerCnt; i++ {
		ID := i + 1
		repo := &repo.Repo{
			Key: &repo.Key{},
			NetworkConfig: &repo.NetworkConfig{
				N:  uint64(peerCnt),
				ID: uint64(ID),
			},
			Certs: &libp2pcert.Certs{
				NodeCertData:   nodeData,
				AgencyCertData: agencyData,
				CACert:         cert,
			},
			Config: &repo.Config{
				Ping: repo.Ping{
					Duration: 2 * time.Second,
				},
			},
		}

		if certVerify {
			repo.Config.Cert.Verify = true
		} else {
			repo.Config.Cert.Verify = false
		}

		idx := strings.LastIndex(addrs[i], "/p2p/")
		local := addrs[i][:idx]
		repo.NetworkConfig.LocalAddr = local
		repo.Key.Libp2pPrivKey = nodeKeys[i]
		repo.Key.PrivKey = privKeys[i]
		repo.NetworkConfig.Nodes = peers(uint64(i), addrs, pids, accounts)

		address, err := privKeys[i].PublicKey().Address()
		require.Nil(t, err)
		vpInfo := &pb.VpInfo{
			Id:      uint64(ID),
			Account: address.String(),
		}
		nodes[uint64(ID)] = vpInfo
		swarm, err := peermgr.New(repo, log.NewWithModule("p2p"), mockLedger)
		require.Nil(t, err)
		err = swarm.Start()
		require.Nil(t, err)
		swarms = append(swarms, swarm)
	}
	return swarms, nodes
}

func stopSwarms(t *testing.T, swarms []*peermgr.Swarm) error {
	for _, swarm := range swarms {
		err := swarm.Stop()
		assert.Nil(t, err)
	}
	return nil
}
