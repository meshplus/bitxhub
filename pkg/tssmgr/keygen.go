package tssmgr

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/meshplus/bitxhub-core/tss"
	"github.com/meshplus/bitxhub-core/tss/conversion"
	"github.com/meshplus/bitxhub-core/tss/keygen"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

func (t *TssMgr) Keygen() error {
	// 1. 获取本地持久化公钥信息
	tssStatus := true // specify whether the current node is a TSS node
	myPoolPkAddr, _, err := t.GetTssPubkey()
	if err != nil {
		tssStatus = false
	}

	// 2. 如果自己有持久化数据，向其他节点请求公钥，验证是否一致
	// 如果自己没有持久化数据，直接keygen
	if tssStatus {
		// record the TSS public key information stored on all nodes
		tssPkMap := map[string]string{}
		tssPkMap[strconv.Itoa(int(t.localID))] = myPoolPkAddr

		// 2.1 向其他节点获取公钥
		for id, addr := range t.fetchTssPkFromOtherPeers() {
			tssPkMap[id] = addr
		}

		// 2.2 检查公钥一致个数
		ok, ids, err := checkTssPkMap(tssPkMap, t.threshold+1)
		if err != nil {
			return fmt.Errorf("check pool pk map: %w", err)
		}

		// 2.3 如果检查累计一致的公钥个数达到t+1，门限签名可以正常使用，将不一致的节点踢出签名组合即可
		// 如果检查不通过，当前的门限签名已经不能正常使用，需要重新keygen
		if ok {
			t.logger.WithFields(logrus.Fields{
				"ids": ids,
			}).Infof("delete culprits from localState")
			// 将不一致的参与方踢出tss节点
			if err := t.DeleteCulprits(ids); err != nil {
				return fmt.Errorf("handle culprits: %w", err)
			}
			// 继续使用之前的公钥
			return nil
		}
	}

	// 3. 开始keygen
	if err = retry.Retry(func(attempt uint) error {
		// 3.1 获取参与keygen节点的公钥
		var (
			keys = []crypto.PubKey{t.localPubK}
		)
		for _, key := range t.fetchPkFromOtherPeers() {
			keys = append(keys, key)
		}
		t.logger.WithFields(logrus.Fields{
			"pubkeys": keys,
		}).Infof("tss keygen peer pubkeys")

		// 3.2 从实例池中取一个tss实例进行keygen
		keygenReq := keygen.NewRequest(keys)
		msgID, err := keygenReq.RequestToMsgId()
		if err != nil {
			return err
		}
		tssInstance := t.tssPools.Get()
		defer t.tssPools.Put(tssInstance)
		err = tssInstance.(*tss.TssInstance).InitTssInfo(msgID, 1, t.localPrivK, t.threshold, t.tssConf, t.keygenPreParams, t.keygenLocalState, t.peerMgr, t.logger)
		if err != nil {
			t.logger.WithFields(logrus.Fields{
				"pubkeys": keys,
				"error":   err,
			}).Warnf("tss init error, retry later...")
			return fmt.Errorf("tss init error: %v", err)
		}
		_, ok := t.tssInstances.Load(msgID)
		if ok {
			return fmt.Errorf("repeated msgID: %s", msgID)
		}
		t.tssInstances.Store(msgID, tssInstance)
		defer t.tssInstances.Delete(msgID)
		resp, err := tssInstance.(*tss.TssInstance).Keygen(keygenReq)
		if err != nil {
			t.logger.WithFields(logrus.Fields{
				"pubkeys": keys,
				"error":   err,
			}).Warnf("tss keygen error, retry later...")
			return fmt.Errorf("tss key generate: %w", err)
		} else {
			t.keygenLocalState = resp.LocalState
			if err := t.stateMgr.SaveLocalState(resp.LocalState); err != nil {
				return err
			}
			return nil
		}
	}, strategy.Wait(500*time.Millisecond), strategy.Limit(5),
	); err != nil {
		t.logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("tss keygen failed")
		return fmt.Errorf("tss keygen failed: %v", err)
	}

	return nil
}

