package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
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
		return nil, fmt.Errorf("init chain broker service failed: %w", err)
	}

	return cbs, nil
}

func (cbs *ChainBrokerService) init() error {
	config := cbs.config
	limiter := cbs.config.Limiter
	rateLimiter, err := ratelimiter.NewRateLimiterWithQuantum(limiter.Interval, limiter.Capacity, limiter.Quantum)
	if err != nil {
		return fmt.Errorf("init rate limiter failed: %w", err)
	}

	grpcOpts := []grpc.ServerOption{
		grpc_middleware.WithUnaryServerChain(ratelimit.UnaryServerInterceptor(rateLimiter), grpc_prometheus.UnaryServerInterceptor),
		grpc_middleware.WithStreamServerChain(ratelimit.StreamServerInterceptor(rateLimiter), grpc_prometheus.StreamServerInterceptor),
		grpc.MaxConcurrentStreams(1000),
		grpc.InitialWindowSize(10 * 1024 * 1024),
		grpc.InitialConnWindowSize(100 * 1024 * 1024),
	}

	if config.Security.EnableTLS {
		pemFilePath := filepath.Join(config.RepoRoot, config.Security.PemFilePath)
		serverKeyPath := filepath.Join(config.RepoRoot, config.Security.ServerKeyPath)
		cert, err := tls.LoadX509KeyPair(pemFilePath, serverKeyPath)
		if err != nil {
			return fmt.Errorf("load tls key failed: %w", err)
		}
		clientCaCert, _ := ioutil.ReadFile(pemFilePath)
		clientCaCertPool := x509.NewCertPool()
		clientCaCertPool.AppendCertsFromPEM(clientCaCert)
		cred := credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: clientCaCertPool})

		grpcOpts = append(grpcOpts, grpc.Creds(cred))
	}
	cbs.server = grpc.NewServer(grpcOpts...)
	return nil
}

func (cbs *ChainBrokerService) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cbs.config.Port.Grpc))
	if err != nil {
		return fmt.Errorf("init listen grpc port %d failed: %w", cbs.config.Port.Grpc, err)
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
			return fmt.Errorf("stop chain broker service failed: %w", err)
		}

		cbs.config.Limiter = config.Limiter
		cbs.config.Security = config.Security
		cbs.config.Grpc = config.Grpc

		if err := cbs.init(); err != nil {
			return fmt.Errorf("init chain broker service failed: %w", err)
		}

		if err := cbs.Start(); err != nil {
			return fmt.Errorf("start chain broker service failed: %w", err)
		}
	}

	return nil
}
