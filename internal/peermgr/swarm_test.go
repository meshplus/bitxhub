package peermgr

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/pkg/repo"
	"github.com/axiomesh/eth-kit/ledger/mock_ledger"
)

func TestSwarm_OtherPeers(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt, false)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}
}

func TestVersionCheck(t *testing.T) {
	peerCnt := 4
	// pass true to change the last Node's version
	swarms := NewSwarms(t, peerCnt, true)
	defer stopSwarms(t, swarms)
	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	msg := &pb.Message{Type: pb.Message_GET_BLOCK, Data: []byte(strconv.Itoa(1))}

	_, err := swarms[0].Send(swarms[1].PeerID(), msg)
	assert.Nil(t, err)

	_, err = swarms[0].Send(swarms[peerCnt-1].PeerID(), msg)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "protocols not supported",
		"err should be protocols not supported")
}

func TestSwarm_OnConnected(t *testing.T) {
	config := generateMockConfig(t)
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	// mock data for ledger
	chainMeta := &types.ChainMeta{
		Height:    1,
		BlockHash: types.NewHashByStr("0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"),
	}
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()

	jsonBytes, err := json.Marshal(config.Genesis.EpochInfo)
	assert.Nil(t, err)

	stateLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).DoAndReturn(func(addr *types.Address, key []byte) (bool, []byte) {
		return true, jsonBytes
	}).AnyTimes()

	stateLedger.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(addr *types.Address, key []byte, value []byte) {},
	).AnyTimes()

	mockLedger.SetState(types.NewAddressByStr(common.NodeManagerContractAddr), []byte(common.NodeManagerContractAddr), jsonBytes)

	var peerID = "16Uiu2HAmRypzJbdbUNYsCV2VVgv9UryYS5d7wejTJXT73mNLJ8AK"

	success, data := mockLedger.GetState(types.NewAddressByStr(common.NodeManagerContractAddr), []byte(common.NodeManagerContractAddr))
	if success {
		stringData := strings.Split(string(data), ",")
		for _, nodeID := range stringData {
			if peerID == nodeID {
				fmt.Println("exist nodeMembernodeID: " + nodeID)
				break
			}
		}
	} else {
		fmt.Println("get nodeMember err")
	}
}

func generateMockConfig(t *testing.T) *repo.Config {
	r, err := repo.Default(t.TempDir())
	assert.Nil(t, err)
	config := r.Config

	for i := 0; i < 4; i++ {
		config.Genesis.Admins = append(config.Genesis.Admins, &repo.Admin{
			Address: types.NewAddress([]byte{byte(1)}).String(),
		})
	}

	return config
}
