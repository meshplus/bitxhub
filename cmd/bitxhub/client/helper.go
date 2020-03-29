package client

import (
	"github.com/meshplus/bitxhub-kit/key"
	rpcx "github.com/meshplus/go-bitxhub-client"
)

func loadClient(keyPath, grpcAddr string) (rpcx.Client, error) {
	key, err := key.LoadKey(keyPath)
	if err != nil {
		return nil, err
	}

	privateKey, err := key.GetPrivateKey("bitxhub")
	if err != nil {
		return nil, err
	}

	return rpcx.New(
		rpcx.WithAddrs([]string{grpcAddr}),
		rpcx.WithPrivateKey(privateKey),
	)
}
