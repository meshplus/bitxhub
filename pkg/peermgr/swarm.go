package peermgr

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum/event"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	network "github.com/meshplus/go-lightp2p"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

const (
	protocolID protocol.ID = "/B1txHu6/1.0.0" // magic protocol
)

var _ PeerManager = (*Swarm)(nil)

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
	gater          connmgr.ConnectionGater

	ledger           *ledger.Ledger
	orderMessageFeed event.Feed
	enablePing       bool
	pingTimeout      time.Duration
	pingC            chan *repo.Ping

	ctx    context.Context
	cancel context.CancelFunc
}

func New(repoConfig *repo.Repo, logger logrus.FieldLogger, ledger *ledger.Ledger) (*Swarm, error) {
	ctx, cancel := context.WithCancel(context.Background())
	swarm := &Swarm{repo: repoConfig, logger: logger, ledger: ledger, ctx: ctx, cancel: cancel}
	if err := swarm.init(); err != nil {
		panic(err)
	}

	return swarm, nil
}

func (swarm *Swarm) init() error {
	var protocolIDs = []string{string(protocolID)}
	// init peers with ips and hosts
	routers := swarm.repo.NetworkConfig.GetVpInfos()
	bootstrap := make([]string, 0)
	for _, p := range routers {
		if p.Id == swarm.repo.NetworkConfig.ID {
			continue
		}
		addr := fmt.Sprintf("%s%s", p.Hosts[0], p.Pid)
		bootstrap = append(bootstrap, addr)
	}

	multiAddrs := make(map[uint64]*peer.AddrInfo)
	p2pPeers, _ := swarm.repo.NetworkConfig.GetNetworkPeers()
	for id, node := range p2pPeers {
		if id == swarm.repo.NetworkConfig.ID {
			continue
		}
		multiAddrs[id] = node
	}

	tpt, err := libp2pcert.New(swarm.repo.Key.Libp2pPrivKey, swarm.repo.Certs)
	if err != nil {
		return fmt.Errorf("create transport: %w", err)
	}

	notifiee := newNotifiee(routers, swarm.logger)
	gater := newConnectionGater(swarm.logger, swarm.ledger)

	opts := []network.Option{
		network.WithLocalAddr(swarm.repo.NetworkConfig.LocalAddr),
		network.WithPrivateKey(swarm.repo.Key.Libp2pPrivKey),
		network.WithProtocolIDs(protocolIDs),
		network.WithLogger(swarm.logger),
		// enable discovery
		network.WithBootstrap(bootstrap),
		network.WithNotify(notifiee),
		network.WithConnectionGater(gater),
	}

	if swarm.repo.Config.Cert.Verify {
		opts = append(opts,
			network.WithTransportId(libp2pcert.ID),
			network.WithTransport(tpt),
		)
	}

	p2p, err := network.New(opts...)
	if err != nil {
		return fmt.Errorf("create p2p: %w", err)
	}
	swarm.localID = swarm.repo.NetworkConfig.ID
	swarm.p2p = p2p
	swarm.enablePing = swarm.repo.Config.Ping.Enable
	swarm.pingTimeout = swarm.repo.Config.Ping.Duration
	swarm.pingC = make(chan *repo.Ping)
	swarm.routers = routers
	swarm.multiAddrs = multiAddrs
	if swarm.piers == nil {
		swarm.piers = newPiers()
	}
	swarm.connectedPeers = sync.Map{}
	swarm.notifiee = notifiee
	swarm.gater = gater
	return nil
}

func (swarm *Swarm) Start() error {
	swarm.p2p.SetMessageHandler(swarm.handleMessage)

	if err := swarm.p2p.Start(); err != nil {
		return fmt.Errorf("start p2p failed: %w", err)
	}

	for id, addr := range swarm.multiAddrs {
		go func(id uint64, addr *peer.AddrInfo, ctx context.Context) {
			if err := retry.Retry(func(attempt uint) error {
				select {
				case <-swarm.ctx.Done():
					return nil

				default:
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
						return fmt.Errorf("connect failed: %w", err)
					}

					swarm.logger.WithFields(logrus.Fields{
						"node": id,
					}).Info("Connect successfully")

					swarm.connectedPeers.Store(id, addr)

					return nil
				}
			},
				strategy.Wait(1*time.Second),
			); err != nil {
				swarm.logger.Error(err)
			}
		}(id, addr, swarm.ctx)
	}

	go swarm.Ping()

	return nil
}

