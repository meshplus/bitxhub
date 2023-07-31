package peermgr

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/axiomesh/axiom-kit/crypto"
	"github.com/axiomesh/axiom-kit/crypto/asym"
	"github.com/axiomesh/axiom-kit/crypto/asym/ecdsa"
	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/internal/repo"
	"github.com/axiomesh/eth-kit/ledger/mock_ledger"
	"github.com/golang/mock/gomock"
	crypto2 "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		pid, err := peer.IDFromPublicKey(libp2pKey.GetPublic())
		require.Nil(t, err)

		peer := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/", port)
		peers = append(peers, peer)
		pids = append(pids, pid.String())
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

func NewSwarms(t *testing.T, peerCnt int) []*Swarm {
	var swarms []*Swarm
	nodeKeys, privKeys, addrs, pids, accounts := genKeysAndConfig(t, peerCnt)
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	chainLedger.EXPECT().GetBlock(gomock.Any()).Return(&types.Block{
		BlockHeader: &types.BlockHeader{
			Number: 1,
		},
	}, nil).AnyTimes()

	chainLedger.EXPECT().GetBlockSign(gomock.Any()).Return([]byte("sign"), nil).AnyTimes()
	chainLedger.EXPECT().GetTransaction(gomock.Any()).Return(&types.Transaction{}, nil).AnyTimes()
	stateLedger.EXPECT().Copy().Return(stateLedger).AnyTimes()
	stateLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).DoAndReturn(func(addr *types.Address, key []byte) (bool, []byte) {
		return false, nil
	}).AnyTimes()

	for i := 0; i < peerCnt; i++ {
		repo := &repo.Repo{
			Key: &repo.Key{},
			NetworkConfig: &repo.NetworkConfig{
				N:  uint64(peerCnt),
				ID: uint64(i + 1),
			},

			Config: &repo.Config{
				RepoRoot: "testdata",
				Ping: repo.Ping{
					Enable:   true,
					Duration: 2 * time.Second,
				},
				P2pLimit: repo.P2pLimiter{
					Limit: 5,
					Burst: 5,
				},
			},
		}

		idx := strings.LastIndex(addrs[i], "/p2p/")
		local := addrs[i][:idx]
		repo.NetworkConfig.LocalAddr = local
		repo.Key.Libp2pPrivKey = nodeKeys[i]
		repo.Key.PrivKey = privKeys[i]
		repo.NetworkConfig.Nodes = peers(uint64(i), addrs, pids, accounts)

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
	_, err = swarms[0].Send(uint64(2), msg)
	require.NotNil(t, err)
	msg.Type = 100
	_, err = swarms[0].Send(uint64(2), msg)
	require.NotNil(t, err)
	for i := 0; i < len(swarms); i++ {
		err = swarms[i].Stop()
		require.Nil(t, err)
	}
}

func TestSwarm_PushTxs(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}
	requests := [][]byte{generateEthTx(t)}
	pushMsg := &pb.BytesSlice{
		Slice: requests,
	}
	pushData, err := pushMsg.MarshalVT()
	require.Nil(t, err)
	msg := &pb.Message{
		Type: pb.Message_PUSH_TXS,
		Data: pushData,
	}
	orderMsgCh := make(chan OrderMessageEvent)
	orderMsgSub := swarms[2].SubscribeOrderMessage(orderMsgCh)

	defer func() {
		orderMsgSub.Unsubscribe()
		close(orderMsgCh)
	}()

	err = retry.Retry(func(attempt uint) error {
		err = swarms[0].AsyncSend(uint64(3), msg)
		if err != nil {
			swarms[0].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	msgData := <-orderMsgCh
	require.True(t, msgData.IsTxsFromRemote)
}

func generateEthTx(t *testing.T) []byte {
	tx := types.Transaction{
		Inner: &types.DynamicFeeTx{
			Nonce: 0,
		},
		Time: time.Time{},
	}
	raw, err := tx.RbftMarshal()
	require.Nil(t, err)
	return raw
}

func TestMessage_FETCH_P2P_PUBKEY(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	msg := &pb.Message{
		Type: pb.Message_FETCH_P2P_PUBKEY,
	}
	var res *pb.Message
	var err error
	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[0].Send(uint64(2), msg)
		if err != nil {
			swarms[0].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_FETCH_P2P_PUBKEY_ACK, res.Type)
}

func stopSwarms(t *testing.T, swarms []*Swarm) error {
	for _, swarm := range swarms {
		err := swarm.Stop()
		assert.Nil(t, err)
	}
	return nil
}

func TestSwarm_Gater(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}
	gater := newConnectionGater(swarms[0].logger, swarms[0].ledger)
	require.True(t, gater.InterceptPeerDial(peer.ID("1")))
	require.True(t, gater.InterceptAddrDial("1", swarms[1].multiAddrs[1].Addrs[0]))
	require.True(t, gater.InterceptAccept(nil))
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
		res, err = swarms[0].Send(uint64(2), msg)
		if err != nil {
			swarms[0].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_GET_BLOCK_ACK, res.Type)
	var block types.Block
	err = block.Unmarshal(res.Data)
	require.Nil(t, err)
	require.Equal(t, uint64(1), block.BlockHeader.Number)

	req := pb.GetBlocksRequest{
		Start: 1,
		End:   1,
	}
	data, err := req.MarshalVT()
	require.Nil(t, err)

	fetchBlocksMsg := &pb.Message{
		Type: pb.Message_GET_BLOCKS,
		Data: data,
	}
	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[2].Send(uint64(1), fetchBlocksMsg)
		if err != nil {
			swarms[2].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_GET_BLOCKS_ACK, res.Type)
	var getBlocksRes pb.GetBlocksResponse
	err = getBlocksRes.UnmarshalVT(res.Data)
	require.Nil(t, err)
	require.Equal(t, 1, len(getBlocksRes.Blocks))

	getBlockHeadersReq := pb.GetBlockHeadersRequest{
		Start: 1,
		End:   1,
	}
	data, err = getBlockHeadersReq.MarshalVT()
	require.Nil(t, err)

	fetchBlockHeadersMsg := &pb.Message{
		Type: pb.Message_GET_BLOCK_HEADERS,
		Data: data,
	}
	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[2].Send(uint64(4), fetchBlockHeadersMsg)
		if err != nil {
			swarms[2].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_GET_BLOCK_HEADERS_ACK, res.Type)

	var getBlockHeaderssRes pb.GetBlockHeadersResponse
	err = getBlockHeaderssRes.UnmarshalVT(res.Data)
	require.Nil(t, err)
	require.Equal(t, 1, len(getBlockHeaderssRes.BlockHeaders))

	fetchBlockSignMsg := &pb.Message{
		Type: pb.Message_FETCH_BLOCK_SIGN,
		Data: []byte("1"),
	}

	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[1].Send(uint64(3), fetchBlockSignMsg)
		if err != nil {
			swarms[1].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_FETCH_BLOCK_SIGN_ACK, res.Type)
	require.NotNil(t, res.Data)
}

func TestSwarm_AsyncSend(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	orderMsgCh := make(chan OrderMessageEvent)
	orderMsgSub := swarms[2].SubscribeOrderMessage(orderMsgCh)

	defer orderMsgSub.Unsubscribe()

	msg := &pb.Message{
		Type: pb.Message_CONSENSUS,
		Data: []byte("1"),
	}
	var err error
	err = retry.Retry(func(attempt uint) error {
		err = swarms[0].AsyncSend(uint64(3), msg)
		if err != nil {
			swarms[0].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)

	require.NotNil(t, <-orderMsgCh)
}
