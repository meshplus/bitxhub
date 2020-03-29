package txpool

import (
	"container/list"
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meshplus/bitxhub/pkg/order"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/internal/plugins/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/meshplus/bitxhub/pkg/storage"
	"github.com/sirupsen/logrus"
)

type TxPool struct {
	sync.RWMutex
	nodeId             uint64     //node id
	height             uint64     //current block height
	pendingTxs         *list.List //pending tx pool
	presenceTxs        sync.Map   //tx cache
	readyC             chan *raftproto.Ready
	peerMgr            peermgr.PeerManager //network manager
	logger             logrus.FieldLogger  //logger
	reqLookUp          *order.ReqLookUp    // bloom filter
	storage            storage.Storage     // storage pending tx
	getTransactionFunc func(hash types.Hash) (*pb.Transaction, error)
	isExecuting        bool          //only raft leader can execute
	packSize           int           //maximum number of transaction packages
	blockTick          time.Duration //block packed period

	ctx    context.Context
	cancel context.CancelFunc
}

//New txpool
func New(config *order.Config, storage storage.Storage, packSize int, blockTick time.Duration) (*TxPool, chan *raftproto.Ready) {
	readyC := make(chan *raftproto.Ready)
	reqLookUp, err := order.NewReqLookUp(storage, config.Logger)
	if err != nil {
		return nil, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &TxPool{
		nodeId:             config.ID,
		peerMgr:            config.PeerMgr,
		logger:             config.Logger,
		readyC:             readyC,
		height:             config.Applied,
		pendingTxs:         list.New(),
		reqLookUp:          reqLookUp,
		storage:            storage,
		packSize:           packSize,
		blockTick:          blockTick,
		getTransactionFunc: config.GetTransactionFunc,
		ctx:                ctx,
		cancel:             cancel,
	}, readyC
}

//Add pending transaction into txpool
func (tp *TxPool) AddPendingTx(tx *pb.Transaction) error {
	hash := tx.TransactionHash
	if e := tp.get(hash); e != nil {
		return nil
	}
	//look up by bloom filter
	if ok := tp.reqLookUp.LookUp(hash.Bytes()); ok {
		//find the tx again by ledger if hash in bloom filter
		if tx, _ := tp.getTransactionFunc(hash); tx != nil {
			return nil
		}
	}
	//add pending tx
	tp.pushBack(hash, tx)
	return nil
}

//Current txpool's size
func (tp *TxPool) PoolSize() int {
	tp.RLock()
	defer tp.RUnlock()
	return tp.pendingTxs.Len()
}

//Remove stored transactions
func (tp *TxPool) RemoveTxs(hashes []types.Hash, isLeader bool) {
	if isLeader {
		tp.BatchDelete(hashes)
	}
	for _, hash := range hashes {
		if !isLeader {
			if e := tp.get(hash); e != nil {
				tp.Lock()
				tp.pendingTxs.Remove(e)
				tp.Unlock()
			}
		}
		tp.presenceTxs.Delete(hash)
	}
}

//Store the bloom filter
func (tp *TxPool) BuildReqLookUp() {
	if err := tp.reqLookUp.Build(); err != nil {
		tp.logger.Errorf("bloom filter persistence error：", err)
	}
}

//Check the txpool status, only leader node can run Execute()
func (tp *TxPool) CheckExecute(isLeader bool) {
	if isLeader {
		if !tp.isExecuting {
			go tp.execute()
		}
	} else {
		if tp.isExecuting {
			tp.cancel()
		}
	}
}

// Schedule to collect txs to the ready channel
func (tp *TxPool) execute() {
	tp.isExecuting = true
	tp.pendingTxs.Init()
	tp.presenceTxs = sync.Map{}
	ticker := time.NewTicker(tp.blockTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ready := tp.ready()
			if ready == nil {
				continue
			}
			tp.logger.WithFields(logrus.Fields{
				"height": ready.Height,
			}).Debugln("block will be generated")
			tp.readyC <- ready
		case <-tp.ctx.Done():
			tp.isExecuting = false
			tp.logger.Infoln("Done txpool execute")
			return
		}
	}

}

func (tp *TxPool) ready() *raftproto.Ready {
	tp.Lock()
	defer tp.Unlock()
	l := tp.pendingTxs.Len()
	if l == 0 {
		return nil
	}

	var size int
	if l > tp.packSize {
		size = tp.packSize
	} else {
		size = l
	}
	hashes := make([]types.Hash, 0, size)
	for i := 0; i < size; i++ {
		front := tp.pendingTxs.Front()
		tx := front.Value.(*pb.Transaction)
		hashes = append(hashes, tx.TransactionHash)
		tp.pendingTxs.Remove(front)
	}
	height := tp.UpdateHeight()
	return &raftproto.Ready{
		TxHashes: hashes,
		Height:   height,
	}
}

//Add the block height
func (tp *TxPool) UpdateHeight() uint64 {
	return atomic.AddUint64(&tp.height, 1)
}

//Get current block height
func (tp *TxPool) GetHeight() uint64 {
	return atomic.LoadUint64(&tp.height)
}

//Get the transaction by txpool or ledger
func (tp *TxPool) GetTx(hash types.Hash, findByStore bool) (*pb.Transaction, bool) {
	if e := tp.get(hash); e != nil {
		return e.Value.(*pb.Transaction), true
	}
	if findByStore {
		// find by txpool store
		tx, ok := tp.load(hash)
		if ok {
			return tx, true
		}
		// find by ledger
		tx, err := tp.getTransactionFunc(hash)
		if err != nil {
			return nil, false
		}
		return tx, true
	}
	return nil, false
}

