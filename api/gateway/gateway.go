package gateway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/rs/cors"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"
)

func Start(config *repo.Config) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard,
			&runtime.JSONPb{OrigName: true, EmitDefaults: true},
		),
	)

	handler := cors.New(cors.Options{
		AllowedOrigins: config.AllowedOrigins,
	}).Handler(mux)

	opts := []grpc.DialOption{grpc.WithInsecure()}

	endpoint := fmt.Sprintf("localhost:%d", config.Port.Grpc)
	err := pb.RegisterChainBrokerHandlerFromEndpoint(ctx, mux, endpoint, opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(fmt.Sprintf(":%d", config.Port.Gateway), wsproxy.WebsocketProxy(handler))
}
