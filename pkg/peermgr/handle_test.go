package peermgr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/golang/mock/gomock"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/stretchr/testify/require"
)

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

func NewSwarms(t *testing.T, peerCnt int) []*Swarm {
	var swarms []*Swarm
	nodeKeys, privKeys, addrs, ids := genKeysAndConfig(t, peerCnt)
	mockCtl := gomock.NewController(t)
	mockLedger := mock_ledger.NewMockLedger(mockCtl)

	mockLedger.EXPECT().GetBlock(gomock.Any()).Return(&pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number: 1,
		},
	}, nil).AnyTimes()

	aer := contracts.AssetExchangeRecord{
		Status: 0,
	}

	data, err := json.Marshal(aer)
	require.Nil(t, err)

	mockLedger.EXPECT().GetBlockSign(gomock.Any()).Return([]byte("sign"), nil).AnyTimes()
	mockLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).Return(true, data).AnyTimes()

	agencyData, err := ioutil.ReadFile("testdata/agency.cert")
	require.Nil(t, err)

	nodeData, err := ioutil.ReadFile("testdata/node.cert")
	require.Nil(t, err)

	caData, err := ioutil.ReadFile("testdata/ca.cert")
	require.Nil(t, err)

	cert, err := libp2pcert.ParseCert(caData)
	require.Nil(t, err)

	for i := 0; i < peerCnt; i++ {
		repo := &repo.Repo{
			Key: &repo.Key{},
			NetworkConfig: &repo.NetworkConfig{
				N:  uint64(peerCnt),
				ID: uint64(i + 1),
			},
			Certs: &libp2pcert.Certs{
				NodeCertData:   nodeData,
				AgencyCertData: agencyData,
				CACert:         cert,
			},
			Config: &repo.Config{
				RepoRoot: "testdata",
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

		swarm, err := New(repo, log.NewWithModule(fmt.Sprintf("swarm%d", i)), mockLedger)
		require.Nil(t, err)
		err = swarm.Start()
		require.Nil(t, err)
		swarms = append(swarms, swarm)
	}
	return swarms
}

func TestSwarm_GetBlockPack(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	msg := &pb.Message{
		Type: pb.Message_GET_BLOCK,
		Data: []byte("aaa"),
	}
	var err error
	_, err = swarms[0].Send(2, msg)
	require.NotNil(t, err)
	msg.Type = 100
	_, err = swarms[0].Send(2, msg)
	require.NotNil(t, err)
	for i := 0; i < len(swarms); i++ {
		err = swarms[i].Stop()
		require.Nil(t, err)
	}
}

func stopSwarms(t *testing.T, swarms []*Swarm) error {
	for _, swarm := range swarms {
		err := swarm.Stop()
		assert.Nil(t, err)
	}
	return nil
}

func TestSwarm_FetchCert(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	msg := &pb.Message{
		Type: pb.Message_FETCH_CERT,
	}
	var res *pb.Message
	var err error
	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[0].Send(2, msg)
		if err != nil {
			swarms[0].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_FETCH_CERT_ACK, res.Type)
}

func TestSwarm_CheckMasterPier(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	msg := &pb.Message{
		Type: pb.Message_CHECK_MASTER_PIER,
		Data: []byte("0x111111122222222333333333"),
	}
	res, err := swarms[0].Send(2, msg)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "wait msg timeout")
	require.Nil(t, res)

	pierName := "0x2222233333444444"
	piers2 := newPiers()
	piers1 := newPiers()

	_, err = piers2.pierMap.setMaster(pierName, "pier-index", 300)
	require.Nil(t, err)

	swarms[0].piers = piers1
	swarms[0].piers.pierChan.newChan(pierName)
	swarms[1].piers = piers2
	msg.Data = []byte(pierName)
	swarms[0].Send(2, msg)
	time.Sleep(500 * time.Millisecond)
	require.NotNil(t, swarms[0].piers.pierChan.checkAddress(pierName))
}

