package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/ratelimiter"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type ChainBrokerService struct {
	config  *repo.Config
	genesis *repo.Genesis
	api     api.CoreAPI
	server  *grpc.Server
	logger  logrus.FieldLogger
	ledger  *ledger.Ledger

	ctx    context.Context
	cancel context.CancelFunc
}

const (
	ACCOUNT_KEY = "account"
)

var auditStreamInterfaceMap = map[string]struct{}{
	"/pb.ChainBroker/SubscribeAuditInfo": {},
}

var auditUnaryInterfaceMap = map[string]struct{}{
	"/pb.ChainBroker/GetBlockHeaders":          {},
	"/pb.ChainBroker/SendTransaction":          {},
	"/pb.ChainBroker/GetPendingNonceByAccount": {},
}

func NewChainBrokerService(api api.CoreAPI, config *repo.Config, genesis *repo.Genesis, ledger *ledger.Ledger) (*ChainBrokerService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cbs := &ChainBrokerService{
		logger:  loggers.Logger(loggers.API),
		config:  config,
		genesis: genesis,
		api:     api,
		ctx:     ctx,
		cancel:  cancel,
		ledger:  ledger,
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

	checkPermissionUnaryServerInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, fmt.Errorf("missing meta data!")
		}
		if ok, err := checkPermissionUnary(md[ACCOUNT_KEY], info, cbs.ledger, cbs.logger); err != nil {
			return nil, fmt.Errorf("checkPermissionUnary err: %v", err)
		} else if !ok {
			return nil, fmt.Errorf("no permission request, account: %s, method: %s", md[ACCOUNT_KEY], info.FullMethod)
		}
		return handler(ctx, req)
	}

	checkPermissionStreamServerInterceptor := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return fmt.Errorf("missing meta data!")
		}
		if ok, err := checkPermissionStream(md[ACCOUNT_KEY], info, cbs.ledger, cbs.logger); err != nil {
			return fmt.Errorf("checkPermissionStream err: %v", err)
		} else if !ok {
			return fmt.Errorf("no permission request, account: %s, method: %s", md[ACCOUNT_KEY], info.FullMethod)
		}
		return handler(srv, ss)
	}

	grpcOpts := []grpc.ServerOption{
		grpc_middleware.WithUnaryServerChain(ratelimit.UnaryServerInterceptor(rateLimiter), grpc_prometheus.UnaryServerInterceptor, checkPermissionUnaryServerInterceptor),
		grpc_middleware.WithStreamServerChain(ratelimit.StreamServerInterceptor(rateLimiter), grpc_prometheus.StreamServerInterceptor, checkPermissionStreamServerInterceptor),
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

func checkPermissionStream(accounts []string, info *grpc.StreamServerInfo, l *ledger.Ledger, logger logrus.FieldLogger) (bool, error) {
	logger.WithFields(logrus.Fields{
		"accounts": accounts,
		"method":   info.FullMethod,
	}).Debug("check permission stream")

	for _, addr := range accounts {
		isAudit, isAvailable, err := checkAuditAccount(addr, l, logger)
		if err != nil {
			return false, err
		}
		if isAudit {
			if !isAvailable {
				logger.WithFields(logrus.Fields{
					"addr":   addr,
					"method": info.FullMethod,
				}).Debug("audit node is not available, no permission stream")
				return false, nil
			} else if _, ok := auditStreamInterfaceMap[info.FullMethod]; !ok {
				logger.WithFields(logrus.Fields{
					"addr":   addr,
					"method": info.FullMethod,
				}).Debug("audit node has no permission to the stream method")
				return false, nil
			}
		}
	}

	logger.WithFields(logrus.Fields{
		"accounts": accounts,
		"method":   info.FullMethod,
	}).Debug("audit node has permission to the stream method")
	return true, nil
}

func checkPermissionUnary(accounts []string, info *grpc.UnaryServerInfo, l *ledger.Ledger, logger logrus.FieldLogger) (bool, error) {
	logger.WithFields(logrus.Fields{
		"accounts": accounts,
		"method":   info.FullMethod,
	}).Debug("check permission unary")

	for _, addr := range accounts {
		isAudit, isAvailable, err := checkAuditAccount(addr, l, logger)
		if err != nil {
			return false, err
		}
		if isAudit {
			if !isAvailable {
				logger.WithFields(logrus.Fields{
					"addr":   addr,
					"method": info.FullMethod,
				}).Debug("audit node is not available, no permission unary")
				return false, nil
			} else if _, ok := auditUnaryInterfaceMap[info.FullMethod]; !ok {
				logger.WithFields(logrus.Fields{
					"addr":   addr,
					"method": info.FullMethod,
				}).Debug("audit node has no permission to the unary method")
				return false, nil
			}
		}
	}

	logger.WithFields(logrus.Fields{
		"accounts": accounts,
		"method":   info.FullMethod,
	}).Debug("the account has permission to the unary method")
	return true, nil
}

// return: isAuditNode, isAvailableAuditNode, error
func checkAuditAccount(addr string, l *ledger.Ledger, logger logrus.FieldLogger) (bool, bool, error) {
	lg := l.Copy()
	ok, nodeData := lg.GetState(constant.NodeManagerContractAddr.Address(), []byte(node_mgr.NodeKey(addr)))
	if !ok {
		logger.WithFields(logrus.Fields{
			"account": addr,
		}).Debug("the account is not an audit node")
		return false, false, nil
	}

	node := &node_mgr.Node{}
	if err := json.Unmarshal(nodeData, node); err != nil {
		logger.WithFields(logrus.Fields{
			"account": addr,
			"err":     err,
		}).Error("checkAuditAccount, unmarshal node error")
		return false, false, err
	}

	if node.NodeType != node_mgr.NVPNode {
		logger.WithFields(logrus.Fields{
			"account": addr,
		}).Debug("the account is not an audit node")
		return false, false, nil
	}

	if node.IsAvailable() {
		logger.WithFields(logrus.Fields{
			"account": addr,
		}).Debug("the account is an available audit node")
		return true, true, nil
	}

	logger.WithFields(logrus.Fields{
		"account": addr,
	}).Debug("the account is an unavailable audit node")
	return true, false, nil
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
