package peermgr

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/connmgr"
	p2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom"
	"github.com/axiomesh/axiom-kit/types/pb"
	network "github.com/axiomesh/axiom-p2p"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/pkg/repo"
)

const (
	protocolID string = "/axiom/1.0.0" // magic protocol
)

var _ PeerManager = (*Swarm)(nil)

type Swarm struct {
	repo           *repo.Repo
	ledger         *ledger.Ledger
	p2p            network.Network
	logger         logrus.FieldLogger
	connectedPeers sync.Map
	enablePing     bool
	pingTimeout    time.Duration
	pingC          chan *repo.Ping
	ctx            context.Context
	cancel         context.CancelFunc
	gater          connmgr.ConnectionGater
	network.PipeManager
}

func New(repoConfig *repo.Repo, logger logrus.FieldLogger, ledger *ledger.Ledger) (*Swarm, error) {
	ctx, cancel := context.WithCancel(context.Background())
	swarm := &Swarm{repo: repoConfig, logger: logger, ledger: ledger, ctx: ctx, cancel: cancel}
	if err := swarm.init(); err != nil {
		return nil, err
	}

	return swarm, nil
}

func (swarm *Swarm) init() error {
	// init peers with ips and hosts
	bootstrap := make([]string, 0)
	for _, a := range swarm.repo.Config.Genesis.EpochInfo.P2PBootstrapNodeAddresses {
		if !strings.Contains(a, swarm.repo.P2PID) {
			bootstrap = append(bootstrap, a)
		}
	}

	var securityType network.SecurityType
	switch swarm.repo.Config.P2P.Security {
	case repo.P2PSecurityTLS:
		securityType = network.SecurityTLS
	case repo.P2PSecurityNoise:
		securityType = network.SecurityNoise
	default:
		securityType = network.SecurityNoise
	}

	var pipeBroadcastType network.PipeBroadcastType
	switch swarm.repo.Config.P2P.Pipe.BroadcastType {
	case repo.P2PPipeBroadcastSimple:
		pipeBroadcastType = network.PipeBroadcastSimple
	case repo.P2PPipeBroadcastGossip:
		pipeBroadcastType = network.PipeBroadcastGossip
	case repo.P2PPipeBroadcastFlood:
		pipeBroadcastType = network.PipeBroadcastFloodSub
	default:
		return fmt.Errorf("unsupported p2p pipe broadcast type: %v", swarm.repo.Config.P2P.Pipe.BroadcastType)
	}

	protocolIDWithVersion := fmt.Sprintf("%s-%x", protocolID, sha256.Sum256([]byte(axiom.VersionSecret)))
	gater := newConnectionGater(swarm.logger, swarm.ledger)
	opts := []network.Option{
		network.WithLocalAddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", swarm.repo.Config.Port.P2P)),
		network.WithPrivateKey(swarm.repo.P2PKey),
		network.WithProtocolID(protocolIDWithVersion),
		network.WithLogger(swarm.logger),
		network.WithTimeout(10*time.Second, swarm.repo.Config.P2P.SendTimeout.ToDuration(), swarm.repo.Config.P2P.ReadTimeout.ToDuration()),
		network.WithSecurity(securityType),
		network.WithPipeBroadcastType(pipeBroadcastType),
		network.WithPipeReceiveMsgCacheSize(swarm.repo.Config.P2P.Pipe.ReceiveMsgCacheSize),
		// enable discovery
		network.WithBootstrap(bootstrap),
		network.WithConnectionGater(gater),
	}

	p2p, err := network.New(swarm.ctx, opts...)
	if err != nil {
		return fmt.Errorf("create p2p: %w", err)
	}
	p2p.SetConnectCallback(swarm.onConnected)
	p2p.SetDisconnectCallback(swarm.onDisconnected)
	swarm.p2p = p2p
	swarm.enablePing = swarm.repo.Config.P2P.Ping.Enable
	swarm.pingTimeout = swarm.repo.Config.P2P.Ping.Duration.ToDuration()
	swarm.pingC = make(chan *repo.Ping)
	swarm.connectedPeers = sync.Map{}
	swarm.gater = gater
	swarm.PipeManager = p2p
	return nil
}

func (swarm *Swarm) Start() error {
	swarm.p2p.SetMessageHandler(swarm.handleMessage)

	if err := swarm.p2p.Start(); err != nil {
		return fmt.Errorf("start p2p failed: %w", err)
	}

	go swarm.Ping()

	return nil
}

func (swarm *Swarm) Stop() error {
	swarm.cancel()
	return swarm.p2p.Stop()
}

func (swarm *Swarm) onConnected(net p2pnetwork.Network, conn p2pnetwork.Conn) error {
	peerID := conn.RemotePeer().String()
	swarm.connectedPeers.Store(peerID, nil)

	return nil
}

func (swarm *Swarm) onDisconnected(peerID string) {
	swarm.connectedPeers.Delete(peerID)
}

func (swarm *Swarm) Ping() {
	if !swarm.enablePing {
		swarm.pingTimeout = math.MaxInt64
	}
	ticker := time.NewTicker(swarm.pingTimeout)
	for {
		select {
		case <-ticker.C:
			fields := logrus.Fields{}
			swarm.connectedPeers.Range(func(key, value any) bool {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				pingCh, err := swarm.p2p.Ping(ctx, key.(string))
				if err != nil {
					return true
				}
				select {
				case res := <-pingCh:
					fields[fmt.Sprintf("%v", key)] = res.RTT
				case <-time.After(time.Second * 5):
					swarm.logger.Errorf("ping to node v timeout", key)
				}
				return true
			})
			swarm.logger.WithFields(fields).Info("ping time")
		case pingConfig := <-swarm.pingC:
			swarm.enablePing = pingConfig.Enable
			swarm.pingTimeout = pingConfig.Duration.ToDuration()
			if !swarm.enablePing {
				swarm.pingTimeout = math.MaxInt64
			}
			ticker.Stop()
			ticker = time.NewTicker(swarm.pingTimeout)
		case <-swarm.ctx.Done():
			return
		}
	}
}

func (swarm *Swarm) SendWithStream(s network.Stream, msg *pb.Message) error {
	data, err := msg.MarshalVT()
	if err != nil {
		return fmt.Errorf("marshal message error: %w", err)
	}

	return s.AsyncSend(data)
}

func (swarm *Swarm) Send(to string, msg *pb.Message) (*pb.Message, error) {
	data, err := msg.MarshalVT()
	if err != nil {
		return nil, fmt.Errorf("marshal message error: %w", err)
	}

	ret, err := swarm.p2p.Send(to, data)
	if err != nil {
		return nil, fmt.Errorf("sync send: %w", err)
	}

	m := &pb.Message{}
	if err := m.UnmarshalVT(ret); err != nil {
		return nil, fmt.Errorf("unmarshal message error: %w", err)
	}

	return m, nil
}

func (swarm *Swarm) Peers() []peer.AddrInfo {
	return swarm.p2p.GetPeers()
}

func (swarm *Swarm) CountConnectedPeers() uint64 {
	var counter uint64
	swarm.connectedPeers.Range(func(k, v any) bool {
		counter++
		return true
	})
	return counter
}

func (swarm *Swarm) PeerID() string {
	return swarm.p2p.PeerID()
}
