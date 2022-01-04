package peermgr

import (
	"context"
	"fmt"
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

func (swarm *Swarm) AskPierMaster(address string) (bool, error) {
	message := &pb.Message{
		Data: []byte(address),
		Type: pb.Message_CHECK_MASTER_PIER,
	}

	if swarm.piers.pierChan.checkAddress(address) {
		return false, fmt.Errorf("is checking pier master")
	}

	ch := swarm.piers.pierChan.newChan(address)

	for id := range swarm.OrderPeers() {
		if err := swarm.AsyncSend(id, message); err != nil {
			swarm.logger.Debugf("send tx to:%d %s", id, err.Error())
			continue
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), askTimeout)
	for {
		select {
		case resp, ok := <-ch:
			if !ok {
				cancel()
				return false, fmt.Errorf("channel closed unexpectedly")
			}
			if resp.Status == pb.CheckPierResponse_HAS_MASTER {
				swarm.logger.Infoln("get p2p response")
				swarm.piers.pierChan.closeChan(address)
				cancel()
				return true, nil
			}
		case <-ctx.Done():
			// swarm.logger.Infoln("timeout!")
			swarm.piers.pierChan.closeChan(address)
			cancel()
			return false, nil
		}
	}
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

func (p *Piers) CheckMaster(address string) bool {
	res := p.pierMap.checkMaster(address)
	if !res {
		p.pierMap.rmPier(address)
	}
	return res
}

func (p *Piers) SetMaster(address string, index string, timeout int64) error {
	return p.pierMap.setMaster(address, index, timeout)
}

func (p *Piers) HeartBeat(address string, index string) error {
	return p.pierMap.heartBeat(address, index)
}

func newPierMap() *pierMap {
	return &pierMap{
		statusMap: make(map[string]*pierStatus),
	}
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

func (pm *pierMap) checkMaster(address string) bool {
	pm.RLock()
	defer pm.RUnlock()

	return pm.cmpOffset(address)
}

func (pm *pierMap) setMaster(address string, index string, timeout int64) error {
	pm.Lock()
	defer pm.Unlock()

	if pm.cmpOffset(address) {
		if pm.statusMap[address].index != index {
			return fmt.Errorf("already has master pier")
		}
	}

	pm.statusMap[address] = &pierStatus{
		index:      index,
		lastActive: time.Now(),
		timeout:    timeout,
	}
	return nil
}

func (pm *pierMap) rmMaster(address string) {
	pm.Lock()
	defer pm.Unlock()

	delete(pm.statusMap, address)
}

func (pm *pierMap) heartBeat(address string, index string) error {
	pm.RLock()
	defer pm.RUnlock()

	p, ok := pm.statusMap[address]
	if !ok {
		return fmt.Errorf("no master pier")
	}
	if p.index != index {
		return fmt.Errorf("wrong pier heart beat")
	}
	p.lastActive = time.Now()
	return nil
}

func (pm *pierMap) cmpOffset(address string) bool {
	p, ok := pm.statusMap[address]
	if !ok {
		return false
	}
	offset := time.Now().Unix() - p.lastActive.Unix()
	return p.timeout > offset
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
