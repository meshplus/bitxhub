package peermgr

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	network "github.com/axiomesh/axiom-p2p"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/internal/repo"
	"github.com/ethereum/go-ethereum/event"
	"github.com/libp2p/go-libp2p/core/connmgr"
	p2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

const (
	protocolID string = "/B1txHu6/1.0.0" // magic protocol
)

var _ PeerManager = (*Swarm)(nil)

type Swarm struct {
	repo    *repo.Repo
	localID uint64
	p2p     network.Network
	logger  logrus.FieldLogger

	// added by
	routers        map[uint64]*types.VpInfo // trace the vp nodes
	multiAddrs     map[uint64]*peer.AddrInfo
	connectedPeers sync.Map
	gater          connmgr.ConnectionGater

	ledger           *ledger.Ledger
	orderMessageFeed event.Feed
	enablePing       bool
	pingTimeout      time.Duration
	pingC            chan *repo.Ping

	msgPushLimiter *rate.Limiter

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

	gater := newConnectionGater(swarm.logger, swarm.ledger)
	opts := []network.Option{
		network.WithLocalAddr(swarm.repo.NetworkConfig.LocalAddr),
		network.WithPrivateKey(swarm.repo.Key.Libp2pPrivKey),
		network.WithProtocolID(protocolID),
		network.WithLogger(swarm.logger),
		network.WithSecurity(network.SecurityTLS),
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
	swarm.localID = swarm.repo.NetworkConfig.ID
	swarm.p2p = p2p
	swarm.enablePing = swarm.repo.Config.Ping.Enable
	swarm.pingTimeout = swarm.repo.Config.Ping.Duration
	swarm.pingC = make(chan *repo.Ping)
	swarm.routers = routers
	swarm.multiAddrs = multiAddrs
	swarm.connectedPeers = sync.Map{}
	swarm.gater = gater

	// Initialize the message limiter and set the number of messages allowed per second to LIMIT
	swarm.msgPushLimiter = rate.NewLimiter(rate.Limit(swarm.repo.Config.P2pLimit.Limit), int(swarm.repo.Config.P2pLimit.Burst))
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
	for id, vp := range swarm.routers {
		if vp.Pid == peerID {
			swarm.connectedPeers.Store(id, swarm.multiAddrs[id])
		}
	}

	return nil
}

func (swarm *Swarm) onDisconnected(peerID string) {
	for id, vp := range swarm.routers {
		if vp.Pid == peerID {
			swarm.connectedPeers.Delete(id)
		}
	}
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

func (swarm *Swarm) AsyncSend(id KeyType, msg *pb.Message) error {
	var (
		addr string
		err  error
	)
	switch to := id.(type) {
	case uint64:
		if addr, err = swarm.findPeer(to); err != nil {
			return fmt.Errorf("p2p asyncSend check peer id type: %w", err)
		}
	case string:
		addr = to
	default:
		return fmt.Errorf("p2p unsupported peer id type: %v", id)
	}

	data, err := msg.MarshalVT()
	if err != nil {
		return fmt.Errorf("marshal message error: %w", err)
	}
	return swarm.p2p.AsyncSend(addr, data)
}

func (swarm *Swarm) SendWithStream(s network.Stream, msg *pb.Message) error {
	data, err := msg.MarshalVT()
	if err != nil {
		return fmt.Errorf("marshal message error: %w", err)
	}

	return s.AsyncSend(data)
}

func (swarm *Swarm) Send(id KeyType, msg *pb.Message) (*pb.Message, error) {
	var (
		addr string
		err  error
	)
	switch to := id.(type) {
	case uint64:
		if addr, err = swarm.findPeer(to); err != nil {
			return nil, fmt.Errorf("p2p send check peer id type: %w", err)
		}
	case string:
		addr = to
	default:
		return nil, fmt.Errorf("p2p unsupported peer id type: %v", id)
	}

	data, err := msg.MarshalVT()
	if err != nil {
		return nil, fmt.Errorf("marshal message error: %w", err)
	}

	ret, err := swarm.p2p.Send(addr, data)
	if err != nil {
		return nil, fmt.Errorf("sync send: %w", err)
	}

	m := &pb.Message{}
	if err := m.UnmarshalVT(ret); err != nil {
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

	data, err := msg.MarshalVT()
	if err != nil {
		return fmt.Errorf("marshal message error: %w", err)
	}

	return swarm.p2p.Broadcast(addrs, data)
}

func (swarm *Swarm) Peers() []peer.AddrInfo {
	return swarm.p2p.GetPeers()
}

func (swarm *Swarm) OrderPeers() map[uint64]*types.VpInfo {
	return swarm.routers
}

func (swarm *Swarm) SubscribeOrderMessage(ch chan<- OrderMessageEvent) event.Subscription {
	return swarm.orderMessageFeed.Subscribe(ch)
}

func (swarm *Swarm) findPeer(id uint64) (string, error) {
	if swarm.routers[id] != nil {
		return swarm.routers[id].Pid, nil
	}
	return "", fmt.Errorf("wrong id: %d", id)
}

// TODO: refactor
func (swarm *Swarm) AddNode(newNodeID uint64, vpInfo *types.VpInfo) {
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
}

func (swarm *Swarm) DelNode(delID uint64) {
	var (
		delNode *types.VpInfo
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

	// TODO: exit self
	// 4. deleted node itself will exit the cluster
	if delID == swarm.localID {
		swarm.reset()
		_ = swarm.p2p.Stop()
		_ = swarm.Stop()
		return
	}
}

func (swarm *Swarm) UpdateRouter(vpInfos map[uint64]*types.VpInfo, isNew bool) bool {
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

func (swarm *Swarm) Disconnect(vpInfos map[uint64]*types.VpInfo) {
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
}

func constructMultiaddr(vpInfo *types.VpInfo) (*peer.AddrInfo, error) {
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
