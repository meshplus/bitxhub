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
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	network "github.com/meshplus/go-lightp2p"
	ma "github.com/multiformats/go-multiaddr"
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

	routers        map[uint64]*pb.VpInfo // trace the vp nodes
	multiAddrs     map[uint64]*peer.AddrInfo
	connectedPeers sync.Map
	notifiee       *notifiee
	piers          *Piers

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
	routers := repoConfig.NetworkConfig.GetVpInfos()
	bootstrap := make([]string, 0)
	for _, p := range routers {
		if p.Id == repoConfig.NetworkConfig.ID {
			continue
		}
		addr := fmt.Sprintf("%s%s", p.Hosts[0], p.Pid)
		bootstrap = append(bootstrap, addr)
	}

	multiAddrs := make(map[uint64]*peer.AddrInfo)
	p2pPeers, _ := repoConfig.NetworkConfig.GetNetworkPeers()
	for id, node := range p2pPeers {
		if id == repoConfig.NetworkConfig.ID {
			continue
		}
		multiAddrs[id] = node
	}

	tpt, err := libp2pcert.New(repoConfig.Key.Libp2pPrivKey, repoConfig.Certs)
	if err != nil {
		return nil, fmt.Errorf("create transport: %w", err)
	}

	notifiee := newNotifiee(routers, logger)

	opts := []network.Option{
		network.WithLocalAddr(repoConfig.NetworkConfig.LocalAddr),
		network.WithPrivateKey(repoConfig.Key.Libp2pPrivKey),
		network.WithProtocolIDs(protocolIDs),
		network.WithLogger(logger),
		// enable discovery
		network.WithBootstrap(bootstrap),
		network.WithNotify(notifiee),
	}

	if repoConfig.Config.Cert.Verify {
		opts = append(opts,
			network.WithTransportId(libp2pcert.ID),
			network.WithTransport(tpt),
		)
	}

	p2p, err := network.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create p2p: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Swarm{
		repo:           repoConfig,
		localID:        repoConfig.NetworkConfig.ID,
		p2p:            p2p,
		logger:         logger,
		ledger:         ledger,
		enablePing:     repoConfig.Config.Ping.Enable,
		pingTimeout:    repoConfig.Config.Ping.Duration,
		routers:        routers,
		multiAddrs:     multiAddrs,
		piers:          newPiers(),
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
				// for restart node, after updating the routing table, some nodes may not exist in routing table
				routers := swarm.notifiee.getPeers()
				if _, ok := routers[id]; !ok {
					swarm.logger.Infof("Can't find node %d from routing table, stopping connect", id)
					return nil
				}
				if err := swarm.p2p.Connect(*addr); err != nil {
					swarm.logger.WithFields(logrus.Fields{
						"node":  id,
						"error": err,
					}).Error("Connect failed")
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
	return swarm.p2p.Stop()
}

func (swarm *Swarm) Ping() {
	ticker := time.NewTicker(swarm.pingTimeout)
	for {
		select {
		case <-ticker.C:
			fields := logrus.Fields{}
			swarm.connectedPeers.Range(func(key, value interface{}) bool {
				info := value.(*peer.AddrInfo)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				pingCh, err := swarm.p2p.Ping(ctx, info.ID.String())
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
	for _, router := range swarm.routers {
		if router.Id == swarm.localID {
			continue
		}
		addrs = append(addrs, router.Pid)
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

func (swarm *Swarm) Peers() map[uint64]*pb.VpInfo {
	return swarm.notifiee.getPeers()
}

func (swarm *Swarm) OtherPeers() map[uint64]*peer.AddrInfo {
	addrInfos := make(map[uint64]*peer.AddrInfo)
	for _, node := range swarm.notifiee.getPeers() {
		if node.Id == swarm.localID {
			continue
		}
		addrInfo := &peer.AddrInfo{
			ID: peer.ID(node.Pid),
		}
		addrInfos[node.Id] = addrInfo
	}
	return addrInfos
}

func (swarm *Swarm) SubscribeOrderMessage(ch chan<- events.OrderMessageEvent) event.Subscription {
	return swarm.orderMessageFeed.Subscribe(ch)
}

func (swarm *Swarm) findPeer(id uint64) (string, error) {
	if swarm.routers[id] != nil {
		return swarm.routers[id].Pid, nil
	}
	newPeerAddr := swarm.notifiee.newPeer
	// new node id should be len(swarm.peers)+1
	if uint64(len(swarm.routers)+1) == id && swarm.notifiee.newPeer != "" {
		swarm.logger.Debugf("Unicast to new peer %s", swarm.notifiee.newPeer)
		return newPeerAddr, nil
	}
	return "", fmt.Errorf("wrong id: %d", id)
}

func (swarm *Swarm) AddNode(newNodeID uint64, vpInfo *pb.VpInfo) {
	if _, ok := swarm.routers[newNodeID]; ok {
		swarm.logger.Warningf("VP[ID: %d, Pid: %s] has already exist in routing table", newNodeID, vpInfo.Pid)
		return
	}
	swarm.logger.Infof("Add vp[ID: %d, Pid: %s] into routing table", newNodeID, vpInfo.Pid)
	// 1. update routers and connectedPeers
	swarm.routers[newNodeID] = vpInfo
	addInfo, err := constructMultiaddr(vpInfo)
	if err != nil {
		swarm.logger.Error("Construct AddrInfo failed")
		return
	}
	swarm.connectedPeers.Store(newNodeID, addInfo)

	// 2. persist routers
	if err := repo.RewriteNetworkConfig(swarm.repo.Config.RepoRoot, swarm.routers, false); err != nil {
		swarm.logger.Errorf("Persist routing table failed, err: %s", err.Error())
		return
	}

	// 3. update notifiee info
	swarm.notifiee.setPeers(swarm.routers)
	for id, p := range swarm.routers {
		swarm.logger.Debugf("=====ID: %d, Addr: %v=====", id, p)
	}
	if swarm.notifiee.newPeer == vpInfo.Pid {
		swarm.logger.Info("Clear notifiee newPeer info")
		swarm.notifiee.newPeer = ""
	} else if swarm.notifiee.newPeer != "" {
		swarm.logger.Warningf("Received vpInfo %v, but it doesn't equal to  notifiee newPeer %s", vpInfo, swarm.notifiee.newPeer)
	}
}

func (swarm *Swarm) DelNode(delID uint64) {
	var (
		delNode *pb.VpInfo
		ok      bool
	)
	if delNode, ok = swarm.routers[delID]; !ok {
		swarm.logger.Warningf("Can't find vp node %d from routing table ", delID)
		return
	}
	swarm.logger.Infof("Delete node [ID: %d, peerInfo: %v] ", delID, delNode)
	// 1. update routing table, multiAddrs and connectedPeers
	delete(swarm.routers, delID)
	delete(swarm.multiAddrs, delID)
	swarm.connectedPeers.Delete(delID)

	// 2. persist routers
	if err := repo.RewriteNetworkConfig(swarm.repo.Config.RepoRoot, swarm.routers, false); err != nil {
		swarm.logger.Errorf("Persist routing table failed, err: %s", err.Error())
		return
	}
	for id, p := range swarm.routers {
		swarm.logger.Debugf("=====ID: %d, Addr: %v=====", id, p)
	}
	// 3. update notifiee info
	swarm.notifiee.setPeers(swarm.routers)

	// 4. deleted node itself will exit the cluster
	if delID == swarm.localID {
		swarm.reset()
		_ = swarm.p2p.Stop()
		_ = swarm.Stop()
		return
	}
}

func (swarm *Swarm) UpdateRouter(vpInfos map[uint64]*pb.VpInfo, isNew bool) bool {
	swarm.logger.Infof("Update router: %+v", vpInfos)
	// 1. update routing table, multiAddrs and connectedPeers
	oldRouters := swarm.routers
	swarm.routers = vpInfos
	for id, _ := range oldRouters {
		if _, ok := vpInfos[id]; !ok {
			delete(swarm.multiAddrs, id)
			swarm.connectedPeers.Delete(id)
		}
	}

	// 2. persist routers
	if err := repo.RewriteNetworkConfig(swarm.repo.Config.RepoRoot, swarm.routers, isNew); err != nil {
		swarm.logger.Errorf("Persist routing table failed, err: %s", err.Error())
		return false
	}

	// 3. update notifiee info
	swarm.notifiee.setPeers(vpInfos)

	// 4. check if a restart node is exist in the routing table, if not, then exit the cluster
	var isExist bool
	for id, _ := range vpInfos {
		if id == swarm.localID {
			isExist = true
			break
		}
	}
	// deleted node itself will exit the cluster
	if !isExist && !isNew {
		swarm.reset()
		_ = swarm.p2p.Stop()
		_ = swarm.Stop()
		return true
	}
	return false
}

func (swarm *Swarm) Disconnect(vpInfos map[uint64]*pb.VpInfo) {
	for id, info := range vpInfos {
		if err := swarm.p2p.Disconnect(info.Pid); err != nil {
			swarm.logger.Errorf("Disconnect peer %s failed, err: %s", err.Error())
		}
		swarm.logger.Infof("Disconnect peer [ID: %d, Pid: %s]", id, info.Pid)
	}
}

func (swarm *Swarm) CountConnectedPeers() uint64 {
	var counter uint64
	swarm.connectedPeers.Range(func(k, v interface{}) bool {
		counter++
		return true
	})
	return counter
}

func (swarm *Swarm) reset() {
	swarm.routers = nil
	swarm.multiAddrs = nil
	swarm.connectedPeers = sync.Map{}
	swarm.notifiee.setPeers(nil)
}

func constructMultiaddr(vpInfo *pb.VpInfo) (*peer.AddrInfo, error) {
	addrs := make([]ma.Multiaddr, 0)
	for _, host := range vpInfo.Hosts {
		addr, err := ma.NewMultiaddr(fmt.Sprintf("%s%s", host, vpInfo.Pid))
		if err != nil {
			return nil, fmt.Errorf("new Multiaddr error:%w", err)
		}
		addrs = append(addrs, addr)
	}

	addrInfo := &peer.AddrInfo{
		ID:    peer.ID(vpInfo.Pid),
		Addrs: addrs,
	}
	return addrInfo, nil
}

func (swarm *Swarm) PierManager() PierManager {
	return swarm
}
