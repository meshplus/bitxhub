package jsonrpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/mux"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/sirupsen/logrus"
)

type ChainBrokerService struct {
	config  *repo.Config
	genesis *repo.Genesis
	api     api.CoreAPI
	server  *rpc.Server
	logger  logrus.FieldLogger

	ctx    context.Context
	cancel context.CancelFunc
}

func NewChainBrokerService(coreAPI api.CoreAPI, config *repo.Config) (*ChainBrokerService, error) {
	server := rpc.NewServer()

	logger := loggers.Logger(loggers.API)

	apis, err := GetAPIs(config, coreAPI, logger)
	if err != nil {
		return nil, err
	}

	// Register all the APIs exposed by the namespace services
	for _, api := range apis {
		if err := server.RegisterName(api.Namespace, api.Service); err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ChainBrokerService{
		logger: logger,
		config: config,
		api:    coreAPI,
		server: server,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (cbs *ChainBrokerService) Start() error {
	router := mux.NewRouter()
	router.Handle("/", cbs.server)

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cbs.config.Port.JsonRpc), router); err != nil {
			cbs.logger.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Failed to start JSON_RPC service")
			return
		}

		cbs.logger.WithFields(logrus.Fields{
			"port": cbs.config.Port.JsonRpc,
		}).Info("JSON-RPC service started")
	}()

	return nil
}

func (cbs *ChainBrokerService) Stop() error {
	cbs.cancel()

	cbs.logger.Info("GRPC service stopped")

	return nil
}