func (swarm *Swarm) Stop() error {
	swarm.cancel()
	return swarm.p2p.Stop()
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
		case pingConfig := <-swarm.pingC:
			swarm.enablePing = pingConfig.Enable
			swarm.pingTimeout = pingConfig.Duration
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

func (swarm *Swarm) AsyncSend(id orderPeerMgr.KeyType, msg *pb.Message) error {
	var (
		addr string
		err  error
	)
	if addr, err = swarm.findPeer(id.(uint64)); err != nil {
		return fmt.Errorf("p2p send: %w", err)
	}

	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("marshal message error: %w", err)
	}
	return swarm.p2p.AsyncSend(addr, data)
}

func (swarm *Swarm) SendWithStream(s network.Stream, msg *pb.Message) error {
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("marshal message error: %w", err)
	}

	return s.AsyncSend(data)
}

func (swarm *Swarm) Send(id orderPeerMgr.KeyType, msg *pb.Message) (*pb.Message, error) {
	var (
		addr string
		err  error
	)
	if addr, err = swarm.findPeer(id.(uint64)); err != nil {
		return nil, fmt.Errorf("check id: %w", err)
	}

	data, err := msg.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal message error: %w", err)
	}

	ret, err := swarm.p2p.Send(addr, data)
	if err != nil {
		return nil, fmt.Errorf("sync send: %w", err)
	}

	m := &pb.Message{}
	if err := m.Unmarshal(ret); err != nil {
		return nil, fmt.Errorf("unmarshal message error: %w", err)
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
		return fmt.Errorf("marshal message error: %w", err)
	}

	return swarm.p2p.Broadcast(addrs, data)
}

func (swarm *Swarm) Peers() map[string]*peer.AddrInfo {
	//TODO: Too much redundant code, Optimize implementation logic.
	addrInfos := make(map[string]*peer.AddrInfo)
	for _, node := range swarm.notifiee.getPeers() {
		addrInfo := &peer.AddrInfo{
			ID: peer.ID(node.Pid),
		}
		addrInfos[strconv.FormatUint(node.Id, 10)] = addrInfo
	}
	return addrInfos
}

func (swarm *Swarm) OrderPeers() map[uint64]*pb.VpInfo {
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

func (swarm *Swarm) SubscribeOrderMessage(ch chan<- orderPeerMgr.OrderMessageEvent) event.Subscription {
	return swarm.orderMessageFeed.Subscribe(ch)
}

func (swarm *Swarm) findPeer(id uint64) (string, error) {
	if swarm.routers[id] != nil {
		return swarm.routers[id].Pid, nil
	}
	newPeerAddr := swarm.notifiee.newPeer
	// new node id should be len(swarm.peers)+1
	if swarm.notifiee.newPeer != "" {
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
		swarm.logger.Errorf("Construct AddrInfo failed: %s", err.Error())
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
	if len(vpInfo.Hosts) == 0 {
		return nil, fmt.Errorf("no hosts found by node:%d", vpInfo.Id)
	}
	for _, host := range vpInfo.Hosts {
		addr, err := ma.NewMultiaddr(fmt.Sprintf("%s%s", host, vpInfo.Pid))
		if err != nil {
			return nil, fmt.Errorf("new Multiaddr error:%w", err)
		}
		addrs = append(addrs, addr)
	}

	addrInfo, err := peer.AddrInfoFromP2pAddr(addrs[0])
	if err != nil {
		return nil, fmt.Errorf("convert multiaddr to addrinfo failed: %w", err)
	}
	return addrInfo, nil
}

func (swarm *Swarm) PierManager() PierManager {
	return swarm
}

func (swarm *Swarm) ReConfig(config interface{}) error {
	switch config.(type) {
	case *repo.Config:
		config := config.(*repo.Config)
		swarm.pingC <- &config.Ping
	case *repo.NetworkConfig:
		config := config.(*repo.NetworkConfig)
		if err := swarm.Stop(); err != nil {
			return fmt.Errorf("stop swarm failed: %w", err)
		}
		swarm.repo.NetworkConfig = config
		if err := swarm.init(); err != nil {
			return fmt.Errorf("init swarm failed: %w", err)
		}
		if err := swarm.Start(); err != nil {
			return fmt.Errorf("start swarm failed: %w", err)
		}
	}
	return nil
}
