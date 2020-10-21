package mempool

import (
	"errors"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
)

var _ MemPool = (*mempoolImpl)(nil)

//go:generate mockgen -destination mock_mempool/mock_mempool.go -package mock_mempool -source types.go
type MemPool interface {
	// Start starts mempool service.
	Start() error

	// Stop stops mempool service.
	Stop()

	// RecvTransaction receives transaction from API.
	RecvTransaction(tx *pb.Transaction) error

	// RecvForwardTxs receives transactions from other vp nodes.
	RecvForwardTxs(txSlice *TxSlice)

	// UpdateLeader updates the leader node info when the consensus module updates the leader status.
	UpdateLeader(uint64)

	// FetchTxn sends the fetching missing transactions request.
	FetchTxn(lostTxnEvent *LocalMissingTxnEvent)

	// RecvFetchTxnRequest receives the fetching missing transactions request,
	// and load the related transactions from its mempool or storage.
	RecvFetchTxnRequest(fetchTxnRequest *FetchTxnRequest)

	// RecvFetchTxnResponse receives the fetching missing transactions response,
	// and posts to consensus module.
	RecvFetchTxnResponse(fetchTxnResponse *FetchTxnResponse)

	// GetChainHeight gets the sequence number of block.
	GetChainHeight() uint64

	// IncreaseChainHeight increases the sequence number of block.
	IncreaseChainHeight()

	// GetBlock return the transactions list by given ready.
	// If the corresponding transaction cannot be found, a list of missing transactions will be returned.
	GetBlockByHashList(ready *raftproto.Ready) (map[uint64]string, []*pb.Transaction)

	// Remove removes the committed transactions from mempool
	CommitTransactions(ready *raftproto.Ready)

	// GetPendingNonceByAccount will return the latest pending nonce of a given account
	GetPendingNonceByAccount(account string) uint64
}

// NewMempool return the mempool instance.
func NewMempool(config *Config, storage storage.Storage, batchC chan *raftproto.Ready) MemPool {
	return newMempoolImpl(config, storage, batchC)
}

// RecvTransaction receives transaction from api and other vp nodes.
func (mpi *mempoolImpl) RecvTransaction(tx *pb.Transaction) error {
	if mpi.txCache.IsFull() && mpi.poolIsFull() {
		return errors.New("transaction cache and pool are full, we will drop this transaction")
	}
	// TODO（YH）: how to inform the client that the nonce of is wrong, need to sync to correct nonce.
	mpi.txCache.recvTxC <- tx
	return nil
}

// RecvTransaction receives transaction from api and other vp nodes.
func (mpi *mempoolImpl) RecvForwardTxs(txSlice *TxSlice) {
	mpi.subscribe.txForwardC <- txSlice
}

// UpdateLeader updates the
func (mpi *mempoolImpl) UpdateLeader(newLeader uint64) {
	// TODO (YH): should block until mempool finishing updating the leader info.
	mpi.subscribe.updateLeaderC <- newLeader
}

// FetchTxn sends the fetch request.
func (mpi *mempoolImpl) FetchTxn(lostTxnEvent *LocalMissingTxnEvent) {
	mpi.subscribe.localMissingTxnEvent <- lostTxnEvent
}

func (mpi *mempoolImpl) RecvFetchTxnRequest(fetchTxnRequest *FetchTxnRequest) {
	mpi.subscribe.fetchTxnRequestC <- fetchTxnRequest
}

func (mpi *mempoolImpl) RecvFetchTxnResponse(fetchTxnResponse *FetchTxnResponse) {
	mpi.subscribe.fetchTxnResponseC <- fetchTxnResponse
}

// Start starts the mempool service.
func (mpi *mempoolImpl) Start() error {
	mpi.logger.Debug("Start Listen mempool events")
	go mpi.listenEvent()
	go mpi.txCache.listenEvent()
	return nil
}

func (mpi *mempoolImpl) Stop() {
	if mpi.close != nil {
		close(mpi.close)
	}
	if mpi.txCache.close != nil {
		close(mpi.txCache.close)
	}
}

func (mpi *mempoolImpl) GetChainHeight() uint64 {
	return mpi.getBatchSeqNo()
}

func (mpi *mempoolImpl) IncreaseChainHeight() {
	mpi.increaseBatchSeqNo()
}

func (mpi *mempoolImpl) GetBlockByHashList(ready *raftproto.Ready) (missingTxnHashList map[uint64]string, txList []*pb.Transaction) {
	waitC := make(chan *mempoolBatch)
	getBlock := &constructBatchEvent{
		ready:  ready,
		result: waitC,
	}
	mpi.subscribe.getBlockC <- getBlock
	// block until finishing constructing related batch
	batch := <-waitC
	return batch.missingTxnHashList, batch.txList
}

func (mpi *mempoolImpl) CommitTransactions(ready *raftproto.Ready) {
	mpi.subscribe.commitTxnC <- ready
}

func (mpi *mempoolImpl) GetPendingNonceByAccount(account string) uint64 {
	waitC := make(chan uint64)
	getNonceRequest := &getNonceRequest{
		account: account,
		waitC:   waitC,
	}
	mpi.subscribe.pendingNonceC <- getNonceRequest
	return <-waitC
}