//Broadcast the new transaction to other nodes
func (tp *TxPool) Broadcast(tx *pb.Transaction) error {
	data, err := tx.Marshal()
	if err != nil {
		return err
	}
	rm := &raftproto.RaftMessage{
		Type: raftproto.RaftMessage_BROADCAST_TX,
		Data: data,
	}
	cmData, err := rm.Marshal()
	if err != nil {
		return err
	}
	msg := &pb.Message{
		Type: pb.Message_CONSENSUS,
		Data: cmData,
	}

	for id := range tp.peerMgr.Peers() {
		if id == tp.nodeId {
			continue
		}
		if err := tp.peerMgr.Send(id, msg); err != nil {
			tp.logger.Debugln("send transaction error:", err)
			continue
		}
	}
	return nil
}

// Fetch tx by local txpool or network
func (tp *TxPool) FetchTx(hash types.Hash) *pb.Transaction {
	if tx, ok := tp.GetTx(hash, false); ok {
		return tx
	}
	raftMessage := &raftproto.RaftMessage{
		Type:   raftproto.RaftMessage_GET_TX,
		FromId: tp.nodeId,
		Data:   hash.Bytes(),
	}
	rmData, err := raftMessage.Marshal()
	if err != nil {
		return nil
	}
	m := &pb.Message{
		Type: pb.Message_CONSENSUS,
		Data: rmData,
	}

	asyncGet := func() (tx *pb.Transaction, err error) {
		for id := range tp.peerMgr.Peers() {
			if id == tp.nodeId {
				continue
			}
			if tx, ok := tp.GetTx(hash, false); ok {
				return tx, nil
			}
			if err := tp.peerMgr.Send(id, m); err != nil {
				return nil, err
			}
		}
		return nil, fmt.Errorf("can't get transaction: %s", hash.String())
	}

	var tx *pb.Transaction
	if err := retry.Retry(func(attempt uint) (err error) {
		tx, err = asyncGet()
		if err != nil {
			//retry times > 2
			if attempt > 2 {
				tp.logger.Debugln(err)
			}
			return err
		}
		return nil
	}, strategy.Wait(200*time.Millisecond)); err != nil {
		tp.logger.Errorln(err)
	}
	return tx
}

// Fetch tx by local txpool or network
func (tp *TxPool) FetchBlock(height uint64) (*pb.Block, error) {
	get := func(height uint64) (block *pb.Block, err error) {
		for id := range tp.peerMgr.Peers() {
			block, err = tp.getBlock(id, int(height))
			if err != nil {
				continue
			}
			return block, nil
		}
		return nil, fmt.Errorf("can't get block: %d", height)
	}

	var block *pb.Block
	if err := retry.Retry(func(attempt uint) (err error) {
		block, err = get(height)
		if err != nil {
			tp.logger.Debugln(err)
			return err
		}
		return nil
	}, strategy.Wait(200*time.Millisecond), strategy.Limit(1)); err != nil {
		return nil, err
	}
	return block, nil
}

//Get block by network
func (tp *TxPool) getBlock(id uint64, i int) (*pb.Block, error) {
	m := &pb.Message{
		Type: pb.Message_GET_BLOCK,
		Data: []byte(strconv.Itoa(i)),
	}

	res, err := tp.peerMgr.SyncSend(id, m)
	if err != nil {
		return nil, err
	}

	block := &pb.Block{}
	if err := block.Unmarshal(res.Data); err != nil {
		return nil, err
	}

	return block, nil
}

func (tp *TxPool) get(key types.Hash) *list.Element {
	e, ok := tp.presenceTxs.Load(key)
	if ok {
		return e.(*list.Element)
	}
	return nil
}

func (tp *TxPool) pushBack(key types.Hash, value interface{}) *list.Element {
	tp.Lock()
	defer tp.Unlock()
	e := tp.pendingTxs.PushBack(value)
	tp.presenceTxs.Store(key, e)
	return e
}

var transactionKey = []byte("tx-")

func compositeKey(prefix []byte, value interface{}) []byte {
	return append(prefix, []byte(fmt.Sprintf("%v", value))...)
}
func (tp *TxPool) store(tx *pb.Transaction) {
	txKey := compositeKey(transactionKey, tx.TransactionHash.Bytes())
	txData, _ := tx.Marshal()
	if err := tp.storage.Put(txKey, txData); err != nil {
		tp.logger.Error("store tx error:", err)
	}
}
func (tp *TxPool) load(hash types.Hash) (*pb.Transaction, bool) {
	txKey := compositeKey(transactionKey, hash.Bytes())
	txData, err := tp.storage.Get(txKey)
	if err != nil {
		return nil, false
	}
	var tx pb.Transaction
	if err := tx.Unmarshal(txData); err != nil {
		tp.logger.Error(err)
		return nil, false
	}
	return &tx, true
}

//batch store txs
func (tp *TxPool) BatchStore(hashes []types.Hash) {
	batch := tp.storage.NewBatch()
	for _, hash := range hashes {
		e := tp.get(hash)
		if e == nil {
			continue
		}
		tx := e.Value.(*pb.Transaction)
		txKey := compositeKey(transactionKey, hash.Bytes())
		txData, _ := tx.Marshal()
		batch.Put(txKey, txData)
	}
	if err := batch.Commit(); err != nil {
		tp.logger.Fatalf("storage batch tx error:", err)
	}
}

//batch delete txs
func (tp *TxPool) BatchDelete(hashes []types.Hash) {
	batch := tp.storage.NewBatch()
	for _, hash := range hashes {
		txKey := compositeKey(transactionKey, hash.Bytes())
		batch.Delete(txKey)
	}
	if err := batch.Commit(); err != nil {
		tp.logger.Fatalf("storage batch tx error:", err)
	}
}
