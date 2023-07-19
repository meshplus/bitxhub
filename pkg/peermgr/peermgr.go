package peermgr

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-model/pb"
	network "github.com/meshplus/go-lightp2p"
)

// TODO: refactor
type OrderMessageEvent struct {
	IsTxsFromRemote bool
	Data            []byte
	Txs             [][]byte
}

type KeyType interface{}

type BasicPeerManager interface {
	// Start
	Start() error

	// Stop
	Stop() error

	// AsyncSend sends message to peer with peer info.
	AsyncSend(KeyType, *pb.Message) error

	// Send sends message waiting response
	Send(KeyType, *pb.Message) (*pb.Message, error)

	// CountConnectedPeers counts connected peer numbers
	CountConnectedPeers() uint64

	// Peers return all peers including local peer.
	Peers() map[string]*peer.AddrInfo
}

type OrderPeerManager interface {
	BasicPeerManager

	// SubscribeOrderMessage
	SubscribeOrderMessage(ch chan<- OrderMessageEvent) event.Subscription

	// AddNode adds a vp peer.
	AddNode(newNodeID uint64, vpInfo *pb.VpInfo)

	// DelNode deletes a vp peer.
	DelNode(delID uint64)

	// UpdateRouter update the local router to quorum router.
	UpdateRouter(vpInfos map[uint64]*pb.VpInfo, isNew bool) bool

	// OtherPeers return peers except local peer.
	OtherPeers() map[uint64]*peer.AddrInfo

	// Broadcast message to all node
	Broadcast(*pb.Message) error

	// Disconnect disconnect with all vp peers.
	Disconnect(vpInfos map[uint64]*pb.VpInfo)

	// OrderPeers return all OrderPeers include account and id.
	OrderPeers() map[uint64]*pb.VpInfo
}

//go:generate mockgen -destination mock_peermgr/mock_peermgr.go -package mock_peermgr -source peermgr.go
type PeerManager interface {
	OrderPeerManager

	// SendWithStream sends message using existed stream
	SendWithStream(network.Stream, *pb.Message) error

	// ReConfig
	ReConfig(config interface{}) error
}
