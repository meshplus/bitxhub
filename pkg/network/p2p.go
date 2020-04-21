package network

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/meshplus/bitxhub/pkg/network/pb"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

var _ Network = (*P2P)(nil)

var (
	connectTimeout = 10 * time.Second
	sendTimeout    = 5 * time.Second
	waitTimeout    = 5 * time.Second
)

type P2P struct {
	config          *Config
	host            host.Host // manage all connections
	streamMng       *streamMgr
	connectCallback ConnectCallback
	handleMessage   MessageHandler
	logger          logrus.FieldLogger

	ctx    context.Context
	cancel context.CancelFunc
}

func New(opts ...Option) (*P2P, error) {
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
		streamMng: newStreamMng(ctx, h, config.protocolID),
		logger:    config.logger,
		ctx:       ctx,
		cancel:    cancel,
	}

	return p2p, nil
}

// Start start the network service.
func (p2p *P2P) Start() error {
	p2p.host.SetStreamHandler(p2p.config.protocolID, p2p.handleNewStream)

	return nil
}

// Connect peer.
func (p2p *P2P) Connect(addr *peer.AddrInfo) error {
	ctx, cancel := context.WithTimeout(p2p.ctx, connectTimeout)
	defer cancel()

	if err := p2p.host.Connect(ctx, *addr); err != nil {
		return err
	}

	p2p.host.Peerstore().AddAddrs(addr.ID, addr.Addrs, peerstore.PermanentAddrTTL)

	if p2p.connectCallback != nil {
		if err := p2p.connectCallback(addr); err != nil {
			return err
		}
	}

	return nil
}

func (p2p *P2P) SetConnectCallback(callback ConnectCallback) {
	p2p.connectCallback = callback
}

func (p2p *P2P) SetMessageHandler(handler MessageHandler) {
	p2p.handleMessage = handler
}

// AsyncSend message to peer with specific id.
func (p2p *P2P) AsyncSend(addr *peer.AddrInfo, msg *pb.Message) error {
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

func (p2p *P2P) SendWithStream(s network.Stream, msg *pb.Message) error {
	return p2p.send(s, msg)
}

func (p2p *P2P) Send(addr *peer.AddrInfo, msg *pb.Message) (*pb.Message, error) {
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
		return nil, fmt.Errorf("sync send msg to node[%s] timeout", addr.ID)
	}

	return recvMsg, nil
}

func (p2p *P2P) Broadcast(ids []*peer.AddrInfo, msg *pb.Message) error {
	for _, id := range ids {
		if err := p2p.AsyncSend(id, msg); err != nil {
			p2p.logger.WithFields(logrus.Fields{
				"error": err,
				"id":    id,
			}).Error("Async Send message")
			continue
		}
	}

	return nil
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

func (p2p *P2P) Disconnect(addr *peer.AddrInfo) error {
	return p2p.host.Network().ClosePeer(addr.ID)
}
