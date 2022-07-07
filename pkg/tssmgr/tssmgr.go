package tssmgr

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	bkg "github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/libp2p/go-libp2p-core/crypto"
	peer_mgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-core/tss"
	"github.com/meshplus/bitxhub-core/tss/conversion"
	"github.com/meshplus/bitxhub-core/tss/message"
	"github.com/meshplus/bitxhub-core/tss/storage"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/sirupsen/logrus"
)

var _ TssManager = (*TssMgr)(nil)

type TssMgr struct {
	localID    uint64
	localPrivK crypto.PrivKey
	localPubK  crypto.PubKey

	threshold       uint64
	thresholdLocker *sync.Mutex
	tssConf         tss.TssConfig
	netConf         *repo.NetworkConfig
	tssRepo         string
	orderReadyPeers map[uint64]bool
	keyRoundDone    atomic.Value

	keygenPreParams  *bkg.LocalPreParams
	keygenLocalState *storage.KeygenLocalState
	tssPools         *sync.Pool
	tssInstances     *sync.Map

	stateMgr       storage.LocalStateManager
	stateMgrLocker *sync.Mutex
	peerMgr        peer_mgr.OrderPeerManager
	logger         logrus.FieldLogger
	keyGenLocker   *sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
}

func NewTssMgr(
	privKey crypto.PrivKey,
	tssConf tss.TssConfig,
	netConf *repo.NetworkConfig,
	repoRoot string,
	preParams *bkg.LocalPreParams,
	peerMgr peer_mgr.OrderPeerManager,
	logger logrus.FieldLogger,
) (tssmgr *TssMgr, err error) {
	tssRepo := filepath.Join(repoRoot, tssConf.TssConfPath)
	// keygen pre params
	// When using the keygen party it is recommended that you pre-compute the
	// "safe primes" and Paillier secret beforehand because this can take some
	// time.
	// This code will generate those parameters using a concurrency limit equal
	// to the number of available CPU cores.
	if preParams == nil || !preParams.Validate() {
		preParams, err = bkg.GeneratePreParams(tssConf.PreParamTimeout)
		if err != nil {
			return nil, fmt.Errorf("fail to generate pre parameters: %w", err)
		}
	}
	if !preParams.Validate() {
		return nil, fmt.Errorf("invalid preparams")
	}

	// Persistent storage of data
	stateManager, err := storage.NewFileStateMgr(tssRepo)
	if err != nil {
		return nil, fmt.Errorf("fail to create file state manager: %w", err)
	}

	tssPools := &sync.Pool{
		New: func() interface{} {
			return new(tss.TssInstance)
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	var keyRoundDone atomic.Value
	keyRoundDone.Store(false)
	return &TssMgr{
		localID:         netConf.ID,
		localPrivK:      privKey,
		localPubK:       privKey.GetPublic(),
		thresholdLocker: &sync.Mutex{},
		tssConf:         tssConf,
		netConf:         netConf,
		tssRepo:         tssRepo,
		keygenPreParams: preParams,
		tssPools:        tssPools,
		orderReadyPeers: make(map[uint64]bool),
		keyRoundDone:    keyRoundDone,
		tssInstances:    &sync.Map{},
		peerMgr:         peerMgr,
		stateMgr:        stateManager,
		stateMgrLocker:  &sync.Mutex{},
		keyGenLocker:    &sync.Mutex{},
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
	}, nil
}

func (t *TssMgr) SetOrderReadyPeers(id uint64) {
	t.orderReadyPeers[id] = true
}

func (t *TssMgr) GetTssStatus() bool {
	return t.keyRoundDone.Load().(bool)
}

func (t *TssMgr) CountOrderReadyPeers() int {
	return len(t.orderReadyPeers)
}

func (t *TssMgr) Start(threshold uint64) {
	// 1. set threshold
	// 2. load tss local state

	t.UpdateThreshold(threshold)

	// 1. get pool addr from file

	if err := t.loadTssLocalState(); err != nil {
		t.logger.Warn("load tss info error: %v", err)
		if checkeErr := retry.Retry(func(attempt uint) error {
			select {
			case <-t.ctx.Done():
				t.logger.Infof("stop checkQuorum")
				return nil

			default:
				checkeErr := t.CheckThreshold()
				if checkeErr != nil {
					t.logger.WithFields(logrus.Fields{"config num": len(t.peerMgr.OrderPeers()),
						"order ready peer": t.orderReadyPeers}).Warning(checkeErr)
					return checkeErr
				}
				return nil
			}
		}, strategy.Wait(2*time.Second)); checkeErr != nil {
			panic(checkeErr)
		}
	}
	t.logger.Infof("Starting the TSS Manager: t-%d", threshold)
}

func (t *TssMgr) sendOrderReady() error {
	if ok := t.orderReadyPeers[t.localID]; !ok {
		t.orderReadyPeers[t.localID] = true
	}
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, t.localID)
	msg := &pb.Message{
		Type: pb.Message_FERCH_TSS_NODES,
		Data: data,
	}
	err := t.peerMgr.Broadcast(msg)
	if err != nil {
		return err
	}
	return nil
}

// make sure bitxhub have t+1 connected nodes
func (t *TssMgr) CheckThreshold() error {
	var (
		err          error
		broadcastErr error
	)
	if len(t.orderReadyPeers) < len(t.peerMgr.Peers()) {
		err = fmt.Errorf("the number of connected Peers don't reach network config")
	}
	// ensure last node receive latest nodes msg
	broadcastErr = t.sendOrderReady()
	if broadcastErr != nil {
		return fmt.Errorf("broadcast local nodes order ready err : %s", broadcastErr)
	}
	return err
}

func (t *TssMgr) GetThreshold() uint64 {
	return t.threshold
}

func (t *TssMgr) UpdateThreshold(threshold uint64) {
	t.thresholdLocker.Lock()
	t.threshold = threshold
	t.thresholdLocker.Unlock()
}

func (t *TssMgr) loadTssLocalState() error {
	// 1. get pool addr from file
	filePath := filepath.Join(t.tssRepo, storage.PoolPkAddrFileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return err
	}
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("file to read from file(%s): %w", filePath, err)
	}

	// 2. get local state by pool addr
	state, err := t.stateMgr.GetLocalState(string(buf))
	if err != nil {
		return fmt.Errorf("failed to get local state: %s,  %v", string(buf), err)
	}

	t.stateMgrLocker.Lock()
	t.keygenLocalState = state
	t.stateMgrLocker.Unlock()
	return nil
}

