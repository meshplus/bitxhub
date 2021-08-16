package peermgr

import (
	"testing"
	"time"

	"github.com/meshplus/bitxhub-model/pb"

	"github.com/stretchr/testify/require"
)

func TestSwarm_OtherPeers(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	peers := swarms[0].OtherPeers()
	require.Equal(t, 3, len(peers))
}

func TestSwarm_AddNode(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	require.Equal(t, 4, len(swarms[0].routers))
	swarms[0].AddNode(5, &pb.VpInfo{
		Id:      5,
		Pid:     "Qmxxxxxxxxxxxxxxx",
		Account: "0x1111111111222222222233333333",
		Hosts:   []string{"/ip4/127.0.0.1/tcp/4003/p2p/,"},
	})

	require.Equal(t, 5, len(swarms[0].routers))

	swarms[0].DelNode(5)
	require.Equal(t, 4, len(swarms[0].routers))

	routers := swarms[0].routers
	delete(routers, 4)
	swarms[0].UpdateRouter(routers, false)
	require.Equal(t, 3, len(swarms[0].routers))
	require.Equal(t, 3, len(swarms[0].Peers()))
	require.Equal(t, 0, len(swarms[0].PierManager().Piers().pierMap.statusMap))

	// find wrong peer
	_, err := swarms[0].findPeer(100)
	require.NotNil(t, err)

	// delete itself
	swarms[0].DelNode(1)
	require.Equal(t, 0, len(swarms[0].routers))
}

func TestSwarm_Broadcast(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}
	msg := &pb.Message{
		Type: pb.Message_FETCH_CERT,
		Data: []byte("Hello"),
	}
	err := swarms[0].Broadcast(msg)
	require.Nil(t, err)
}

func TestSwarm_Disconnect(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	m := make(map[uint64]*pb.VpInfo)
	m[2] = &pb.VpInfo{Id: 2}
	m[3] = &pb.VpInfo{Id: 3}
	m[4] = &pb.VpInfo{Id: 4}
	swarms[0].Disconnect(m)
	require.Equal(t, 4, len(swarms[0].routers))
}
