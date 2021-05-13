package coreapi

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/model"
	"github.com/meshplus/bitxid"
	"github.com/sirupsen/logrus"
)

type BrokerAPI CoreAPI

var _ api.BrokerAPI = (*BrokerAPI)(nil)

func (b *BrokerAPI) HandleTransaction(tx pb.Transaction) error {
	if tx.GetHash() == nil {
		return fmt.Errorf("transaction hash is nil")
	}

	b.logger.WithFields(logrus.Fields{
		"hash": tx.GetHash().String(),
	}).Debugf("Receive tx")

	go func() {
		if err := b.bxh.Order.Prepare(tx); err != nil {
			b.logger.Error(err)
		}
	}()

	return nil
}

func (b *BrokerAPI) HandleView(tx pb.Transaction) (*pb.Receipt, error) {
	if tx.GetHash() == nil {
		return nil, fmt.Errorf("transaction hash is nil")
	}

	b.logger.WithFields(logrus.Fields{
		"hash": tx.GetHash().String(),
	}).Debugf("Receive view")

	receipts := b.bxh.ViewExecutor.ApplyReadonlyTransactions([]pb.Transaction{tx})

	return receipts[0], nil
}

func (b *BrokerAPI) GetTransaction(hash *types.Hash) (pb.Transaction, error) {
	return b.bxh.Ledger.GetTransaction(hash)
}

func (b *BrokerAPI) GetTransactionMeta(hash *types.Hash) (*pb.TransactionMeta, error) {
	return b.bxh.Ledger.GetTransactionMeta(hash)
}

func (b *BrokerAPI) GetReceipt(hash *types.Hash) (*pb.Receipt, error) {
	return b.bxh.Ledger.GetReceipt(hash)
}

func (b *BrokerAPI) AddPier(did bitxid.DID, pierID string, isUnion bool) (chan *pb.InterchainTxWrappers, error) {
	return b.bxh.Router.AddPier(did, pierID, isUnion)
}

func (b *BrokerAPI) GetBlockHeader(begin, end uint64, ch chan<- *pb.BlockHeader) error {
	return b.bxh.Router.GetBlockHeader(begin, end, ch)
}

