package syncer

import (
	"fmt"
	"github.com/meshplus/bitxhub-kit/types"
	"io/ioutil"
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
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/cert"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/stretchr/testify/require"
)

func TestStateSyncer_SyncCFTBlocks(t *testing.T) {
	peerCnt := 3
	swarms := NewSwarms(t, peerCnt)

	otherPeers := swarms[0].OtherPeers()
	peerIds := make([]uint64, 0)
	for id, _ := range otherPeers {
		peerIds = append(peerIds, id)
	}
	logger := log.NewWithModule("syncer")
	syncer, err := New(10, swarms[0], 2, peerIds, logger)
	require.Nil(t, err)

	begin := 2
	end := 100
	blockCh := make(chan *pb.Block, 1024)
	go syncer.SyncCFTBlocks(uint64(begin), uint64(end), blockCh)

	blocks := make([]*pb.Block, 0)
	for block := range blockCh {
		if block == nil {
			break
		}
		blocks = append(blocks, block)
	}

	require.Equal(t, len(blocks), end-begin+1)

}

func TestStateSyncer_SyncBFTBlocks(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)

	//time.Sleep(100 * time.Millisecond)
	otherPeers := swarms[0].OtherPeers()
	peerIds := make([]uint64, 0)
	for id, _ := range otherPeers {
		peerIds = append(peerIds, id)
	}
	logger := log.NewWithModule("syncer")
	syncer, err := New(10, swarms[0], 3, peerIds, logger)
	require.Nil(t, err)

	begin := 2
	end := 100
	blockCh := make(chan *pb.Block, 1024)

	metaHash := types.NewHashByStr("0xbC1C6897f97782F3161492d5CcfBE0691502f15894A0b2f2f40069C995E33cCB")
	go syncer.SyncBFTBlocks(uint64(begin), uint64(end), metaHash, blockCh)

	blocks := make([]*pb.Block, 0)
	for block := range blockCh {
		if block == nil {
			break
		}
		blocks = append(blocks, block)
	}

	require.Equal(t, len(blocks), end-begin+1)
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

func NewSwarms(t *testing.T, peerCnt int) []*peermgr.Swarm {
	var swarms []*peermgr.Swarm

	blocks := genBlocks(1024)
	nodeKeys, privKeys, addrs, ids := genKeysAndConfig(t, peerCnt)
	mockCtl := gomock.NewController(t)
	mockLedger := mock_ledger.NewMockLedger(mockCtl)
	mockLedger.EXPECT().GetBlock(gomock.Any()).DoAndReturn(func(height uint64) (*pb.Block, error) {
		return blocks[height-1], nil
	}).AnyTimes()

	agencyData, err := ioutil.ReadFile("testdata/agency.cert")
	require.Nil(t, err)

	nodeData, err := ioutil.ReadFile("testdata/node.cert")
	require.Nil(t, err)

	caData, err := ioutil.ReadFile("testdata/ca.cert")
	require.Nil(t, err)

	cert, err := cert.ParseCert(caData)
	require.Nil(t, err)

	for i := 0; i < peerCnt; i++ {
		repo := &repo.Repo{
			Key: &repo.Key{},
			NetworkConfig: &repo.NetworkConfig{
				N:  uint64(peerCnt),
				ID: uint64(i + 1),
			},
			Certs: &repo.Certs{
				NodeCertData:   nodeData,
				AgencyCertData: agencyData,
				CACert:         cert,
			},
			Config: &repo.Config{
				Ping: repo.Ping{
					Enable:   true,
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

		swarm, err := peermgr.New(repo, log.NewWithModule(fmt.Sprintf("swarm%d", i)), mockLedger)
		require.Nil(t, err)
		err = swarm.Start()
		require.Nil(t, err)
		swarms = append(swarms, swarm)
	}
	return swarms
}

func genBlocks(count int) []*pb.Block {
	blocks := make([]*pb.Block, 0, count)
	for height := 1; height <= count; height++ {
		block := &pb.Block{}
		if height == 1 {
			block.BlockHeader = &pb.BlockHeader{
				Number:      1,
				StateRoot:   types.NewHashByStr("0xc30B6E0ad5327fc8548f4BaFab3271cA6a5bD92f084095958c84970165bfA6E7"),
				TxRoot:      nil,
				ReceiptRoot: nil,
				ParentHash:  nil,
				Timestamp:   0,
				Version:     nil,
			}
			block.BlockHash = types.NewHashByStr("0xbC1C6897f97782F3161492d5CcfBE0691502f15894A0b2f2f40069C995E33cCB")
		} else {
			block.BlockHeader = &pb.BlockHeader{
				Number:      uint64(height),
				StateRoot:   blocks[len(blocks)-1].BlockHeader.StateRoot,
				TxRoot:      nil,
				ReceiptRoot: nil,
				ParentHash:  blocks[len(blocks)-1].BlockHash,
				Timestamp:   0,
				Version:     nil,
			}
			block.BlockHash = block.Hash()
		}
		blocks = append(blocks, block)
	}
	return blocks
}
