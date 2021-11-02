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
	logger := loggers.Logger(loggers.API)

	ctx, cancel := context.WithCancel(context.Background())

	cbs := &ChainBrokerService{
		logger: logger,
		config: config,
		api:    coreAPI,
		ctx:    ctx,
		cancel: cancel,
	}

	if err := cbs.init(); err != nil {
		return nil, fmt.Errorf("init chain broker service failed: %w", err)
	}

	return cbs, nil
}

func (cbs *ChainBrokerService) init() error {
	cbs.server = rpc.NewServer()

	apis, err := GetAPIs(cbs.config, cbs.api, cbs.logger)
	if err != nil {
		return fmt.Errorf("get apis failed: %w", err)
	}

	// Register all the APIs exposed by the namespace services
	for _, api := range apis {
		if err := cbs.server.RegisterName(api.Namespace, api.Service); err != nil {
			return fmt.Errorf("register name %s for service %v failed: %w", api.Namespace, api.Service, err)
		}
	}

	return nil
}

func (cbs *ChainBrokerService) Start() error {
	router := mux.NewRouter()
	router.Handle("/", cbs.server)

	go func() {
		cbs.logger.WithFields(logrus.Fields{
			"port": cbs.config.Port.JsonRpc,
		}).Info("JSON-RPC service started")

		if err := http.ListenAndServe(fmt.Sprintf(":%d", cbs.config.Port.JsonRpc), router); err != nil {
			cbs.logger.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Errorf("Failed to start JSON_RPC service: %s", err.Error())
			return
		}
	}()

	return nil
}

func (cbs *ChainBrokerService) Stop() error {
	cbs.cancel()

	cbs.server.Stop()

	cbs.logger.Info("JSON-RPC service stopped")

	return nil
}

func (cbs *ChainBrokerService) ReConfig(config *repo.Config) error {
	if cbs.config.JsonRpc != config.JsonRpc {
		if err := cbs.Stop(); err != nil {
			return fmt.Errorf("stop chain broker service failed: %w", err)
		}

		cbs.config.JsonRpc = config.JsonRpc

		if err := cbs.init(); err != nil {
			return fmt.Errorf("init chain broker service failed: %w", err)
		}

		if err := cbs.Start(); err != nil {
			return fmt.Errorf("start chain broker service failed: %w", err)
		}
	}

	return nil
}
