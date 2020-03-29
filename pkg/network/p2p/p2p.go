package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	net "github.com/meshplus/bitxhub/pkg/network"
	"github.com/meshplus/bitxhub/pkg/network/proto"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ net.Network = (*P2P)(nil)

var ErrorPeerNotFound = errors.New("peer not found")

var (
	connectTimeout = 10 * time.Second
	sendTimeout    = 5 * time.Second
	waitTimeout    = 5 * time.Second
)

type P2P struct {
	config          *Config
	host            host.Host // manage all connections
	recvQ           chan *net.MessageStream
	streamMng       *streamMgr
	connectCallback net.ConnectCallback
	logger          logrus.FieldLogger
	idStore         net.IDStore

	ctx    context.Context
	cancel context.CancelFunc
}

func New(opts ...Option) (net.Network, error) {
	config, err := generateConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	h, err := libp2p.New(ctx,
		libp2p.Identity(config.privKey),
		libp2p.ListenAddrStrings(config.localAddr))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create libp2p: %w", err)
	}

	p2p := &P2P{
		config:    config,
		host:      h,
		idStore:   config.idStore,
		recvQ:     make(chan *net.MessageStream),
		streamMng: newStreamMng(ctx, h, config.protocolID),
		logger:    config.logger,
		ctx:       ctx,
		cancel:    cancel,
	}

	if config.idStore == nil {
		p2p.idStore = NewIDStore()
	}

	return p2p, nil
}

// Start start the network service.
func (p2p *P2P) Start() error {
	p2p.host.SetStreamHandler(p2p.config.protocolID, p2p.handleNewStream)

	return nil
}

// Connect peer.
func (p2p *P2P) Connect(id net.ID) error {
	ctx, cancel := context.WithTimeout(p2p.ctx, connectTimeout)
	defer cancel()

	pid := p2p.idStore.Addr(id)
	if pid == nil {
		return ErrorPeerNotFound
	}

	if err := p2p.host.Connect(ctx, *pid); err != nil {
		return err
	}

	p2p.host.Peerstore().AddAddrs(pid.ID, pid.Addrs, peerstore.PermanentAddrTTL)

	if p2p.connectCallback != nil {
		if err := p2p.connectCallback(id); err != nil {
			return err
		}
	}

	return nil
}

func (p2p *P2P) SetConnectCallback(callback net.ConnectCallback) {
	p2p.connectCallback = callback
}

// Send message to peer with specific id.
func (p2p *P2P) Send(id net.ID, msg *proto.Message) error {
	addr := p2p.idStore.Addr(id)
	if addr == nil {
		return ErrorPeerNotFound
	}

	s, err := p2p.streamMng.get(addr.ID)
	if err != nil {
		return fmt.Errorf("get stream: %w", err)
	}

	if err := p2p.send(s, msg); err != nil {
		p2p.streamMng.remove(addr.ID)
		return err
	}

	return nil
}

func (p2p *P2P) SendWithStream(s network.Stream, msg *proto.Message) error {
	return p2p.send(s, msg)
}

func (p2p *P2P) SyncSend(id net.ID, msg *proto.Message) (*proto.Message, error) {
	addr := p2p.idStore.Addr(id)
	if addr == nil {
		return nil, ErrorPeerNotFound
	}

	s, err := p2p.streamMng.get(addr.ID)
	if err != nil {
		return nil, fmt.Errorf("get stream: %w", err)
	}

	if err := p2p.send(s, msg); err != nil {
		p2p.streamMng.remove(addr.ID)
		return nil, err
	}

	recvMsg := waitMsg(s, waitTimeout)
	if recvMsg == nil {
		return nil, fmt.Errorf("sync send msg to node%d timeout", id)
	}

	return recvMsg, nil
}

func (p2p *P2P) Broadcast(ids []net.ID, msg *proto.Message) error {
	for _, id := range ids {
		if err := p2p.Send(id, msg); err != nil {
			p2p.logger.WithFields(logrus.Fields{
				"error": err,
				"id":    id,
			}).Error("Send message")
			continue
		}
	}

	return nil
}

func (p2p *P2P) Receive() <-chan *net.MessageStream {
	return p2p.recvQ
}

// Stop stop the network service.
func (p2p *P2P) Stop() error {
	p2p.cancel()

	return p2p.host.Close()
}

// AddrToPeerInfo transfer addr to PeerInfo
// addr example: "/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64"
func AddrToPeerInfo(multiAddr string) (*peer.AddrInfo, error) {
	maddr, err := ma.NewMultiaddr(multiAddr)
	if err != nil {
		return nil, err
	}

	return peer.AddrInfoFromP2pAddr(maddr)
}

func (p2p *P2P) Disconnect(id net.ID) error {
	addr := p2p.idStore.Addr(id)
	if addr == nil {
		return ErrorPeerNotFound
	}

	return p2p.host.Network().ClosePeer(addr.ID)
}

func (p2p *P2P) IDStore() net.IDStore {
	return p2p.idStore
}