func (t *TssMgr) Stop() {
	t.cancel()
	t.logger.Info("The Tss and p2p server has been stopped successfully")
}

func (t *TssMgr) PutTssMsg(msg *pb.Message, msgID string) {
	if err := retry.Retry(func(attempt uint) error {
		instance, ok := t.tssInstances.Load(msgID)
		if !ok {
			wireMsg := &message.WireMessage{}
			if err := json.Unmarshal(msg.Data, wireMsg); err != nil {
				return fmt.Errorf("wire msg unmarshal error: %v", err)
			}

			t.logger.WithFields(logrus.Fields{"msgID": wireMsg.MsgID, "type": wireMsg.MsgType}).Debug("wrong tss msg")
			return fmt.Errorf("tss instance not found, msgID: %s", msgID)
		} else {
			instance.(*tss.TssInstance).PutTssMsg(msg)
			return nil
		}
	}, strategy.Wait(500*time.Millisecond), strategy.Limit(5),
	); err != nil {
		t.logger.WithFields(logrus.Fields{
			"msgID": msgID,
		}).Warnf("tss instance not found")
		return
	}
	return
}

func (t *TssMgr) GetTssPubkey() (string, *ecdsa.PublicKey, error) {
	if t.keygenLocalState == nil {
		return "", nil, fmt.Errorf("tss local state is nil")
	}
	pk, err := conversion.GetECDSAPubKeyFromPubKeyData(t.keygenLocalState.PubKeyData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get ECDSA pubKey from pubkey data: %v", err)
	}

	return t.keygenLocalState.PubKeyAddr, pk, nil
}

func (t *TssMgr) GetTssInfo() (*pb.TssInfo, error) {
	if t.keygenLocalState == nil {
		return nil, fmt.Errorf("tss local state is nil")
	}

	return &pb.TssInfo{
		PartiesPkMap: t.keygenLocalState.ParticipantPksMap,
		Pubkey:       t.keygenLocalState.PubKeyData,
	}, nil
}

func (t *TssMgr) DeleteTssNodes(nodes []string) error {
	t.stateMgrLocker.Lock()
	defer t.stateMgrLocker.Unlock()
	// 1. get pool addr from file
	filePath := filepath.Join(t.tssRepo, storage.PoolPkAddrFileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return err
	}

	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("file to read from file(%s): %w", filePath, err)
	}

	// 2. get local state by pool addr
	state, err := t.stateMgr.GetLocalState(string(buf))
	if err != nil {
		return fmt.Errorf("failed to get local state: %s,  %v", string(buf), err)
	}

	// 3. delete culprits
	for _, id := range nodes {
		delete(state.ParticipantPksMap, id)
	}

	// 4. update local state
	err = t.stateMgr.SaveLocalState(state)
	if err != nil {
		return fmt.Errorf("save local state error: %v", err)
	}

	return nil
}
