package tssmgr

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	crypto3 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/meshplus/bitxhub-core/tss"
	"github.com/meshplus/bitxhub-core/tss/conversion"
	"github.com/meshplus/bitxhub-core/tss/keysign"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

// KeySign return :
// - signature data
// - blame nodes id list
// - error
func (t *TssMgr) KeySign(signers []string, msgs []string, randomN string) ([]byte, []string, error) {
	//t.keyRoundDone.Store(false)
	//defer t.keyRoundDone.Store(true)
	// 1. get pool pubKey
	_, pk, err := t.GetTssPubkey()
	if err != nil {
		return nil, nil, fmt.Errorf("get tss pubkey error: %w", err)
	}

	// 2. get signers pk
	tssInfo, err := t.GetTssInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("fail to get keygen parties pk map error: %w", err)
	}
	signersPk := make([]crypto3.PubKey, 0)
	for _, id := range signers {
		data, ok := tssInfo.PartiesPkMap[id]
		if !ok {
			return nil, nil, fmt.Errorf("party %s is not keygen party", id)
		}
		pk, err := conversion.GetPubKeyFromPubKeyData(data)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to conversion pubkeydata to pubkey: %w", err)
		}
		signersPk = append(signersPk, pk)
	}

	// 3, new req to sign
	keysignReq := keysign.NewRequest(pk, msgs, signersPk, randomN)
	msgID, err := keysignReq.RequestToMsgId()
	if err != nil {
		return nil, nil, err
	}
	tssInstance := t.tssPools.Get().(*tss.TssInstance)
	defer t.tssPools.Put(tssInstance)
	err = tssInstance.InitTssInfo(msgID, len(keysignReq.Messages), t.localPrivK, t.threshold, t.tssConf, t.keygenPreParams, t.keygenLocalState, t.peerMgr, t.logger)
	if err != nil {
		return nil, nil, fmt.Errorf("tss init error: %v", err)
	}

	// for this msgID, start key sign round
	t.keyRoundDone.Add(msgID, &KeyRoundDoneInfo{ParitiesIDLen: len(signers), RemoteDoneIDLen: 0})

	_, ok := t.tssInstances.Load(msgID)
	if ok {
		return nil, nil, fmt.Errorf("repeated msgID: %s", msgID)
	}
	t.tssInstances.Store(msgID, tssInstance)
	defer t.tssInstances.Delete(msgID)
	resp, err := tssInstance.Keysign(keysignReq)
	t.logger.WithFields(logrus.Fields{"resp": resp, "err": err}).Debug("get key sign")
	if err != nil {
		if errors.Is(err, tss.ErrNotActiveSigner) {
			return nil, nil, err
		} else if resp != nil && len(resp.Blame.BlameNodes) != 0 {
			culpritIDs := make([]string, 0)
			for _, node := range resp.Blame.BlameNodes {
				culpritIDs = append(culpritIDs, node.PartyID)
			}
			// 广播恶意参与者
			if len(culpritIDs) != 0 {
				t.broadcastCulprits(culpritIDs)
			}
			return nil, culpritIDs, err
		} else {
			return nil, nil, fmt.Errorf("failed to tss key sign: %v", err)
		}
	}

	signData, err := json.Marshal(resp.Signatures)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal tss signatures: %v", err)
	}

	return signData, nil, nil
}

func (t *TssMgr) broadcastCulprits(culprits []string) {
	var (
		wg = sync.WaitGroup{}
	)

	wg.Add(len(t.peerMgr.OtherPeers()))
	for pid := range t.peerMgr.OtherPeers() {
		go func(pid uint64, wg *sync.WaitGroup) {
			if err := retry.Retry(func(attempt uint) error {
				err := t.sendCulprits(pid, culprits)
				if err != nil {
					t.logger.WithFields(logrus.Fields{
						"pid": pid,
						"err": err.Error(),
					}).Warnf("fetch tss info from other peers error")
					return err
				} else {
					return nil
				}
			}, strategy.Limit(5), strategy.Wait(500*time.Millisecond),
			); err != nil {
				t.logger.WithFields(logrus.Fields{
					"pid": pid,
					"err": err.Error(),
				}).Warnf("retry error")
			}

			wg.Done()
		}(pid, &wg)
	}
	wg.Wait()
}

func (t *TssMgr) sendCulprits(pid uint64, culprits []string) error {
	req := pb.Message{
		Type: pb.Message_TSS_CULPRITS,
		Data: []byte(strings.Join(culprits, ",")),
	}

	err := t.peerMgr.AsyncSend(pid, &req)
	if err != nil {
		return fmt.Errorf("send message to %d failed: %w", pid, err)
	}
	return nil
}
