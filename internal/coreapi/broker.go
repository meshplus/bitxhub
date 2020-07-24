package coreapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/model"
	"github.com/sirupsen/logrus"
)

type BrokerAPI CoreAPI

var _ api.BrokerAPI = (*BrokerAPI)(nil)

func (b *BrokerAPI) HandleTransaction(tx *pb.Transaction) error {
	b.logger.WithFields(logrus.Fields{
		"hash": tx.TransactionHash.String(),
	}).Debugf("Receive tx")

	go func() {
		if err := b.bxh.Order.Prepare(tx); err != nil {
			b.logger.Error(err)
		}
	}()

	return nil
}

func (b *BrokerAPI) HandleView(tx *pb.Transaction) (*pb.Receipt, error) {
	b.logger.WithFields(logrus.Fields{
		"hash": tx.TransactionHash.String(),
	}).Debugf("Receive view")

	receipts := b.bxh.ViewExecutor.ApplyReadonlyTransactions([]*pb.Transaction{tx})

	return receipts[0], nil
}

func (b *BrokerAPI) GetTransaction(hash types.Hash) (*pb.Transaction, error) {
	return b.bxh.Ledger.GetTransaction(hash)
}

func (b *BrokerAPI) GetTransactionMeta(hash types.Hash) (*pb.TransactionMeta, error) {
	return b.bxh.Ledger.GetTransactionMeta(hash)
}

func (b *BrokerAPI) GetReceipt(hash types.Hash) (*pb.Receipt, error) {
	return b.bxh.Ledger.GetReceipt(hash)
}

func (b *BrokerAPI) AddPier(key string) (chan *pb.InterchainTxWrapper, error) {
	return b.bxh.Router.AddPier(key)
}

func (b *BrokerAPI) GetBlockHeader(begin, end uint64, ch chan<- *pb.BlockHeader) error {
	return b.bxh.Router.GetBlockHeader(begin, end, ch)
}

func (b *BrokerAPI) GetInterchainTxWrapper(pid string, begin, end uint64, ch chan<- *pb.InterchainTxWrapper) error {
	return b.bxh.Router.GetInterchainTxWrapper(pid, begin, end, ch)
}

func (b *BrokerAPI) GetBlock(mode string, value string) (*pb.Block, error) {
	switch mode {
	case "HEIGHT":
		height, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("wrong block number: %s", value)
		}
		return b.bxh.Ledger.GetBlock(height)
	case "HASH":
		return b.bxh.Ledger.GetBlockByHash(types.String2Hash(value))
	default:
		return nil, fmt.Errorf("wrong args about getting block: %s", mode)
	}
}

func (b *BrokerAPI) GetBlocks(start uint64, end uint64) ([]*pb.Block, error) {
	meta := b.bxh.Ledger.GetChainMeta()

	var blocks []*pb.Block
	if meta.Height < end {
		end = meta.Height
	}
	for i := start; i > 0 && i <= end; i++ {
		b, err := b.GetBlock("HEIGHT", strconv.Itoa(int(i)))
		if err != nil {
			continue
		}
		blocks = append(blocks, b)
	}

	return blocks, nil
}

func (b *BrokerAPI) RemovePier(key string) {
	b.bxh.Router.RemovePier(key)
}

func (b *BrokerAPI) OrderReady() bool {
	return b.bxh.Order.Ready()
}

func (b *BrokerAPI) FetchAssetExchangeSignsFromOtherPeers(id string) map[string][]byte {
	var (
		result = make(map[string][]byte)
		wg     = sync.WaitGroup{}
		lock   = sync.Mutex{}
	)

	wg.Add(len(b.bxh.PeerMgr.OtherPeers()))
	for pid := range b.bxh.PeerMgr.OtherPeers() {
		go func(pid uint64, result map[string][]byte, wg *sync.WaitGroup, lock *sync.Mutex) {
			address, sign, err := b.requestAssetExchangeSignFromPeer(pid, id)
			if err != nil {
				b.logger.WithFields(logrus.Fields{
					"pid": pid,
					"err": err.Error(),
				}).Warnf("Get asset exchange sign with error")
			} else {
				lock.Lock()
				result[address] = sign
				lock.Unlock()
			}
			wg.Done()
		}(pid, result, &wg, &lock)
	}

	wg.Wait()

	return result
}

func (b *BrokerAPI) requestAssetExchangeSignFromPeer(peerId uint64, assetExchangeId string) (string, []byte, error) {
	req := pb.Message{
		Type: pb.Message_FETCH_ASSET_EXCHANEG_SIGN,
		Data: []byte(assetExchangeId),
	}

	resp, err := b.bxh.PeerMgr.Send(peerId, &req)
	if err != nil {
		return "", nil, err
	}
	if resp == nil || resp.Type != pb.Message_FETCH_ASSET_EXCHANEG_SIGN {
		return "", nil, fmt.Errorf("invalid asset exchange sign resp")
	}

	data := model.MerkleWrapperSign{}
	if err := data.Unmarshal(resp.Data); err != nil {
		return "", nil, err
	}

	return data.Address, data.Signature, nil
}

func (b *BrokerAPI) GetAssetExchangeSign(id string) (string, []byte, error) {
	ok, record := b.bxh.Ledger.GetState(constant.AssetExchangeContractAddr.Address(), []byte(contracts.AssetExchangeKey(id)))
	if !ok {
		return "", nil, fmt.Errorf("cannot find asset exchange record with id %s", id)
	}

	aer := contracts.AssetExchangeRecord{}
	if err := json.Unmarshal(record, &aer); err != nil {
		return "", nil, err
	}

	result := fmt.Sprintf("%s-%d", id, aer.Status)

	key := b.bxh.GetPrivKey()
	sign, err := key.PrivKey.Sign([]byte(result))
	if err != nil {
		return "", nil, fmt.Errorf("fetch asset exchange sign: %w", err)
	}

	return key.Address, sign, nil
}