func TestSwarm_Send(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	msg := &pb.Message{
		Type: pb.Message_GET_BLOCK,
		Data: []byte("1"),
	}
	var res *pb.Message
	var err error
	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[0].Send(2, msg)
		if err != nil {
			swarms[0].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_GET_BLOCK_ACK, res.Type)
	var block pb.Block
	err = block.Unmarshal(res.Data)
	require.Nil(t, err)
	require.Equal(t, uint64(1), block.BlockHeader.Number)

	req := pb.GetBlocksRequest{
		Start: 1,
		End:   1,
	}
	data, err := req.Marshal()
	require.Nil(t, err)

	fetchBlocksMsg := &pb.Message{
		Type: pb.Message_GET_BLOCKS,
		Data: data,
	}
	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[2].Send(1, fetchBlocksMsg)
		if err != nil {
			swarms[2].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_GET_BLOCKS_ACK, res.Type)
	var getBlocksRes pb.GetBlocksResponse
	err = getBlocksRes.Unmarshal(res.Data)
	require.Nil(t, err)
	require.Equal(t, 1, len(getBlocksRes.Blocks))

	getBlockHeadersReq := pb.GetBlockHeadersRequest{
		Start: 1,
		End:   1,
	}
	data, err = getBlockHeadersReq.Marshal()
	require.Nil(t, err)

	fetchBlockHeadersMsg := &pb.Message{
		Type: pb.Message_GET_BLOCK_HEADERS,
		Data: data,
	}
	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[2].Send(4, fetchBlockHeadersMsg)
		if err != nil {
			swarms[2].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_GET_BLOCK_HEADERS_ACK, res.Type)

	var getBlockHeaderssRes pb.GetBlockHeadersResponse
	err = getBlockHeaderssRes.Unmarshal(res.Data)
	require.Nil(t, err)
	require.Equal(t, 1, len(getBlockHeaderssRes.BlockHeaders))

	fetchBlockSignMsg := &pb.Message{
		Type: pb.Message_FETCH_BLOCK_SIGN,
		Data: []byte("1"),
	}

	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[1].Send(3, fetchBlockSignMsg)
		if err != nil {
			swarms[1].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_FETCH_BLOCK_SIGN_ACK, res.Type)
	require.NotNil(t, res.Data)

	fetchAESMsg := &pb.Message{
		Type: pb.Message_FETCH_ASSET_EXCHANEG_SIGN,
		Data: []byte("1"),
	}

	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[2].Send(4, fetchAESMsg)
		if err != nil {
			swarms[2].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_FETCH_ASSET_EXCHANGE_SIGN_ACK, res.Type)
	require.NotNil(t, res.Data)

	fetchIBTPSignMsg := &pb.Message{
		Type: pb.Message_FETCH_IBTP_SIGN,
		Data: []byte("1"),
	}

	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[3].Send(1, fetchIBTPSignMsg)
		if err != nil {
			swarms[1].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_FETCH_IBTP_SIGN_ACK, res.Type)
	require.NotNil(t, res.Data)
}

//func TestSwarm_AsyncSend(t *testing.T) {
//	peerCnt := 4
//	swarms := NewSwarms(t, peerCnt)
//
//	for swarms[0].CountConnectedPeers() != 3 {
//		time.Sleep(100 * time.Millisecond)
//	}
//
//	orderMsgCh := make(chan events.OrderMessageEvent)
//	orderMsgSub := swarms[2].SubscribeOrderMessage(orderMsgCh)
//
//	defer orderMsgSub.Unsubscribe()
//
//	msg := &pb.Message{
//		Type: pb.Message_CONSENSUS,
//		Data: []byte("1"),
//	}
//	var err error
//	err = retry.Retry(func(attempt uint) error {
//		err = swarms[0].AsyncSend(3, msg)
//		if err != nil {
//			swarms[0].logger.Errorf(err.Error())
//			return err
//		}
//		return nil
//	}, strategy.Wait(50*time.Millisecond))
//	require.Nil(t, err)
//
//	require.NotNil(t, <-orderMsgCh)
//}
