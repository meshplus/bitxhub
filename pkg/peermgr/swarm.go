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
	"github.com/meshplus/bitxhub-kit/types"
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
	repo    *repo.Repo
	localID uint64
	p2p     network.Network
	logger  logrus.FieldLogger

	routers  map[uint64]*VPInfo // trace the vp nodes
	multiAddrs     map[uint64]*peer.AddrInfo
	connectedPeers sync.Map
	notifiee       *notifiee

	ledger           ledger.Ledger
	orderMessageFeed event.Feed
	enablePing       bool
	pingTimeout      time.Duration

	ctx    context.Context
	cancel context.CancelFunc
}

func New(repoConfig *repo.Repo, logger logrus.FieldLogger, ledger ledger.Ledger) (*Swarm, error) {
	var protocolIDs = []string{string(protocolID)}
	// init peers with ips and hosts
	peers := make([]string, 0)
	for _, addr := range repo.NetworkConfig.Nodes {
		peers = append(peers, addr.Addr)
	}

	validators := make(map[uint64]*VPInfo)
	for i, node := range repo.NetworkConfig.Nodes {
		keyAddr := *types.NewAddressByStr(repo.AllAddresses.Addresses[i])
		IpInfo := repo.NetworkConfig.VpNodes[node.ID]
		vpInfo := &VPInfo{
			KeyAddr: keyAddr.String(),
			IPAddr:  IpInfo.ID.String(),
		}
		validators[node.ID] = vpInfo
	}

	multiAddrs := make(map[uint64]*peer.AddrInfo)
	for _, node := range repo.NetworkConfig.Nodes {
		if node.ID == repo.NetworkConfig.ID {
			continue
		}
		IpInfo := repo.NetworkConfig.VpNodes[node.ID]
		addInfo := &peer.AddrInfo{
			ID:    IpInfo.ID,
			Addrs: IpInfo.Addrs,
		}
		multiAddrs[node.ID] = addInfo
	}
	notifiee := newNotifiee(validators, logger)
	p2p, err := network.New(
		network.WithLocalAddr(repoConfig.NetworkConfig.LocalAddr),
		network.WithPrivateKey(repoConfig.Key.Libp2pPrivKey),
		network.WithProtocolIDs(protocolIDs),
		network.WithLogger(logger),
		// enable discovery
		network.WithBootstrap(peers),
		network.WithNotify(notifiee),
	)
	if err != nil {
		return nil, fmt.Errorf("create p2p: %w", err)
	}

	peers, err := repoConfig.NetworkConfig.GetPeers()
	if err != nil {
		return nil, fmt.Errorf("get peers:%w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Swarm{
		repo:           repoConfig,
		localID:        repoConfig.NetworkConfig.ID,
		p2p:            p2p,
		logger:         logger,
		ledger:         ledger,
		enablePing:     repo.Config.Ping.Enable,
		pingTimeout:    repo.Config.Ping.Duration,
		routers:        validators,
		multiAddrs:     multiAddrs,
		connectedPeers: sync.Map{},
		notifiee:       notifiee,
		ctx:            ctx,
		cancel:         cancel,
	}, nil
}

func (swarm *Swarm) Start() error {
	swarm.p2p.SetMessageHandler(swarm.handleMessage)

	if err := swarm.p2p.Start(); err != nil {
		return err
	}

	for id, addr := range swarm.multiAddrs {
		go func(id uint64, addr *peer.AddrInfo) {
			if err := retry.Retry(func(attempt uint) error {
				if err := swarm.p2p.Connect(*addr); err != nil {
					swarm.logger.WithFields(logrus.Fields{
						"node":  id,
						"error": err,
					}).Error("Connect failed")
					return err
				}

				if err := swarm.verifyCertOrDisconnect(id); err != nil {
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

	if swarm.enablePing {
		go swarm.Ping()
	}

	return nil
}

func (swarm *Swarm) Stop() error {
	swarm.cancel()
	return nil
}

func (swarm *Swarm) verifyCertOrDisconnect(id uint64) error {
	if err := swarm.verifyCert(id); err != nil {
		if err = swarm.p2p.Disconnect(swarm.routers[id].IPAddr); err != nil {
			return err
		}
		return err
	}
	return nil
}

func (swarm *Swarm) Ping() {
	ticker := time.NewTicker(swarm.pingTimeout)
	for {
		select {
		case <-ticker.C:
			fields := logrus.Fields{}
			swarm.connectedPeers.Range(func(key, value interface{}) bool {
				info := value.(*peer.AddrInfo)
				pingCh, err := swarm.p2p.Ping(info.ID.String())
				if err != nil {
					return true
				}
				select {
				case res := <-pingCh:
					fields[fmt.Sprintf("%d", key.(uint64))] = res.RTT
				case <-time.After(time.Second * 5):
					swarm.logger.Errorf("ping to node %d timeout", key.(uint64))
				}
				return true
			})
			swarm.logger.WithFields(fields).Info("ping time")
		case <-swarm.ctx.Done():
			return
		}
	}
}

func (swarm *Swarm) AsyncSend(id uint64, msg *pb.Message) error {
	var (
		addr string
		err  error
	)
	if addr, err = swarm.findPeer(id); err != nil {
		return fmt.Errorf("p2p send: %w", err)
	}

	data, err := msg.Marshal()
	if err != nil {
		return err
	}
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
	var (
		addr string
		err  error
	)
	if addr, err = swarm.findPeer(id); err != nil {
		return nil, fmt.Errorf("check id: %w", err)
	}

	data, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

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
	addrs := make([]string, 0, len(swarm.routers))
	for id, addr := range swarm.routers {
		if id == swarm.localID {
			continue
		}
		addrs = append(addrs, addr.IPAddr)
	}

	// if we are in adding node but hasn't finished updateN, new node hash will be temporarily recorded
	// in swarm.notifiee.newPeer.
	if swarm.notifiee.newPeer != "" {
		swarm.logger.Debugf("Broadcast to new peer %s", swarm.notifiee.newPeer)
		addrs = append(addrs, swarm.notifiee.newPeer)
	}

	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	return swarm.p2p.Broadcast(addrs, data)
}

func (swarm *Swarm) Peers() map[uint64]*VPInfo {
	return swarm.notifiee.getPeers()
}

func (swarm *Swarm) OtherPeers() map[uint64]*peer.AddrInfo {
	addrInfos := make(map[uint64]*peer.AddrInfo)
	for id, addr := range swarm.notifiee.getPeers() {
		if id == swarm.localID {
			continue
		}
		addrInfo := &peer.AddrInfo{
			ID: peer.ID(addr.IPAddr),
		}
		addrInfos[id] = addrInfo
	}
	return addrInfos
}

func (swarm *Swarm) SubscribeOrderMessage(ch chan<- events.OrderMessageEvent) event.Subscription {
	return swarm.orderMessageFeed.Subscribe(ch)
}

func (swarm *Swarm) verifyCert(id uint64) error {
	if _, err := swarm.findPeer(id); err != nil {
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

	err = swarm.p2p.Disconnect(swarm.routers[id].IPAddr)
	if err != nil {
		return fmt.Errorf("disconnect peer: %w", err)
	}

	return nil
}

func (swarm *Swarm) findPeer(id uint64) (string, error) {
	if swarm.routers[id] != nil {
		return swarm.routers[id].IPAddr, nil
	}
	newPeerAddr := swarm.notifiee.newPeer
	// new node id should be len(swarm.peers)+1
	if uint64(len(swarm.routers)+1) == id && swarm.notifiee.newPeer != "" {
		return newPeerAddr, nil
	}
	return "", fmt.Errorf("wrong id: %d", id)
}

// todo (YH): persist config and update connectedPeers and multiAddrs info?
func (swarm *Swarm) AddNode(newNodeID uint64, vpInfo *VPInfo) {
	if _, ok := swarm.routers[newNodeID]; ok {
		swarm.logger.Warningf("VP[ID: %d, IpAddr: %s] has already exist in routing table", newNodeID, vpInfo.IPAddr)
		return
	}
	swarm.logger.Infof("Add vp[ID: %d, IpAddr: %s] into routing table", newNodeID, vpInfo.IPAddr)
	swarm.routers[newNodeID] = vpInfo
	for id, p := range swarm.routers {
		swarm.logger.Debugf("=====ID: %d, Addr: %v=====", id, p)
	}
	// update notifiee info
	swarm.notifiee.setPeers(swarm.routers)
	if swarm.notifiee.newPeer == vpInfo.IPAddr {
		swarm.logger.Info("Clear notifiee newPeer info")
		swarm.notifiee.newPeer = ""
	} else {
		swarm.logger.Warningf("Received vpInfo %v, but it doesn't equal to  notifiee newPeer %s", vpInfo, swarm.notifiee.newPeer)
	}
}

func (swarm *Swarm) DelNode(delID uint64) {
	if delID == swarm.localID {
		// deleted node itself will exit the cluster
		swarm.reset()
		_ = swarm.p2p.Stop()
		_ = swarm.Stop()
		return
	}
	var (
		delNode *VPInfo
		ok      bool
	)
	if delNode, ok = swarm.routers[delID]; !ok {
		swarm.logger.Warningf("Can't find vp node %d from routing table ", delID)
		return
	}
	swarm.logger.Infof("Delete node [ID: %d, peerInfo: %v] ", delID, delNode)
	delete(swarm.routers, delID)
	delete(swarm.multiAddrs, delID)
	swarm.connectedPeers.Delete(delID)
	for id, p := range swarm.routers {
		swarm.logger.Debugf("=====ID: %d, Addr: %v=====", id, p)
	}
	// update notifiee info
	swarm.notifiee.setPeers(swarm.routers)
}

func (swarm *Swarm) UpdateRouter(vpInfos map[uint64]*VPInfo, isNew bool) bool {
	swarm.logger.Infof("Update router: %+v", vpInfos)
	swarm.routers = vpInfos
	// check if self is exist in the routing table, and if not, exit the cluster
	var isExist bool
	for id, _ := range vpInfos {
		if id == swarm.localID {
			isExist = true
			break
		}
	}
	if !isExist && !isNew {
		// deleted node itself will exit the cluster
		swarm.reset()
		_ = swarm.p2p.Stop()
		_ = swarm.Stop()
		return true
	}
	// update notifiee info
	swarm.notifiee.setPeers(vpInfos)
	return false
}

func (swarm *Swarm) reset() {
	swarm.routers = nil
	swarm.multiAddrs = nil
	swarm.connectedPeers = sync.Map{}
	swarm.notifiee.setPeers(nil)
}

func (swarm *Swarm) Disconnect(vpInfos map[uint64]*VPInfo) {
	for id, info := range vpInfos {
		if err := swarm.p2p.Disconnect(info.IPAddr); err != nil {
			swarm.logger.Errorf("Disconnect peer %s failed, err: %s", err.Error())
		}
		swarm.logger.Infof("Disconnect peer [ID: %d, Pid: %s]", id, info.IPAddr)
	}
	swarm.logger.Infof("======== NOTE!!! THIS NODE HAS BEEN DELETED!!!")
}
