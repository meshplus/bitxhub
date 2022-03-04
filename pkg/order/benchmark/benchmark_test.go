package benchmark

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bitxhub/bitxhub-order-rbft/rbft"
	"github.com/meshplus/bitxhub-core/order"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/require"
)

var (
	txCount int
)

func TestMulti_Rbft_Node(t *testing.T) {
	peerCnt := 4
	swarms, nodes := newSwarms(t, peerCnt, true)
	defer stopSwarms(t, swarms)

	repoRoot, err := ioutil.TempDir("", "nodes")
	defer os.RemoveAll(repoRoot)

	fileData, err := ioutil.ReadFile("../../../config/order.toml")
	require.Nil(t, err)

	orders := make([]order.Order, 0)
	exs := make([]*mockExecutor, 0)
	for i := 0; i < peerCnt; i++ {
		nodePath := fmt.Sprintf("node%d", i)
		nodeRepo := filepath.Join(repoRoot, nodePath)
		err := os.Mkdir(nodeRepo, 0744)
		require.Nil(t, err)
		orderPath := filepath.Join(nodeRepo, "order.toml")
		err = ioutil.WriteFile(orderPath, fileData, 0744)
		require.Nil(t, err)

		ID := i + 1
		rbft, err := rbft.NewNode(
			order.WithRepoRoot(nodeRepo),
			order.WithID(uint64(ID)),
			order.WithNodes(nodes),
			order.WithPeerManager(swarms[i]),
			order.WithStoragePath(repo.GetStoragePath(nodeRepo, "order")),
			order.WithLogger(log.NewWithModule("consensus")),
			order.WithGetBlockByHeightFunc(nil),
			order.WithApplied(1),
			order.WithGetAccountNonceFunc(func(address *types.Address) uint64 {
				return 0
			}),
		)
		require.Nil(t, err)
		err = rbft.Start()
		require.Nil(t, err)
		orders = append(orders, rbft)
		ex := newMockExecutor(i)
		exs = append(exs, ex)
		go listen(t, rbft, swarms[i], ex)
	}

	for {
		time.Sleep(2 * time.Second)
		for i, _ := range orders {
			err = orders[i].Ready()
			if err == nil {
				go listenCommit(t, orders[i], exs[i])
			}
		}
		break
	}
	time.Sleep(2 * time.Second)
	go sendTx(t, 1000000, orders[0])
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			fmt.Printf("!!!!!!!!!!!!cal tps,tps is %d\n", txCount)
			txCount = 0
		case block := <-exs[0].endBlockC:
			txCount += len(block.Transactions.Transactions)
		}
	}

}

func sendTx(t *testing.T, count int, node order.Order) {
	privKey, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	for i := 0; i < count; i++ {
		tx := generateTx(privKey, uint64(i))
		err := node.Prepare(tx)
		require.Nil(t, err)
		//fmt.Printf("sendTx%d succuss\n", i)
	}
}

func listenCommit(t *testing.T, node order.Order, ex *mockExecutor) {
	for {
		select {
		case commitEvent := <-node.Commit():
			//blockHeight := commitEvent.Block.BlockHeader.Number
			//fmt.Printf("!!!!!!receive node%d block %d", ex.id, blockHeight)
			txHashList := make([]*types.Hash, 0)
			for _, tx := range commitEvent.Block.Transactions.Transactions {
				txHashList = append(txHashList, tx.GetHash())
			}

			block := &pb.Block{}
			block = commitEvent.Block
			block.BlockHash = block.Hash()
			ex.blockC <- &executedEvent{
				Block:      block,
				TxHashList: txHashList,
			}
		}
	}
}
