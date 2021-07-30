package grpc

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
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
	"google.golang.org/grpc/metadata"
)

var caCertData []byte

type ServerAccessFunc func(ctx context.Context, handler grpc.UnaryHandler) error

var serverAccessInterceptorfunc = func(ctx context.Context, handler grpc.UnaryHandler) error {
	incomingContext, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fmt.Errorf("client access failed")
	}

	var subCertData string
	if val, ok := incomingContext["access"]; ok {
		subCertData = val[0]
	}
	if len(subCertData) == 0 {
		return fmt.Errorf("client access failed: not found key 'access' from header")
	}
	decodeString, err := base64.StdEncoding.DecodeString(subCertData)
	if err != nil {
		return fmt.Errorf("parse subCertData: %w", err)
	}
	block, _ := pem.Decode(decodeString)
	subCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse sub cert: %w", err)
	}

	block, _ = pem.Decode(caCertData)
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse ca cert: %w", err)
	}

	err = subCert.CheckSignatureFrom(caCert)
	if err != nil {
		return err
	}
	return nil
}

type ChainBrokerService struct {
	config  *repo.Config
	genesis *repo.Genesis
	api     api.CoreAPI
	server  *grpc.Server
	logger  logrus.FieldLogger

	ctx    context.Context
	cancel context.CancelFunc
}

// CreateUnaryServerAccessInterceptor returns a new unary server interceptors with serverAccessFunc.
func CreateUnaryServerAccessInterceptor(serverAccessFunc ServerAccessFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := serverAccessFunc(ctx, handler); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
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
	rateLimiter, err := ratelimiter.NewRateLimiterWithQuantum(limiter.Interval, limiter.Capacity, limiter.Quantum)
	if err != nil {
		return err
	}

	var us []grpc.UnaryServerInterceptor
	var ss []grpc.StreamServerInterceptor
	us = append(us, ratelimit.UnaryServerInterceptor(rateLimiter), grpc_prometheus.UnaryServerInterceptor)
	ss = append(ss, ratelimit.StreamServerInterceptor(rateLimiter), grpc_prometheus.StreamServerInterceptor)
	grpcOpts := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(1000),
		grpc.InitialWindowSize(10 * 1024 * 1024),
		grpc.InitialConnWindowSize(100 * 1024 * 1024),
	}

	if config.Security.EnableTLS {
		caCertPath := filepath.Join(config.RepoRoot, config.Cert.AgencyCertPath)
		caCertData, err = ioutil.ReadFile(caCertPath)
		if err != nil {
			return err
		}
		us = append(us, CreateUnaryServerAccessInterceptor(serverAccessInterceptorfunc))
		pemFilePath := filepath.Join(config.RepoRoot, config.Security.PemFilePath)
		serverKeyPath := filepath.Join(config.RepoRoot, config.Security.ServerKeyPath)
		cred, err := credentials.NewServerTLSFromFile(pemFilePath, serverKeyPath)
		if err != nil {
			return err
		}
		grpcOpts = append(grpcOpts, grpc.Creds(cred))
	}
	grpcOpts = append(grpcOpts, grpc_middleware.WithUnaryServerChain(us...), grpc_middleware.WithStreamServerChain(ss...))
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
