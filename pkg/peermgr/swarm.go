package peermgr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum/event"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/cert"
	network "github.com/meshplus/go-lightp2p"
	"github.com/sirupsen/logrus"
)

const (
	protocolID protocol.ID = "/B1txHu6/1.0.0" // magic protocol
)

type Swarm struct {
	repo           *repo.Repo
	p2p            network.Network
	logger         logrus.FieldLogger
	peers          map[uint64]*peer.AddrInfo
	connectedPeers sync.Map
	ledger         ledger.Ledger

	orderMessageFeed event.Feed

	ctx    context.Context
	cancel context.CancelFunc
}

func New(repo *repo.Repo, logger logrus.FieldLogger, ledger ledger.Ledger) (*Swarm, error) {
	var protocolIDs = []string{string(protocolID)}
	p2p, err := network.New(
		network.WithLocalAddr(repo.NetworkConfig.LocalAddr),
		network.WithPrivateKey(repo.Key.Libp2pPrivKey),
		network.WithProtocolIDs(protocolIDs),
		network.WithLogger(logger),
	)

	if err != nil {
		return nil, fmt.Errorf("create p2p: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Swarm{
		repo:           repo,
		p2p:            p2p,
		logger:         logger,
		ledger:         ledger,
		peers:          repo.NetworkConfig.OtherNodes,
		connectedPeers: sync.Map{},
		ctx:            ctx,
		cancel:         cancel,
	}, nil
}

func (swarm *Swarm) Start() error {
	swarm.p2p.SetMessageHandler(swarm.handleMessage)

	if err := swarm.p2p.Start(); err != nil {
		return err
	}

	for id, addr := range swarm.peers {
		go func(id uint64, addr *peer.AddrInfo) {
			if err := retry.Retry(func(attempt uint) error {
				if err := swarm.p2p.Connect(*addr); err != nil {
					swarm.logger.WithFields(logrus.Fields{
						"node":  id,
						"error": err,
					}).Error("Connect failed")
					return err
				}

				if err := swarm.verifyCert(id); err != nil {
					if attempt != 0 && attempt%5 == 0 {
						swarm.logger.WithFields(logrus.Fields{
							"node":  id,
							"error": err,
						}).Error("Verify cert")
					}

					return err
				}

				swarm.logger.WithFields(logrus.Fields{
					"node": id,
				}).Info("Connect successfully")

				swarm.connectedPeers.Store(id, addr)

				return nil
			},
				strategy.Wait(1*time.Second),
			); err != nil {
				swarm.logger.Error(err)
			}
		}(id, addr)
	}

	return nil
}

func (swarm *Swarm) Stop() error {
	swarm.cancel()

	return nil
}

func (swarm *Swarm) AsyncSend(id uint64, msg *pb.Message) error {
	if err := swarm.checkID(id); err != nil {
		return fmt.Errorf("p2p send: %w", err)
	}

	data, err := msg.Marshal()
	if err != nil {
		return err
	}
	addr := swarm.peers[id].ID.String()
	return swarm.p2p.AsyncSend(addr, data)
}

func (swarm *Swarm) SendWithStream(s network.Stream, msg *pb.Message) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	return s.AsyncSend(data)
}

func (swarm *Swarm) Send(id uint64, msg *pb.Message) (*pb.Message, error) {
	if err := swarm.checkID(id); err != nil {
		return nil, fmt.Errorf("check id: %w", err)
	}

	data, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	addr := swarm.peers[id].ID.String()
	ret, err := swarm.p2p.Send(addr, data)
	if err != nil {
		return nil, fmt.Errorf("sync send: %w", err)
	}

	m := &pb.Message{}
	if err := m.Unmarshal(ret); err != nil {
		return nil, err
	}

	return m, nil
}

func (swarm *Swarm) Broadcast(msg *pb.Message) error {
	addrs := make([]string, 0, len(swarm.peers))
	for _, addr := range swarm.peers {
		addrs = append(addrs, addr.ID.String())
	}

	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	return swarm.p2p.Broadcast(addrs, data)
}

func (swarm *Swarm) Peers() map[uint64]*peer.AddrInfo {
	m := make(map[uint64]*peer.AddrInfo)
	for id, addr := range swarm.peers {
		m[id] = addr
	}

	return m
}

func (swarm *Swarm) OtherPeers() map[uint64]*peer.AddrInfo {
	m := swarm.Peers()
	delete(m, swarm.repo.NetworkConfig.ID)

	return m
}

func (swarm *Swarm) SubscribeOrderMessage(ch chan<- events.OrderMessageEvent) event.Subscription {
	return swarm.orderMessageFeed.Subscribe(ch)
}

func (swarm *Swarm) verifyCert(id uint64) error {
	if err := swarm.checkID(id); err != nil {
		return fmt.Errorf("check id: %w", err)
	}

	msg := &pb.Message{
		Type: pb.Message_FETCH_CERT,
	}

	ret, err := swarm.Send(id, msg)
	if err != nil {
		return fmt.Errorf("sync send: %w", err)
	}

	certs := &model.CertsMessage{}
	if err := certs.Unmarshal(ret.Data); err != nil {
		return fmt.Errorf("unmarshal certs: %w", err)
	}

	nodeCert, err := cert.ParseCert(certs.NodeCert)
	if err != nil {
		return fmt.Errorf("parse node cert: %w", err)
	}

	agencyCert, err := cert.ParseCert(certs.AgencyCert)
	if err != nil {
		return fmt.Errorf("parse agency cert: %w", err)
	}

	if err := verifyCerts(nodeCert, agencyCert, swarm.repo.Certs.CACert); err != nil {
		return fmt.Errorf("verify certs: %w", err)
	}

	err = swarm.p2p.Disconnect(swarm.peers[id].ID.String())
	if err != nil {
		return fmt.Errorf("disconnect peer: %w", err)
	}

	return nil
}

func (swarm *Swarm) checkID(id uint64) error {
	if swarm.peers[id] == nil {
		return fmt.Errorf("wrong id: %d", id)
	}

	return nil
}
