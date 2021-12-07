package grpc

import (
	"context"
	"fmt"
	"net"
	"path/filepath"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/ratelimiter"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const MAX_MESSAGE_SIZE = 256 * 1024 * 1024

type ChainBrokerService struct {
	config  *repo.Config
	genesis *repo.Genesis
	api     api.CoreAPI
	server  *grpc.Server
	logger  logrus.FieldLogger

	ctx    context.Context
	cancel context.CancelFunc
}

func NewChainBrokerService(api api.CoreAPI, config *repo.Config, genesis *repo.Genesis) (*ChainBrokerService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cbs := &ChainBrokerService{
		logger:  loggers.Logger(loggers.API),
		config:  config,
		genesis: genesis,
		api:     api,
		ctx:     ctx,
		cancel:  cancel,
	}

	if err := cbs.init(); err != nil {
		return nil, err
	}
	return cbs, nil
}

func (cbs *ChainBrokerService) init() error {
	config := cbs.config
	limiter := cbs.config.Limiter
	rateLimiter := ratelimiter.NewRateLimiterWithQuantum(limiter.Interval, limiter.Capacity, limiter.Quantum)

	grpcOpts := []grpc.ServerOption{
		grpc_middleware.WithUnaryServerChain(ratelimit.UnaryServerInterceptor(rateLimiter), grpc_prometheus.UnaryServerInterceptor),
		grpc_middleware.WithStreamServerChain(ratelimit.StreamServerInterceptor(rateLimiter), grpc_prometheus.StreamServerInterceptor),
		grpc.MaxConcurrentStreams(1000),
		grpc.InitialWindowSize(10 * 1024 * 1024),
		grpc.InitialConnWindowSize(100 * 1024 * 1024),
		grpc.MaxSendMsgSize(MAX_MESSAGE_SIZE),
		grpc.MaxRecvMsgSize(MAX_MESSAGE_SIZE),
	}

	if config.Security.EnableTLS {
		pemFilePath := filepath.Join(config.RepoRoot, config.Security.PemFilePath)
		serverKeyPath := filepath.Join(config.RepoRoot, config.Security.ServerKeyPath)
		cred, err := credentials.NewServerTLSFromFile(pemFilePath, serverKeyPath)
		if err != nil {
			return err
		}
		grpcOpts = append(grpcOpts, grpc.Creds(cred))
	}
	cbs.server = grpc.NewServer(grpcOpts...)
	return nil
}

func (cbs *ChainBrokerService) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cbs.config.Port.Grpc))
	if err != nil {
		return err
	}

	pb.RegisterChainBrokerServer(cbs.server, cbs)

	cbs.logger.WithFields(logrus.Fields{
		"port": cbs.config.Port.Grpc,
	}).Info("GRPC service started")

	go func() {
		err := cbs.server.Serve(lis)
		if err != nil {
			cbs.logger.Error(err)
		}
	}()

	return nil
}

func (cbs *ChainBrokerService) Stop() error {
	cbs.cancel()

	cbs.server.Stop()

	cbs.logger.Info("GRPC service stopped")

	return nil
}

func (cbs *ChainBrokerService) ReConfig(config *repo.Config) error {
	if cbs.config.Limiter.Capacity != config.Limiter.Capacity ||
		cbs.config.Limiter.Interval.String() != config.Limiter.Interval.String() ||
		cbs.config.Limiter.Quantum != config.Limiter.Quantum ||
		cbs.config.Security.ServerKeyPath != config.Security.ServerKeyPath ||
		cbs.config.Security.PemFilePath != config.Security.PemFilePath ||
		cbs.config.Security.EnableTLS != config.Security.EnableTLS ||
		cbs.config.Grpc != config.Grpc {
		if err := cbs.Stop(); err != nil {
			return err
		}

		cbs.config.Limiter = config.Limiter
		cbs.config.Security = config.Security
		cbs.config.Grpc = config.Grpc

		if err := cbs.init(); err != nil {
			return err
		}

		if err := cbs.Start(); err != nil {
			return err
		}
	}

	return nil
}
