package etcdraft

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const to = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"

func TestNode_Start(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "node")
	assert.Nil(t, err)
	defer os.RemoveAll(repoRoot)

	var ID uint64 = 1
	nodes := make(map[uint64]*pb.VpInfo)
	vpInfo := &pb.VpInfo{
		Id:      ID,
		Account: types.NewAddressByStr("000000000000000000000000000000000000000a").String(),
	}
	nodes[ID] = vpInfo
	fileData, err := ioutil.ReadFile("./testdata/order.toml")
	require.Nil(t, err)
	err = ioutil.WriteFile(filepath.Join(repoRoot, "order.toml"), fileData, 0644)
	require.Nil(t, err)

	mockCtl := gomock.NewController(t)

	mockPeermgr := mock_peermgr.NewMockPeerManager(mockCtl)
	peers := make(map[uint64]*pb.VpInfo)
	otherPeers := make(map[uint64]*peer.AddrInfo, 5)
	mockPeermgr.EXPECT().Peers().Return(peers).AnyTimes()
	mockPeermgr.EXPECT().OtherPeers().Return(otherPeers).AnyTimes()
	mockPeermgr.EXPECT().Broadcast(gomock.Any()).AnyTimes()

	order, err := NewNode(
		order.WithRepoRoot(repoRoot),
		order.WithID(ID),
		order.WithNodes(nodes),
		order.WithPeerManager(mockPeermgr),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithLogger(log.NewWithModule("consensus")),
		order.WithApplied(1),
	)
	require.Nil(t, err)

	err = order.Start()
	require.Nil(t, err)

	for {
		time.Sleep(200 * time.Millisecond)
		err := order.Ready()
		if err == nil {
			break
		}
	}
	tx := generateTx()
	err = order.Prepare(tx)
	require.Nil(t, err)

	commitEvent := <-order.Commit()
	require.Equal(t, uint64(2), commitEvent.Block.BlockHeader.Number)
	require.Equal(t, 1, len(commitEvent.Block.Transactions))

	order.Stop()
}

func TestMulti_Node_Start(t *testing.T) {
	peerCnt := 4
	swarms, nodes := newSwarms(t, peerCnt)

	//time.Sleep(3 * time.Second)
	repoRoot, err := ioutil.TempDir("", "nodes")
	defer os.RemoveAll(repoRoot)

	fileData, err := ioutil.ReadFile("../../../config/order.toml")
	require.Nil(t, err)

	orders := make([]order.Order, 0)
	for i := 0; i < peerCnt; i++ {
		nodePath := fmt.Sprintf("node%d", i)
		nodeRepo := filepath.Join(repoRoot, nodePath)
		err := os.Mkdir(nodeRepo, 0744)
		require.Nil(t, err)
		orderPath := filepath.Join(nodeRepo, "order.toml")
		err = ioutil.WriteFile(orderPath, fileData, 0744)
		require.Nil(t, err)

		ID := i + 1
		order, err := NewNode(
			order.WithRepoRoot(nodeRepo),
			order.WithID(uint64(ID)),
			order.WithNodes(nodes),
			order.WithPeerManager(swarms[i]),
			order.WithStoragePath(repo.GetStoragePath(nodeRepo, "order")),
			order.WithLogger(log.NewWithModule("consensus")),
			order.WithGetBlockByHeightFunc(nil),
			order.WithApplied(1),
		)
		require.Nil(t, err)
		err = order.Start()
		require.Nil(t, err)
		orders = append(orders, order)
		go listen(t, order, swarms[i])
	}

	for {
		time.Sleep(200 * time.Millisecond)
		err := orders[0].Ready()
		if err == nil {
			break
		}
	}
	tx := generateTx()
	err = orders[0].Prepare(tx)
	require.Nil(t, err)
	for i := 0; i < len(orders); i++ {
		commitEvent := <-orders[i].Commit()
		require.Equal(t, uint64(2), commitEvent.Block.BlockHeader.Number)
		require.Equal(t, 1, len(commitEvent.Block.Transactions))
	}
}

func listen(t *testing.T, order order.Order, swarm *peermgr.Swarm) {
	orderMsgCh := make(chan events.OrderMessageEvent)
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

func generateTx() *pb.Transaction {
	privKey, _ := asym.GenerateKeyPair(crypto.Secp256k1)

	from, _ := privKey.PublicKey().Address()

	tx := &pb.Transaction{
		From:      from,
		To:        types.NewAddressByStr(to),
		Timestamp: time.Now().UnixNano(),
		Nonce:     1,
	}
	_ = tx.Sign(privKey)
	tx.TransactionHash = tx.Hash()
	return tx
}

func peers(id uint64, addrs []string, ids []string) []*repo.NetworkNodes {
	m := make([]*repo.NetworkNodes, 0, len(addrs))
	for i, addr := range addrs {
		m = append(m, &repo.NetworkNodes{
			ID:      uint64(i + 1),
			Account: "",
			Pid:     ids[i],
			Hosts:   []string{addr},
		})
	}
	return m
}

func genKeysAndConfig(t *testing.T, peerCnt int) ([]crypto2.PrivKey, []crypto.PrivateKey, []string, []string) {
	var nodeKeys []crypto2.PrivKey
	var privKeys []crypto.PrivateKey
	var peers []string
	var ids []string

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
		ids = append(ids, id.String())
		port++

		privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
		require.Nil(t, err)

		privKeys = append(privKeys, privKey)
	}

	return nodeKeys, privKeys, peers, ids
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

func newSwarms(t *testing.T, peerCnt int) ([]*peermgr.Swarm, map[uint64]*pb.VpInfo) {
	var swarms []*peermgr.Swarm
	nodes := make(map[uint64]*pb.VpInfo)
	nodeKeys, privKeys, addrs, ids := genKeysAndConfig(t, peerCnt)
	mockCtl := gomock.NewController(t)
	mockLedger := mock_ledger.NewMockLedger(mockCtl)

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
		idx := strings.LastIndex(addrs[i], "/p2p/")
		local := addrs[i][:idx]
		repo.NetworkConfig.LocalAddr = local
		repo.Key.Libp2pPrivKey = nodeKeys[i]
		repo.Key.PrivKey = privKeys[i]
		repo.NetworkConfig.Nodes = peers(uint64(i), addrs, ids)

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
