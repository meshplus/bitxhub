package tester

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/meshplus/bitxhub/internal/app"
	"github.com/meshplus/bitxhub/internal/coreapi"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/require"
)

func TestTester(t *testing.T) {
	node1 := setupNode(t, "./test_data/config/node1")
	node2 := setupNode(t, "./test_data/config/node2")
	node3 := setupNode(t, "./test_data/config/node3")
	node4 := setupNode(t, "./test_data/config/node4")

	for {
		if node1.Broker().OrderReady() &&
			node2.Broker().OrderReady() &&
			node3.Broker().OrderReady() &&
			node4.Broker().OrderReady() {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	suite.Run(t, &API{api: node1})
	suite.Run(t, &RegisterAppchain{api: node2})
	suite.Run(t, &Interchain{api: node3})
	suite.Run(t, &Role{api: node4})
	suite.Run(t, &Store{api: node1})
}

func setupNode(t *testing.T, path string) api.CoreAPI {
	repoRoot, err := repo.PathRootWithDefault(path)
	require.Nil(t, err)

	repo, err := repo.Load(repoRoot)
	require.Nil(t, err)

	loggers.Initialize(repo.Config)

	bxh, err := app.NewTesterBitXHub(repo)
	require.Nil(t, err)

	api, err := coreapi.New(bxh)
	require.Nil(t, err)

	go func() {
		err = bxh.Start()
		require.Nil(t, err)
	}()

	return api
}
