package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	net "github.com/meshplus/bitxhub/pkg/network"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
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

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)

	msg := []byte("hello")
	err = p1.AsyncSend(addr2, net.Message(msg))
	assert.Nil(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch := p2.Receive()
	for {
		select {
		case c := <-ch:
			assert.EqualValues(t, msg, c.Message.Data)
			return
		case <-ctx.Done():
			assert.Error(t, fmt.Errorf("timeout"))
			return
		}
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
	ch := p2.Receive()

	go func() {
		for i := 0; i < N; i++ {
			time.Sleep(200 * time.Microsecond)
			err = p1.AsyncSend(addr2, net.Message(msg))
			assert.Nil(t, err)
		}

	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count := 0
	for {
		select {
		case c := <-ch:
			assert.EqualValues(t, msg, c.Message.Data)
		case <-ctx.Done():
			assert.Error(t, fmt.Errorf("timeout"))
		}
		count++
		if count == N {
			return
		}
	}
}

func generateNetwork(t *testing.T, port int) (net.Network, *peer.AddrInfo) {
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