func (b *BrokerAPI) GetInterchainTxWrappers(did string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error {
	return b.bxh.Router.GetInterchainTxWrappers(did, begin, end, ch)
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
		hash := types.NewHashByStr(value)
		if hash == nil {
			return nil, fmt.Errorf("invalid format of block hash for querying block")
		}
		return b.bxh.Ledger.GetBlockByHash(hash)
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

func (b *BrokerAPI) GetBlockHeaders(start uint64, end uint64) ([]*pb.BlockHeader, error) {
	meta := b.bxh.Ledger.GetChainMeta()

	var blockHeaders []*pb.BlockHeader
	if meta.Height < end {
		end = meta.Height
	}
	for i := start; i > 0 && i <= end; i++ {
		b, err := b.GetBlock("HEIGHT", strconv.Itoa(int(i)))
		if err != nil {
			continue
		}
		blockHeaders = append(blockHeaders, b.BlockHeader)
	}

	return blockHeaders, nil
}

func (b *BrokerAPI) RemovePier(did bitxid.DID, pierID string, isUnion bool) {
	b.bxh.Router.RemovePier(did, pierID, isUnion)
}

func (b *BrokerAPI) OrderReady() error {
	return b.bxh.Order.Ready()
}

func (b *BrokerAPI) FetchSignsFromOtherPeers(id string, typ pb.GetMultiSignsRequest_Type) map[string][]byte {
	var (
		result = make(map[string][]byte)
		wg     = sync.WaitGroup{}
		lock   = sync.Mutex{}
	)

	wg.Add(len(b.bxh.PeerMgr.OtherPeers()))
	for pid := range b.bxh.PeerMgr.OtherPeers() {
		go func(pid uint64, result map[string][]byte, wg *sync.WaitGroup, lock *sync.Mutex) {
			var (
				address string
				sign    []byte
				err     error
			)
			switch typ {
			case pb.GetMultiSignsRequest_ASSET_EXCHANGE:
				address, sign, err = b.requestAssetExchangeSignFromPeer(pid, id)
			case pb.GetMultiSignsRequest_IBTP:
				address, sign, err = b.requestIBTPSignPeer(pid, id)
			case pb.GetMultiSignsRequest_BLOCK_HEADER:
				address, sign, err = b.requestBlockHeaderSignFromPeer(pid, id)
			}

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
	if resp == nil || resp.Type != pb.Message_FETCH_ASSET_EXCHANGE_SIGN_ACK {
		return "", nil, fmt.Errorf("invalid asset exchange sign resp")
	}

	data := model.MerkleWrapperSign{}
	if err := data.Unmarshal(resp.Data); err != nil {
		return "", nil, err
	}

	return data.Address, data.Signature, nil
}

func (b *BrokerAPI) requestIBTPSignPeer(pid uint64, ibtpHash string) (string, []byte, error) {
	req := pb.Message{
		Type: pb.Message_FETCH_IBTP_SIGN,
		Data: []byte(ibtpHash),
	}

	resp, err := b.bxh.PeerMgr.Send(pid, &req)
	if err != nil {
		return "", nil, err
	}
	if resp == nil || resp.Type != pb.Message_FETCH_IBTP_SIGN_ACK {
		return "", nil, fmt.Errorf("invalid fetch ibtp sign resp")
	}

	data := model.MerkleWrapperSign{}
	if err := data.Unmarshal(resp.Data); err != nil {
		return "", nil, err
	}

	return data.Address, data.Signature, nil
}

func (b *BrokerAPI) requestBlockHeaderSignFromPeer(pid uint64, height string) (string, []byte, error) {
	req := pb.Message{
		Type: pb.Message_FETCH_BLOCK_SIGN,
		Data: []byte(height),
	}

	resp, err := b.bxh.PeerMgr.Send(pid, &req)
	if err != nil {
		return "", nil, err
	}

	if resp == nil || resp.Type != pb.Message_FETCH_BLOCK_SIGN_ACK {
		return "", nil, fmt.Errorf("invalid fetch block header sign resp")
	}

	data := model.MerkleWrapperSign{}
	if err := data.Unmarshal(resp.Data); err != nil {
		return "", nil, err
	}

	return data.Address, data.Signature, nil
}

func (b *BrokerAPI) GetSign(content string, typ pb.GetMultiSignsRequest_Type) (string, []byte, error) {
	switch typ {
	case pb.GetMultiSignsRequest_ASSET_EXCHANGE:
		id := content
		ok, record := b.bxh.Ledger.GetState(constant.AssetExchangeContractAddr.Address(), []byte(contracts.AssetExchangeKey(id)))
		if !ok {
			return "", nil, fmt.Errorf("cannot find asset exchange record with id %s", id)
		}

		aer := contracts.AssetExchangeRecord{}
		if err := json.Unmarshal(record, &aer); err != nil {
			return "", nil, err
		}

		addr, sign, err := b.getSign(fmt.Sprintf("%s-%d", id, aer.Status))
		if err != nil {
			return "", nil, fmt.Errorf("fetch asset exchange sign: %w", err)
		}
		return addr, sign, nil
	case pb.GetMultiSignsRequest_IBTP:
		addr, sign, err := b.getSign(content)
		if err != nil {
			return "", nil, fmt.Errorf("get ibtp sign: %w", err)
		}
		return addr, sign, nil
	case pb.GetMultiSignsRequest_BLOCK_HEADER:
		height, err := strconv.ParseUint(content, 10, 64)
		if err != nil {
			return "", nil, fmt.Errorf("get block header sign: %w", err)
		}

		sign, err := b.bxh.Ledger.GetBlockSign(height)
		if err != nil {
			return "", nil, fmt.Errorf("get block sign: %w", err)
		}

		return b.bxh.GetPrivKey().Address, sign, nil
	default:
		return "", nil, fmt.Errorf("unsupported get sign type")
	}

}

func (b *BrokerAPI) getSign(content string) (string, []byte, error) {
	hash := sha256.Sum256([]byte(content))
	key := b.bxh.GetPrivKey()
	sign, err := key.PrivKey.Sign(hash[:])
	if err != nil {
		return "", nil, fmt.Errorf("bitxhub sign: %w", err)
	}
	return key.Address, sign, nil
}

func (b BrokerAPI) GetPendingNonceByAccount(account string) uint64 {
	return b.bxh.Order.GetPendingNonceByAccount(account)
}

func (b BrokerAPI) DelVPNode(delID uint64) error {
	return b.bxh.Order.DelNode(delID)
}

func (b BrokerAPI) GetPendingTransactions(max int) []pb.Transaction {
	// TODO
	return nil
}

func (b BrokerAPI) GetPoolTransaction(hash *types.Hash) pb.Transaction {
	// TODO

	return nil
}
