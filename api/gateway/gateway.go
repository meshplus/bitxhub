package gateway

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/rs/cors"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func Start(config *repo.Config) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard,
			&runtime.JSONPb{OrigName: true, EmitDefaults: false, EnumsAsInts: true},
		),
	)

	handler := cors.New(cors.Options{
		AllowedOrigins: config.AllowedOrigins,
	}).Handler(mux)

	endpoint := fmt.Sprintf("localhost:%d", config.Port.Grpc)
	if config.Security.EnableTLS {
		pemFilePath := filepath.Join(config.RepoRoot, config.Security.PemFilePath)
		serverKeyPath := filepath.Join(config.RepoRoot, config.Security.ServerKeyPath)
		cred, err := credentials.NewServerTLSFromFile(pemFilePath, serverKeyPath)
		if err != nil {
			return err
		}

		conn, err := grpc.DialContext(ctx, endpoint, grpc.WithTransportCredentials(cred))
		if err != nil {
			return err
		}
		err = pb.RegisterChainBrokerHandler(ctx, mux, conn)
		if err != nil {
			return err
		}
		return http.ListenAndServeTLS(fmt.Sprintf(":%d", config.Port.Gateway), pemFilePath, serverKeyPath, wsproxy.WebsocketProxy(handler))
	} else {
		opts := []grpc.DialOption{grpc.WithInsecure()}
		err := pb.RegisterChainBrokerHandlerFromEndpoint(ctx, mux, endpoint, opts)
		if err != nil {
			return err
		}
		return http.ListenAndServe(fmt.Sprintf(":%d", config.Port.Gateway), wsproxy.WebsocketProxy(handler))
	}
}
