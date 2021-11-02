package gateway

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Gateway struct {
	server          *http.Server
	mux             *runtime.ServeMux
	certFile        string
	keyFile         string
	gatewayCertFile string
	gatewayKeyFile  string
	endpoint        string
	config          repo.Config
	logger          logrus.FieldLogger

	ctx    context.Context
	cancel context.CancelFunc
}

func NewGateway(config *repo.Config) *Gateway {
	gateway := &Gateway{
		config: *config,
		logger: loggers.Logger(loggers.API),
	}
	gateway.init()

	return gateway
}

func (g *Gateway) init() {
	g.ctx, g.cancel = context.WithCancel(context.Background())
	config := g.config
	g.mux = runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard,
			&runtime.JSONPb{OrigName: true, EmitDefaults: false, EnumsAsInts: true},
		),
	)
	handler := cors.New(cors.Options{
		AllowedOrigins: config.AllowedOrigins,
	}).Handler(g.mux)
	g.endpoint = fmt.Sprintf("localhost:%d", config.Port.Grpc)

	if config.Security.EnableTLS {
		g.certFile = filepath.Join(config.RepoRoot, config.Security.PemFilePath)
		g.keyFile = filepath.Join(config.RepoRoot, config.Security.ServerKeyPath)
		g.gatewayCertFile = filepath.Join(config.RepoRoot, config.Security.GatewayCertPath)
		g.gatewayKeyFile = filepath.Join(config.RepoRoot, config.Security.GatewayKeyPath)
		clientCaCert, _ := ioutil.ReadFile(g.certFile)
		clientCaCertPool := x509.NewCertPool()
		clientCaCertPool.AppendCertsFromPEM(clientCaCert)
		g.server = &http.Server{
			Addr:    fmt.Sprintf(":%d", config.Port.Gateway),
			Handler: wsproxy.WebsocketProxy(handler),
			TLSConfig: &tls.Config{
				//MinVersion: tls.VersionTLS12,
				ClientCAs:  clientCaCertPool,
				ClientAuth: tls.RequireAndVerifyClientCert,
			},
		}
	} else {
		g.certFile = ""
		g.keyFile = ""
		g.server = &http.Server{
			Addr:    fmt.Sprintf(":%d", config.Port.Gateway),
			Handler: wsproxy.WebsocketProxy(handler)}
	}
}

func (g *Gateway) Start() error {
	g.logger.WithField("port", g.config.Port.Gateway).Info("Gateway service started")
	if g.certFile != "" || g.keyFile != "" {
		cert, err := tls.LoadX509KeyPair(g.gatewayCertFile, g.gatewayKeyFile)
		if err != nil {
			return fmt.Errorf("load tls key failed: %w", err)
		}
		cred := credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true})

		conn, err := grpc.DialContext(g.ctx, g.endpoint, grpc.WithTransportCredentials(cred))
		if err != nil {
			return fmt.Errorf("dial context failed: %w", err)
		}
		err = pb.RegisterChainBrokerHandler(g.ctx, g.mux, conn)
		if err != nil {
			return fmt.Errorf("register chain broker handler failed: %w", err)
		}

		go func() {
			err := g.server.ListenAndServeTLS(g.certFile, g.keyFile)
			if err != nil {
				g.logger.Errorf("ListenAndServeTLS failed: %s", err.Error())
			}
		}()
	} else {
		opts := []grpc.DialOption{grpc.WithInsecure()}
		err := pb.RegisterChainBrokerHandlerFromEndpoint(g.ctx, g.mux, g.endpoint, opts)
		if err != nil {
			return fmt.Errorf("register chain broker handler from endpoint %s failed: %w", g.endpoint, err)
		}

		go func() {
			err := g.server.ListenAndServe()
			if err != nil {
				g.logger.Errorf("ListenAndServe failed: %v", err)
			}
		}()
	}

	return nil
}

func (g *Gateway) Stop() error {
	g.cancel()
	g.logger.Info("Gateway service stopped")
	return g.server.Close()
}

func (g *Gateway) ReConfig(config *repo.Config) error {
	if g.config.Security.EnableTLS != config.Security.EnableTLS ||
		g.config.Security.PemFilePath != config.Security.PemFilePath ||
		g.config.Security.ServerKeyPath != config.Security.ServerKeyPath ||
		g.config.Grpc != config.Grpc ||
		g.config.Port.Gateway != config.Port.Gateway ||
		!equalAllowedOrigins(g.config.AllowedOrigins, config.AllowedOrigins) {

		if err := g.Stop(); err != nil {
			return fmt.Errorf("stop gateway failed: %w", err)
		}

		g.config.Security = config.Security
		g.config.Grpc = config.Grpc
		g.config.Port.Gateway = config.Port.Gateway
		g.config.AllowedOrigins = config.AllowedOrigins

		g.init()

		if err := g.Start(); err != nil {
			return fmt.Errorf("start gateway failed: %w", err)
		}
	}

	return nil

}

func equalAllowedOrigins(strings0, strings1 []string) bool {
	if len(strings0) != len(strings1) {
		return false
	}

	sort.Strings(strings0)
	sort.Strings(strings1)

	for i := range strings0 {
		if !strings.EqualFold(strings0[i], strings1[i]) {
			return false
		}
	}

	return true
}
