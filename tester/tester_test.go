package tester

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/api/jsonrpc"
	"github.com/meshplus/bitxhub/internal/app"
	"github.com/meshplus/bitxhub/internal/coreapi"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/internal/router"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/order/etcdraft"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestTester(t *testing.T) {
	node1 := setupNode(t, "./test_data/config/node1")
	node2 := setupNode(t, "./test_data/config/node2")
	node3 := setupNode(t, "./test_data/config/node3")
	node4 := setupNode(t, "./test_data/config/node4")

	for {
		err1 := node1.Broker().OrderReady()
		err2 := node2.Broker().OrderReady()
		err3 := node3.Broker().OrderReady()
		err4 := node4.Broker().OrderReady()
		if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	keyPath1 := filepath.Join("./test_data/config/node1/key.json")
	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
	require.Nil(t, err)
	fromAdmin1, err := priAdmin1.PublicKey().Address()
	require.Nil(t, err)
	adminNonce1 := node1.Broker().GetPendingNonceByAccount(fromAdmin1.String())
	require.Nil(t, err)
	keyPath2 := filepath.Join("./test_data/config/node2/key.json")
	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
	require.Nil(t, err)
	fromAdmin2, err := priAdmin2.PublicKey().Address()
	require.Nil(t, err)
	keyPath3 := filepath.Join("./test_data/config/node3/key.json")
	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
	require.Nil(t, err)
	fromAdmin3, err := priAdmin3.PublicKey().Address()
	require.Nil(t, err)
	keyPath4 := filepath.Join("./test_data/config/node4/key.json")
	priAdmin4, err := asym.RestorePrivateKey(keyPath4, "bitxhub")
	require.Nil(t, err)
	fromAdmin4, err := priAdmin4.PublicKey().Address()
	require.Nil(t, err)
	// init registry first
	adminDid := genUniqueRelaychainDID(fromAdmin1.String())
	args := []*pb.Arg{
		pb.String(adminDid),
	}
	ret, err := invokeBVMContract(node1, priAdmin1, adminNonce1, constant.MethodRegistryContractAddr.Address(), "Init", args...)
	require.Nil(t, err)
	require.True(t, ret.IsSuccess(), string(ret.Ret))
	adminNonce1++
	// set admin for method registry for other nodes
	args = []*pb.Arg{
		pb.String(adminDid),
		pb.String(genUniqueRelaychainDID(fromAdmin2.String())),
	}
	ret, err = invokeBVMContract(node1, priAdmin1, adminNonce1, constant.MethodRegistryContractAddr.Address(), "AddAdmin", args...)
	require.Nil(t, err)
	require.True(t, ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	args = []*pb.Arg{
		pb.String(adminDid),
		pb.String(genUniqueRelaychainDID(fromAdmin3.String())),
	}
	ret, err = invokeBVMContract(node1, priAdmin1, adminNonce1, constant.MethodRegistryContractAddr.Address(), "AddAdmin", args...)
	require.Nil(t, err)
	require.True(t, ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	args = []*pb.Arg{
		pb.String(adminDid),
		pb.String(genUniqueRelaychainDID(fromAdmin4.String())),
	}
	ret, err = invokeBVMContract(node1, priAdmin1, adminNonce1, constant.MethodRegistryContractAddr.Address(), "AddAdmin", args...)
	require.Nil(t, err)
	require.True(t, ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	suite.Run(t, &API{api: node1})
	suite.Run(t, &RegisterAppchain{api: node2})
	suite.Run(t, &Interchain{api: node3})
	suite.Run(t, &Role{api: node4})
	suite.Run(t, &Store{api: node1})
	suite.Run(t, &Governance{api: node2})
}

func setupNode(t *testing.T, path string) api.CoreAPI {
	cleanStorage(path)
	repoRoot, err := repo.PathRootWithDefault(path)
	require.Nil(t, err)

	repo, err := repo.Load(repoRoot)
	require.Nil(t, err)

	loggers.Initialize(repo.Config)

	bxh, err := newTesterBitXHub(repo)
	require.Nil(t, err)

	api, err := coreapi.New(bxh)
	require.Nil(t, err)

	// start json-rpc service
	cbs, err := jsonrpc.NewChainBrokerService(api, repo.Config)
	require.Nil(t, err)

	err = cbs.Start()
	require.Nil(t, err)

	go func() {
		err = bxh.Start()
		require.Nil(t, err)
	}()

	return api
}

func newTesterBitXHub(rep *repo.Repo) (*app.BitXHub, error) {
	repoRoot := rep.Config.RepoRoot

	bxh, err := app.GenerateBitXHubWithoutOrder(rep)
	if err != nil {
		return nil, err
	}

	chainMeta := bxh.Ledger.GetChainMeta()

	m := rep.NetworkConfig.GetVpInfos()

	order, err := etcdraft.NewNode(
		order.WithRepoRoot(repoRoot),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithPluginPath(rep.Config.Plugin),
		order.WithNodes(m),
		order.WithID(rep.NetworkConfig.ID),
		order.WithIsNew(rep.NetworkConfig.New),
		order.WithPeerManager(bxh.PeerMgr),
		order.WithLogger(loggers.Logger(loggers.Order)),
		order.WithApplied(chainMeta.Height),
		order.WithDigest(chainMeta.BlockHash.String()),
		order.WithGetChainMetaFunc(bxh.Ledger.GetChainMeta),
		order.WithGetBlockByHeightFunc(bxh.Ledger.GetBlock),
		order.WithGetAccountNonceFunc(bxh.Ledger.GetNonce),
	)

	if err != nil {
		return nil, err
	}

	r, err := router.New(loggers.Logger(loggers.Router), rep, bxh.Ledger, bxh.PeerMgr, order.Quorum())
	if err != nil {
		return nil, fmt.Errorf("create InterchainRouter: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	bxh.Ctx = ctx
	bxh.Cancel = cancel
	bxh.Order = order
	bxh.Router = r

	return bxh, nil
}

func cleanStorage(basePath string) {
	filePath := path.Join(basePath, "storage")
	err := os.RemoveAll(filePath)
	if err != nil {
		fmt.Printf("Clean storage failed, error: %s", err.Error())
		return
	}
}
