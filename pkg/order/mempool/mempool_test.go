package mempool

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/google/btree"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/require"
)

var (
	InterchainContractAddr = types.String2Address("000000000000000000000000000000000000000a")
	appchains              = []string{
		"0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997",
		"0xa8ae1bbc1105944a84a71b89056930d951d420fe",
		"0x929545f44692178edb7fa468b44c5351596184ba",
		"0x7368022e6659236983eb959b8a1fa22577d48294",
	}
)

func TestCoreMemPool_RecvTransactions(t *testing.T) {
	readyTxs := make([]*pb.Transaction, 0)
	nonreadyTxs := make([]*pb.Transaction, 0)
	txIndex := make(map[string]*pb.Transaction)
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)
	pubKey := privKey.PublicKey()
	addr, err := pubKey.Address()
	require.Nil(t, err)

	sort.Strings(appchains)
	readyTxsLen := 2
	for _, appchain := range appchains {
		for i := 1; i <= readyTxsLen; i++ {
			readyTxs = append(readyTxs, mockTxhelper(t, txIndex, appchain, uint64(i)))
			// add unready txs
			nonreadyTxs = append(nonreadyTxs, mockTxhelper(t, txIndex, appchain, uint64(i+readyTxsLen+1)))
		}
	}

	// set timestamp and signature for txs
	for _, tx := range readyTxs {
		addSigAndTime(t, tx, addr, privKey)
	}
	for _, tx := range nonreadyTxs {
		addSigAndTime(t, tx, addr, privKey)
	}

	// shuffle tx order
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(readyTxs), func(i, j int) {
		readyTxs[i], readyTxs[j] = readyTxs[j], readyTxs[i]
	})
	memPool := New()
	require.Nil(t, memPool.RecvTransactions(readyTxs))
	require.Nil(t, memPool.RecvTransactions(nonreadyTxs))

	// check if all txs are indexed in memPool.allTxs
	// and if all txs are indexed by its account and nonce
	require.Equal(t, len(appchains), len(memPool.transactionStore.allTxs))
	checkAllTxs(t, memPool, txIndex, readyTxsLen, readyTxsLen)
	checkHashMap(t, memPool, true, readyTxs, nonreadyTxs)

	// check if priorityIndex is correctly recorded
	require.Equal(t, len(readyTxs), memPool.transactionStore.priorityIndex.data.Len())
	for _, tx := range readyTxs {
		ok := memPool.transactionStore.priorityIndex.data.Has(makeKey(tx))
		require.True(t, ok)
		ok = memPool.transactionStore.parkingLotIndex.data.Has(makeKey(tx))
		require.True(t, !ok)
	}

	// check if parkingLotIndex is correctly recorded
	require.Equal(t, len(nonreadyTxs), memPool.transactionStore.parkingLotIndex.data.Len())
	for _, tx := range nonreadyTxs {
		ok := memPool.transactionStore.parkingLotIndex.data.Has(makeKey(tx))
		require.True(t, ok)
		ok = memPool.transactionStore.priorityIndex.data.Has(makeKey(tx))
		require.True(t, !ok)
	}

	// add the missing tx for each appchain
	missingTxs := make([]*pb.Transaction, 0, len(appchains))
	for _, appchain := range appchains {
		missingTxs = append(missingTxs, mockTxhelper(t, txIndex, appchain, uint64(readyTxsLen+1)))
	}
	for _, tx := range missingTxs {
		addSigAndTime(t, tx, addr, privKey)
	}

	require.Nil(t, memPool.RecvTransactions(missingTxs))

	// check if parkingLotIndex is empty now
	require.Equal(t, 0, memPool.transactionStore.parkingLotIndex.data.Len())
	// check if priorityIndex has received missingTxs and txs from original parkingLotIndex
	for _, tx := range missingTxs {
		ok := memPool.transactionStore.priorityIndex.data.Has(makeKey(tx))
		require.True(t, ok)
	}
	for _, tx := range nonreadyTxs {
		ok := memPool.transactionStore.priorityIndex.data.Has(makeKey(tx))
		require.True(t, ok)
	}
	checkHashMap(t, memPool, true, readyTxs, nonreadyTxs, missingTxs)
}

