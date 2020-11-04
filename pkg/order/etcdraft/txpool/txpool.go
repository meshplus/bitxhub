package txpool

import (
	"container/list"
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/meshplus/bitxhub/pkg/storage"
	"github.com/sirupsen/logrus"
)

type getTransactionFunc func(hash types.Hash) (*pb.Transaction, error)

var cancelKey string = "cancelKey"

type TxPool struct {
	sync.RWMutex                             //lock for the pendingTxs
	nodeId             uint64                //node id
	height             uint64                //current block height
	isExecuting        bool                  //only raft leader can execute
	pendingTxs         *list.List            //pending tx pool
	presenceTxs        sync.Map              //tx cache
	ackTxs             map[types.Hash]bool   //ack tx means get tx by pb.RaftMessage_GET_TX_ACK
	readyC             chan *raftproto.Ready //ready channel, receive by raft Propose channel
	peerMgr            peermgr.PeerManager   //network manager
	logger             logrus.FieldLogger    //logger
	reqLookUp          *order.ReqLookUp      //bloom filter
	storage            storage.Storage       //storage pending tx
	config             *Config               //tx pool config
	poolContext        *poolContext
	getTransactionFunc getTransactionFunc //get transaction by ledger
}

type Config struct {
	PackSize  int           //how many transactions should the primary pack
	BlockTick time.Duration //block packaging time period
	PoolSize  int           //how many transactions could the txPool stores in total
	SetSize   int           //how many transactions should the node broadcast at once
}

type poolContext struct {
	ctx       context.Context    //context
	cancel    context.CancelFunc //stop Execute
	timestamp string
}

//New txpool
func New(config *order.Config, storage storage.Storage, txPoolConfig *Config) (*TxPool, chan *raftproto.Ready) {
	readyC := make(chan *raftproto.Ready)
	reqLookUp, err := order.NewReqLookUp(storage, config.Logger)
	if err != nil {
		return nil, nil
	}
	txPool := &TxPool{
		nodeId:             config.ID,
		peerMgr:            config.PeerMgr,
		logger:             config.Logger,
		readyC:             readyC,
		height:             config.Applied,
		pendingTxs:         list.New(),
		ackTxs:             make(map[types.Hash]bool),
		reqLookUp:          reqLookUp,
		storage:            storage,
		getTransactionFunc: config.GetTransactionFunc,
		config:             txPoolConfig,
		poolContext:        newTxPoolContext(),
	}
	return txPool, readyC
}

//AddPendingTx add pending transaction into txpool
func (tp *TxPool) AddPendingTx(tx *pb.Transaction, isAckTx bool) error {
	if tp.PoolSize() >= tp.config.PoolSize {
		tp.logger.Warningf("Tx pool size: %d is full", tp.PoolSize())
		return nil
	}
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
	tp.pushBack(hash, tx, isAckTx)
	return nil
}

//Current txpool's size
func (tp *TxPool) PoolSize() int {
	tp.RLock()
	defer tp.RUnlock()
	return tp.pendingTxs.Len()
}

//RemoveTxs remove txs from the cache
func (tp *TxPool) RemoveTxs(hashes []types.Hash, isLeader bool) {
	tp.Lock()
	defer tp.Unlock()
	for _, hash := range hashes {
		if !isLeader {
			if e := tp.get(hash); e != nil {
				tp.pendingTxs.Remove(e)
			}
		}
		tp.presenceTxs.Delete(hash)
	}

}

//BuildReqLookUp store the bloom filter
func (tp *TxPool) BuildReqLookUp() {
	if err := tp.reqLookUp.Build(); err != nil {
		tp.logger.Errorf("bloom filter persistence error：", err)
	}
}

//CheckExecute check the txpool status, only leader node can run Execute()
func (tp *TxPool) CheckExecute(isLeader bool) {
	if isLeader {
		if !tp.isExecuting {
			go tp.execute()
		}
	} else {
		if tp.isExecuting {
			tp.poolContext.cancel()
		}
	}
}

//execute init
func (tp *TxPool) executeInit() {
	tp.Lock()
	defer tp.Unlock()
	tp.isExecuting = true
	tp.pendingTxs.Init()
	tp.presenceTxs = sync.Map{}
	tp.poolContext = newTxPoolContext()
	tp.logger.Debugf("Replica %d start txpool execute", tp.nodeId)
}

//execute schedule to collect txs to the ready channel
func (tp *TxPool) execute() {
	tp.executeInit()
	timestamp := tp.poolContext.ctx.Value(cancelKey).(string)
	ticker := time.NewTicker(tp.config.BlockTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ready := tp.ready()
			if ready == nil {
				continue
			}
			tp.readyC <- ready
		case <-tp.poolContext.ctx.Done():
			value := tp.poolContext.ctx.Value(cancelKey)
			newTimestamp := value.(string)
			if timestamp == newTimestamp {
				tp.isExecuting = false
				tp.logger.Info("Stop batching the transactions")
				return
			}
			tp.logger.Warning("Out of date done execute message")
		}
	}

}