func checkTssPkMap(pkAddrMap map[string]string, threshold uint64) (bool, []string, error) {
	freq := make(map[string]int, len(pkAddrMap))
	for _, addr := range pkAddrMap {
		freq[addr]++
	}
	maxFreq := -1
	var pkAddr string
	for key, counter := range freq {
		if counter > maxFreq {
			maxFreq = counter
			pkAddr = key
		}
	}

	if pkAddr == "" {
		return false, nil, nil
	}

	if maxFreq < int(threshold) {
		return false, nil, nil
	}

	ids := []string{}
	for id, addr := range pkAddrMap {
		if addr != pkAddr {
			ids = append(ids, id)
		}
	}

	return true, ids, nil
}

func (t *TssMgr) fetchPkFromOtherPeers() map[uint64]crypto.PubKey {
	var (
		result = make(map[uint64]crypto.PubKey)
		wg     = sync.WaitGroup{}
		lock   = sync.Mutex{}
	)

	wg.Add(len(t.peerMgr.OtherPeers()))
	for pid := range t.peerMgr.OtherPeers() {
		// 当某节点重试一定次数后仍未拿到可以不要，只要有门限以上个即可，在创建密钥时会做数量的检查
		go func(pid uint64, result map[uint64]crypto.PubKey, wg *sync.WaitGroup, lock *sync.Mutex) {
			if err := retry.Retry(func(attempt uint) error {
				pk, err := t.requestPkFromPeer(pid)
				if err != nil {
					t.logger.WithFields(logrus.Fields{
						"pid": pid,
						"err": err.Error(),
					}).Warnf("fetch pubkey from other peers error")
					return err
				} else {
					lock.Lock()
					result[pid] = pk
					lock.Unlock()
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
		}(pid, result, &wg, &lock)
	}

	wg.Wait()

	return result
}

func (t *TssMgr) requestPkFromPeer(pid uint64) (crypto.PubKey, error) {
	req := pb.Message{
		Type: pb.Message_FETCH_P2P_PUBKEY,
	}

	resp, err := t.peerMgr.Send(pid, &req)
	if err != nil {
		return nil, fmt.Errorf("send message to %d failed: %w", pid, err)
	}
	if resp == nil || resp.Type != pb.Message_FETCH_P2P_PUBKEY_ACK {
		return nil, fmt.Errorf("invalid fetch p2p pk resp")
	}

	return conversion.GetPubKeyFromPubKeyData(resp.Data)
}

func (t *TssMgr) fetchTssPkFromOtherPeers() map[string]string {
	var (
		result = make(map[string]string)
		wg     = sync.WaitGroup{}
		lock   = sync.Mutex{}
	)

	wg.Add(len(t.peerMgr.OtherPeers()))
	for pid := range t.peerMgr.OtherPeers() {
		go func(pid uint64, result map[string]string, wg *sync.WaitGroup, lock *sync.Mutex) {
			defer wg.Done()
			if err := retry.Retry(func(attempt uint) error {
				pkAddr, err := t.requestTssPkFromPeer(pid)
				if err != nil {
					t.logger.WithFields(logrus.Fields{
						"pid": pid,
						"err": err.Error(),
					}).Warnf("Get peer tss pubkey with error")
					return err
				} else {
					lock.Lock()
					result[strconv.Itoa(int(pid))] = pkAddr
					lock.Unlock()
					return nil
				}
			}, strategy.Wait(500*time.Millisecond),
			); err != nil {
				t.logger.WithFields(logrus.Fields{
					"pid": pid,
					"err": err.Error(),
				}).Warnf("retry error")
			}
		}(pid, result, &wg, &lock)
	}

	wg.Wait()
	return result
}

func (t *TssMgr) requestTssPkFromPeer(pid uint64) (string, error) {
	req := pb.Message{
		Type: pb.Message_FETCH_TSS_PUBKEY,
	}

	resp, err := t.peerMgr.Send(pid, &req)
	if err != nil {
		return "", fmt.Errorf("send message to %d failed: %w", pid, err)
	}
	if resp == nil || resp.Type != pb.Message_FETCH_TSS_PUBKEY_ACK {
		return "", fmt.Errorf("invalid fetch tss pk resp")
	}

	return string(resp.Data), nil
}
