package mempool

import (
	"github.com/meshplus/bitxhub-model/pb"
)

// broadcast the new transaction to other nodes
func (mpi *mempoolImpl) broadcast(m *pb.Message) {
	for id := range mpi.peerMgr.Peers() {
		if id == mpi.localID {
			continue
		}
		go func(id uint64) {
			if err := mpi.peerMgr.AsyncSend(id, m); err != nil {
				mpi.logger.Debugf("Send tx slice to peer %d failed, err: %s", id, err.Error())
			}
		}(id)
	}
}

func (mpi *mempoolImpl) unicast(to uint64, m *pb.Message) {
	go func() {
		if err := mpi.peerMgr.AsyncSend(to, m); err != nil {
			mpi.logger.Warningf("Send message to peer %d failed, err: %s", to, err.Error())
		}
	}()
}