//ready pack the block
func (tp *TxPool) ready() *raftproto.Ready {
	tp.Lock()
	defer tp.Unlock()
	l := tp.pendingTxs.Len()
	if l == 0 {
		return nil
	}

	var size int
	if l > tp.config.PackSize {
		size = tp.config.PackSize
	} else {
		size = l
	}
	hashes := make([]types.Hash, 0, size)
	for i := 0; i < size; i++ {
		front := tp.pendingTxs.Front()
		tx := front.Value.(*pb.Transaction)
		hash := tx.TransactionHash
		tp.pendingTxs.Remove(front)
		if _, ok := tp.ackTxs[hash]; ok {
			delete(tp.ackTxs, hash)
			continue
		}
		hashes = append(hashes, hash)
	}
	if len(hashes) == 0 {
		return nil
	}
	height := tp.UpdateHeight()
	tp.logger.Debugf("Leader generate a transaction batch with %d txs, which height is %d, " +
		"and now there are %d pending txs in txPool", len(hashes), height, tp.pendingTxs.Len())
	return &raftproto.Ready{
		TxHashes: hashes,
		Height:   height,
	}
}

//UpdateHeight add the block height
func (tp *TxPool) UpdateHeight() uint64 {
	return atomic.AddUint64(&tp.height, 1)
}

//GetHeight get current block height
func (tp *TxPool) GetHeight() uint64 {
	return atomic.LoadUint64(&tp.height)
}

//GetTx get the transaction by txpool or ledger
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
		if err := tp.peerMgr.AsyncSend(id, msg); err != nil {
			tp.logger.Warningf("Send tx to:%d %s", id, err.Error())
			continue
		}
	}
	return nil
}

// Fetch tx by local txpool or network
func (tp *TxPool) FetchTx(hash types.Hash, height uint64) *pb.Transaction {
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
		if tx, ok := tp.GetTx(hash, false); ok {
			return tx, nil
		}
		if err := tp.peerMgr.Broadcast(m); err != nil {
			tp.logger.Debugln(err)
		}
		return nil, fmt.Errorf("can't get tx: %s, block_height:%d", hash.String(), height)
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
	}, strategy.Wait(50*time.Millisecond)); err != nil {
		tp.logger.Errorln(err)
	}
	return tx
}

func (tp *TxPool) get(key types.Hash) *list.Element {
	e, ok := tp.presenceTxs.Load(key)
	if ok {
		return e.(*list.Element)
	}
	return nil
}

func (tp *TxPool) pushBack(key types.Hash, value interface{}, isAckTx bool) *list.Element {
	tp.Lock()
	defer tp.Unlock()
	if e := tp.get(key); e != nil {
		return nil
	}
	if isAckTx {
		tp.ackTxs[key] = true
	}
	e := tp.pendingTxs.PushBack(value)
	tp.presenceTxs.Store(key, e)
	return e
}

func compositeKey(value interface{}) []byte {
	var prefix = []byte("tx-")
	return append(prefix, []byte(fmt.Sprintf("%v", value))...)
}

func (tp *TxPool) store(tx *pb.Transaction) {
	txKey := compositeKey(tx.TransactionHash.Bytes())
	txData, _ := tx.Marshal()
	tp.storage.Put(txKey, txData)
}

func (tp *TxPool) load(hash types.Hash) (*pb.Transaction, bool) {
	txKey := compositeKey(hash.Bytes())
	txData := tp.storage.Get(txKey)
	if txData == nil {
		return nil, false
	}
	var tx pb.Transaction
	if err := tx.Unmarshal(txData); err != nil {
		tp.logger.Error(err)
		return nil, false
	}
	return &tx, true
}

//BatchStore batch store txs
func (tp *TxPool) BatchStore(hashes []types.Hash) {
	batch := tp.storage.NewBatch()
	for _, hash := range hashes {
		e := tp.get(hash)
		if e == nil {
			tp.logger.Debugln("BatchStore not found tx:", hash.String())
			continue
		}
		tx := e.Value.(*pb.Transaction)
		txKey := compositeKey(hash.Bytes())
		txData, _ := tx.Marshal()
		batch.Put(txKey, txData)
	}
	batch.Commit()
}

//BatchDelete batch delete txs
func (tp *TxPool) BatchDelete(hashes []types.Hash) {
	batch := tp.storage.NewBatch()
	for _, hash := range hashes {
		txKey := compositeKey(hash.Bytes())
		batch.Delete(txKey)
	}
	batch.Commit()
}

func newTxPoolContext() *poolContext {
	timestamp := time.Now().UnixNano()
	key := strconv.FormatInt(timestamp, 10)
	newCtx := context.WithValue(context.Background(), cancelKey, key)
	ctx, cancel := context.WithCancel(newCtx)
	return &poolContext{
		ctx:       ctx,
		cancel:    cancel,
		timestamp: key,
	}
}
