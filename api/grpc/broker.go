package grpc

import (
	"context"
	"fmt"
	"net"
	"path/filepath"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
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
	config                  *repo.Config
	genesis                 *repo.Genesis
	api                     api.CoreAPI
	server                  *grpc.Server
	supportCryptoTypeToName map[crypto.KeyType]string
	logger                  logrus.FieldLogger

	ctx    context.Context
	cancel context.CancelFunc
}

func NewChainBrokerService(api api.CoreAPI, config *repo.Config, genesis *repo.Genesis) (*ChainBrokerService, error) {
	limiter := config.Limiter
	rateLimiter := ratelimiter.NewRateLimiterWithQuantum(limiter.Interval, limiter.Capacity, limiter.Quantum)

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
		cred, err := credentials.NewServerTLSFromFile(pemFilePath, serverKeyPath)
		if err != nil {
			return nil, err
		}
		grpcOpts = append(grpcOpts, grpc.Creds(cred))
	}

	supportCryptoTypeToName, err := CheckAlgorithms(config.Crypto.Algorithms)
	if err != nil {
		return nil, err
	}

	server := grpc.NewServer(grpcOpts...)
	ctx, cancel := context.WithCancel(context.Background())
	return &ChainBrokerService{
		supportCryptoTypeToName: supportCryptoTypeToName,
		logger:                  loggers.Logger(loggers.API),
		config:                  config,
		genesis:                 genesis,
		api:                     api,
		server:                  server,
		ctx:                     ctx,
		cancel:                  cancel,
	}, nil
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

	cbs.logger.Info("GRPC service stopped")

	return nil
}

func CheckAlgorithms(algorithms []string) (map[crypto.KeyType]string, error) {
	supportCryptoTypeToName := make(map[crypto.KeyType]string)
	for _, algorithm := range algorithms {
		cryptoType, err := crypto.CryptoNameToType(algorithm)
		if err != nil {
			return nil, err
		}
		if !asym.SupportedKeyType(cryptoType) {
			return nil, fmt.Errorf("unsupport algorithm:%s", algorithm)
		}
		supportCryptoTypeToName[cryptoType] = algorithm
	}

	fmt.Printf("Supported crypto type:")
	for _, name := range supportCryptoTypeToName {
		fmt.Printf("%s ", name)
	}
	fmt.Printf("\n\n")

	return supportCryptoTypeToName, nil
}
