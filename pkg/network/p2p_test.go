package network

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/meshplus/bitxhub/pkg/network/pb"
	"github.com/stretchr/testify/assert"
)

const (
	protocolID protocol.ID = "/test/1.0.0" // magic protocol
)

func TestP2P_Connect(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6001)
	p2, addr2 := generateNetwork(t, 6002)

	err := p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)
}

func TestP2p_ConnectWithNullIDStore(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6003)
	p2, addr2 := generateNetwork(t, 6004)

	err := p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)
}

func TestP2P_Send(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6005)
	p2, addr2 := generateNetwork(t, 6006)

	msg := []byte("hello")

	ch := make(chan struct{})

	p2.SetMessageHandler(func(s network.Stream, data []byte) {
		assert.EqualValues(t, msg, data)
		close(ch)
	})

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)

	err = p1.AsyncSend(addr2, &pb.Message{Data: msg})
	assert.Nil(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ch:
		return
	case <-ctx.Done():
		assert.Error(t, fmt.Errorf("timeout"))
		return
	}
}

func TestP2p_MultiSend(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6007)
	p2, addr2 := generateNetwork(t, 6008)

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)

	N := 50
	msg := []byte("hello")
	count := 0
	ch := make(chan struct{})

	p2.SetMessageHandler(func(s network.Stream, data []byte) {
		assert.EqualValues(t, msg, data)
		count++
		if count == N {
			close(ch)
			return
		}
	})

	go func() {
		for i := 0; i < N; i++ {
			time.Sleep(200 * time.Microsecond)
			err = p1.AsyncSend(addr2, &pb.Message{Data: msg})
			assert.Nil(t, err)
		}

	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ch:
		return
	case <-ctx.Done():
		assert.Error(t, fmt.Errorf("timeout"))
	}
}

func generateNetwork(t *testing.T, port int) (Network, *peer.AddrInfo) {
	privKey, pubKey, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	assert.Nil(t, err)

	pid1, err := peer.IDFromPublicKey(pubKey)
	assert.Nil(t, err)
	addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)
	maddr := fmt.Sprintf("%s/p2p/%s", addr, pid1)
	p2p, err := New(
		WithLocalAddr(addr),
		WithPrivateKey(privKey),
		WithProtocolID(protocolID),
	)
	assert.Nil(t, err)

	info, err := AddrToPeerInfo(maddr)
	assert.Nil(t, err)

	return p2p, info
}
