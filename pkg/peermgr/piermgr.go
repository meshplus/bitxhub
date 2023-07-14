package peermgr

import (
	"context"
	"fmt"
	"github.com/meshplus/bitxhub-model/constant"
	"sort"
	"sync"
	"time"

	"github.com/meshplus/bitxhub-model/pb"
)

const (
	askTimeout = 10 * time.Second
)

func (swarm *Swarm) Piers() *Piers {
	return swarm.piers
}

func (swarm *Swarm) AskPierMaster(address string) ([]string, error) {
	message := &pb.Message{
		Data: []byte(address),
		Type: pb.Message_CHECK_MASTER_PIER,
	}

	if swarm.piers.pierChan.checkAddress(address) {
		return nil, fmt.Errorf("is checking pier master")
	}

	ch := swarm.piers.pierChan.newChan(address)

	var resps []string
	var n int
	for id := range swarm.Peers() {
		if err := swarm.AsyncSend(id, message); err != nil {
			swarm.logger.Debugf("send tx to:%d %s", id, err.Error())
			continue
		}
		n++
	}

	ctx, cancel := context.WithTimeout(context.Background(), askTimeout)
BreakLoop:
	for {
		select {
		case resp, ok := <-ch:
			if !ok {
				cancel()
				return nil, fmt.Errorf("channel closed unexpectedly")
			}
			swarm.logger.Infoln("get p2p response")
			resps = append(resps, resp.Index)
			if len(resps) == n {
				break BreakLoop
			}
		case <-ctx.Done():
			swarm.logger.Infoln("timeout!")
			break BreakLoop
		}
	}
	swarm.piers.pierChan.closeChan(address)
	cancel()

	var masterIDs []string
	for _, m := range resps {
		if m != constant.NoMaster {
			masterIDs = append(masterIDs, m)
		}
	}
	sort.Strings(masterIDs)
	return masterIDs, nil
}

type Piers struct {
	pierMap  *pierMap
	pierChan *pierChan
}

type pierMap struct {
	statusMap map[string]*pierStatus

	sync.RWMutex
}

type pierStatus struct {
	index      string
	lastActive time.Time
	timeout    int64
}

func newPiers() *Piers {
	return &Piers{
		pierMap:  newPierMap(),
		pierChan: newPierChan(),
	}
}

func (p *Piers) HasPier(address string) bool {
	return p.pierMap.hasPier(address)
}

func (p *Piers) CheckMaster2(address string) bool {
	_, res := p.pierMap.checkMaster(address)
	if !res {
		p.pierMap.rmPier(address)
	}
	return res
}

// CheckMaster func.
// 检查address下是否存在其他主节点，若存在，返回其ID，若不存在，返回一个特殊字符串。
func (p *Piers) CheckMaster(address string) string {
	id, exist := p.pierMap.checkMaster(address)
	if !exist {
		p.pierMap.rmPier(address)
		return constant.NoMaster
	}
	return id
}

func (p *Piers) SetMaster(address string, index string, timeout int64) (string, error) {
	return p.pierMap.setMaster(address, index, timeout)
}

func (p *Piers) HeartBeat(address string, index string) (string, error) {
	return p.pierMap.heartBeat(address, index)
}

func newPierMap() *pierMap {
	return &pierMap{
		statusMap: make(map[string]*pierStatus),
	}
}

func (pm *pierMap) getMaster(address string) string {
	pm.RLock()
	defer pm.RUnlock()

	m := pm.statusMap[address]
	if m != nil {
		return m.index
	}
	return constant.NoMaster
}

func (pm *pierMap) hasPier(address string) bool {
	pm.RLock()
	defer pm.RUnlock()

	_, ok := pm.statusMap[address]
	return ok
}

func (pm *pierMap) rmPier(address string) {
	pm.Lock()
	defer pm.Unlock()

	delete(pm.statusMap, address)
}

func (pm *pierMap) checkMaster(address string) (string, bool) {
	pm.RLock()
	defer pm.RUnlock()

	return pm.cmpOffset(address)
}

func (pm *pierMap) setMaster(address string, index string, timeout int64) (string, error) {
	pm.Lock()
	defer pm.Unlock()

	id, exist := pm.cmpOffset(address)
	if exist {
		if id < index {
			return id, fmt.Errorf("already has master pier")
		}
	}

	pm.statusMap[address] = &pierStatus{
		index:      index,
		lastActive: time.Now(),
		timeout:    timeout,
	}
	return index, nil
}

func (pm *pierMap) heartBeat(address string, index string) (string, error) {
	pm.RLock()
	defer pm.RUnlock()

	p, ok := pm.statusMap[address]
	if !ok {
		return constant.NoMaster, fmt.Errorf("no master pier")
	}
	if p.index != index {
		return p.index, fmt.Errorf("wrong pier heart beat")
	}
	p.lastActive = time.Now()
	return p.index, nil
}

func (pm *pierMap) cmpOffset(address string) (string, bool) {
	// 是否存在主节点，且主节点是否已超时
	p, ok := pm.statusMap[address]
	if !ok || p.timeout > time.Now().Unix()-p.lastActive.Unix() {
		return constant.NoMaster, false
	}
	return pm.statusMap[address].index, true
}

type pierChan struct {
	chanMap map[string]chan *pb.CheckPierResponse

	sync.RWMutex
}

func newPierChan() *pierChan {
	return &pierChan{
		chanMap: make(map[string]chan *pb.CheckPierResponse),
	}
}

func (pc *pierChan) checkAddress(address string) bool {
	pc.RLock()
	defer pc.RUnlock()

	_, ok := pc.chanMap[address]
	return ok
}

func (pc *pierChan) newChan(address string) chan *pb.CheckPierResponse {
	pc.Lock()
	defer pc.Unlock()

	ch := make(chan *pb.CheckPierResponse, 1)
	pc.chanMap[address] = ch

	return ch
}

func (pc *pierChan) writeChan(resp *pb.CheckPierResponse) {
	pc.RLock()
	defer pc.RUnlock()

	ch, ok := pc.chanMap[resp.Address]
	if !ok {
		return
	}
	ch <- resp
}

func (pc *pierChan) closeChan(address string) {
	pc.Lock()
	defer pc.Unlock()

	close(pc.chanMap[address])
	delete(pc.chanMap, address)
}
