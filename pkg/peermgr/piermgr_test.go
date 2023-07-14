package peermgr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSwarm_AskPierMaster(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	ret, err := swarms[0].PierManager().AskPierMaster("0x22222222222222222222")
	require.Nil(t, err)
	require.False(t, ret)

	err = swarms[0].PierManager().Piers().HeartBeat("0x222222222222", "2")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "no master pier")

	swarms[0].piers.pierMap.rmPier("0x222222222222222222222")
}

func TestSwarm_Piers(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	err := swarms[0].PierManager().Piers().SetMaster("0xmaster", "1", 300)
	require.Nil(t, err)
	ret := swarms[0].PierManager().Piers().CheckMaster2("0xmaster")
	require.True(t, ret)
	res := swarms[0].PierManager().Piers().HasPier("0xmaster")
	require.True(t, res)
}
