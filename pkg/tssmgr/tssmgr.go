package tssmgr

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	bkg "github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/libp2p/go-libp2p-core/crypto"
	peer_mgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-core/tss"
	"github.com/meshplus/bitxhub-core/tss/conversion"
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

	keygenPreParams  *bkg.LocalPreParams
	keygenLocalState *storage.KeygenLocalState
	tssPools         *sync.Pool
	tssInstances     *sync.Map

	stateMgr       storage.LocalStateManager
	stateMgrLocker *sync.Mutex
	peerMgr        peer_mgr.OrderPeerManager
	logger         logrus.FieldLogger
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
		tssInstances:    &sync.Map{},
		peerMgr:         peerMgr,
		stateMgr:        stateManager,
		stateMgrLocker:  &sync.Mutex{},
		logger:          logger,
	}, nil
}

func (t *TssMgr) Start(threshold uint64) {
	// 1. set threshold
	t.UpdateThreshold(threshold)

	// 2. load tss local state
	if err := t.loadTssLocalState(); err != nil {
		t.logger.Warn("load tss info error: %v", err)
	}

	t.logger.Infof("Starting the TSS Manager: t-%d", threshold)
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
	t.logger.Info("The Tss and p2p server has been stopped successfully")
}

func (t *TssMgr) PutTssMsg(msg *pb.Message, msgID string) {
	if err := retry.Retry(func(attempt uint) error {
		instance, ok := t.tssInstances.Load(msgID)
		if !ok {
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