func TestCoreMemPool_RecvTransactions_Margin(t *testing.T) {
	readyTxs := make([]*pb.Transaction, 0)
	identicalNonceTxs := make([]*pb.Transaction, 0)
	replayedTxs := make([]*pb.Transaction, 0)
	readyTxIndex := make(map[string]*pb.Transaction)
	identicalNonceTxIndex := make(map[string]*pb.Transaction)
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)
	pubKey := privKey.PublicKey()
	addr, err := pubKey.Address()
	require.Nil(t, err)

	sort.Strings(appchains)
	readyTxsLen := 2
	for _, appchain := range appchains {
		for i := 1; i <= readyTxsLen; i++ {
			tx := mockTxhelper(t, readyTxIndex, appchain, uint64(i))
			readyTxs = append(readyTxs, tx)
			// add tx with same index but different content
			identicalNonceTx := mockTxhelper(t, identicalNonceTxIndex, appchain, uint64(i))
			identicalNonceTxs = append(identicalNonceTxs, identicalNonceTx)
		}
	}

	// set timestamp and signature for txs
	for _, tx := range readyTxs {
		addSigAndTime(t, tx, addr, privKey)
		// add repeated txs
		replayedTxs = append(replayedTxs, tx)
	}
	for _, tx := range identicalNonceTxs {
		addSigAndTime(t, tx, addr, privKey)
	}

	memPool := New()
	require.Nil(t, memPool.RecvTransactions(readyTxs))
	require.NotNil(t, memPool.RecvTransactions(replayedTxs))
	err = memPool.RecvTransactions(identicalNonceTxs)
	require.NotNil(t, err)

	require.Equal(t, len(appchains), len(memPool.transactionStore.allTxs))
	checkAllTxs(t, memPool, readyTxIndex, readyTxsLen, 0)
	checkHashMap(t, memPool, true, readyTxs)
	checkHashMap(t, memPool, false, identicalNonceTxs)
}

func checkAllTxs(t *testing.T, memPool *CoreMemPool,
	txIndex map[string]*pb.Transaction, readyTxsLen, nonReadyTxLen int) {
	for _, appchain := range appchains {
		idx := uint64(1)
		accountAddr := fmt.Sprintf("%s-%s", appchain, appchain)

		txMap, ok := memPool.transactionStore.allTxs[accountAddr]
		require.True(t, ok)
		require.NotNil(t, txMap.index)
		require.Equal(t, readyTxsLen+nonReadyTxLen, txMap.index.data.Len())
		require.Equal(t, readyTxsLen+nonReadyTxLen, len(txMap.items))
		txMap.index.data.Ascend(func(i btree.Item) bool {
			orderedKey := i.(*orderedQueueKey)
			if idx <= uint64(readyTxsLen) {
				require.Equal(t, orderedKey.seqNo, idx)
			} else {
				require.Equal(t, orderedKey.seqNo, idx+1)
			}
			require.Equal(t, orderedKey.accountAddress, accountAddr)

			ibtpID := fmt.Sprintf("%s-%s-%d", appchain, appchain, orderedKey.seqNo)
			require.Equal(t, txIndex[ibtpID], txMap.items[orderedKey.seqNo])
			idx++
			return true
		})
	}
}

func checkHashMap(t *testing.T, memPool *CoreMemPool, expectedStatus bool, txsSlice ...[]*pb.Transaction) {
	for _, txs := range txsSlice {
		for _, tx := range txs {
			_, ok := memPool.transactionStore.txHashMap[tx.TransactionHash.Hex()]
			require.Equal(t, expectedStatus, ok)
		}
	}
}

func mockTxhelper(t *testing.T, txIndex map[string]*pb.Transaction, appchainAddr string, index uint64) *pb.Transaction {
	ibtp := mockIBTP(t, appchainAddr, appchainAddr, index)
	tx := mockInterchainTx(t, ibtp)
	txIndex[ibtp.ID()] = tx
	return tx
}

func addSigAndTime(t *testing.T, tx *pb.Transaction, addr types.Address, privKey crypto.PrivateKey) {
	tx.Timestamp = time.Now().UnixNano()
	tx.From = addr
	sig, err := privKey.Sign(tx.SignHash().Bytes())
	tx.Signature = sig
	require.Nil(t, err)
	tx.TransactionHash = tx.Hash()
}

func mockInterchainTx(t *testing.T, ibtp *pb.IBTP) *pb.Transaction {
	ib, err := ibtp.Marshal()
	require.Nil(t, err)

	ipd := &pb.InvokePayload{
		Method: "HandleIBTP",
		Args:   []*pb.Arg{{Value: ib}},
	}
	pd, err := ipd.Marshal()
	require.Nil(t, err)

	data := &pb.TransactionData{
		VmType:  pb.TransactionData_BVM,
		Type:    pb.TransactionData_INVOKE,
		Payload: pd,
	}

	return &pb.Transaction{
		To:    InterchainContractAddr,
		Nonce: int64(ibtp.Index),
		Data:  data,
		Extra: []byte(fmt.Sprintf("%s-%s", ibtp.From, ibtp.To)),
	}
}

func mockIBTP(t *testing.T, from, to string, nonce uint64) *pb.IBTP {
	content := pb.Content{
		SrcContractId: from,
		DstContractId: from,
		Func:          "interchainget",
		Args:          [][]byte{[]byte("Alice"), []byte("10")},
	}

	bytes, err := content.Marshal()
	require.Nil(t, err)

	ibtppd, err := json.Marshal(pb.Payload{
		Encrypted: false,
		Content:   bytes,
	})
	require.Nil(t, err)

	return &pb.IBTP{
		From:      from,
		To:        to,
		Payload:   ibtppd,
		Index:     nonce,
		Type:      pb.IBTP_INTERCHAIN,
		Timestamp: time.Now().UnixNano(),
	}
}
