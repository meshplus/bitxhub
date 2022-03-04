package benchmark

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-core/order"
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const to = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"

var count int

func listen(t *testing.T, order order.Order, swarm *peermgr.Swarm, mockExecutor *mockExecutor) {
	orderMsgCh := make(chan orderPeerMgr.OrderMessageEvent)
	sub := swarm.SubscribeOrderMessage(orderMsgCh)
	defer sub.Unsubscribe()
	for {
		select {
		case ev := <-mockExecutor.blockC:
			go func() {
				mockExecutor.endBlockC <- ev.Block
				order.ReportState(ev.Block.BlockHeader.Number, ev.Block.BlockHash, ev.TxHashList)
			}()
		case ev := <-orderMsgCh:
			//msg := &rbftpb.ConsensusMessage{}
			//
			//p2pmsg := &pb.Message{}
			//err := p2pmsg.Unmarshal(ev.Data)
			//if err != nil {
			//	fmt.Errorf("unmarshal p2p msg err")
			//}
			//if p2pmsg.Type == pb.Message_CONSENSUS {
			//	err = msg.Unmarshal(p2pmsg.Data)
			//	if err != nil {
			//		fmt.Errorf("unmarshal msg err")
			//	}
			//	if msg.Type == rbftpb.Type_CHECKPOINT {
			//		count++
			//		fmt.Printf("------------get checkpoint:%d", count)
			//	}
			//}
			go func() {
				err := order.Step(ev.Data)
				require.Nil(t, err)
			}()
		}
	}
}

func generateTx(privKey crypto.PrivateKey, nonce uint64) pb.Transaction {
	from, _ := privKey.PublicKey().Address()
	tx := &pb.BxhTransaction{
		From:      from,
		To:        types.NewAddressByStr(to),
		Timestamp: time.Now().UnixNano(),
		Nonce:     nonce,
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

type mockExecutor struct {
	blockC    chan *executedEvent
	endBlockC chan *pb.Block
	id        int
}

type executedEvent struct {
	Block          *pb.Block
	InterchainMeta *pb.InterchainMeta
	TxHashList     []*types.Hash
}

func newMockExecutor(id int) *mockExecutor {
	return &mockExecutor{
		blockC:    make(chan *executedEvent, 1024),
		endBlockC: make(chan *pb.Block, 1024000),
		id:        id,
	}
}
