module github.com/meshplus/bitxhub

require (
	github.com/Rican7/retry v0.1.0
	github.com/cbergoon/merkletree v0.2.0
	github.com/cheynewallace/tabby v1.1.1
	github.com/common-nighthawk/go-figure v0.0.0-20190529165535-67e0ed34491a
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/ethereum/go-ethereum v1.10.2
	github.com/fatih/color v1.7.0
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/gobuffalo/packd v1.0.0
	github.com/gobuffalo/packr v1.30.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.5.0
	github.com/google/btree v1.0.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/juju/ratelimit v1.0.1
	github.com/libp2p/go-libp2p-core v0.5.6
	github.com/magiconair/properties v1.8.4
	github.com/meshplus/bitxhub-core v0.1.0-rc1.0.20210513030054-c3b3ff976db4
	github.com/meshplus/bitxhub-kit v1.2.0
	github.com/meshplus/bitxhub-model v1.1.2-0.20210513025421-8fde99a99b3f
	github.com/meshplus/bitxid v0.0.0-20210412025850-e0eaf0f9063a
	github.com/meshplus/go-libp2p-cert v0.0.0-20210125063330-7c25fd5b7a49
	github.com/meshplus/go-lightp2p v0.0.0-20210120082108-df5a536a6192
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.0
	github.com/orcaman/concurrent-map v0.0.0-20190826125027-8c72a8bb44f6
	github.com/pelletier/go-toml v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.0
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/rogpeppe/go-internal v1.5.2 // indirect
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cast v1.3.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/tidwall/gjson v1.6.8
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5
	github.com/urfave/cli v1.22.1
	github.com/wasmerio/go-ext-wasm v0.3.1
	github.com/willf/bitset v1.1.11 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	go.uber.org/atomic v1.7.0
	google.golang.org/grpc v1.33.2
)

replace github.com/golang/protobuf => github.com/golang/protobuf v1.3.2

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200218151345-dad8c97a84f5

replace google.golang.org/grpc => google.golang.org/grpc v1.33.0

replace github.com/hyperledger/fabric => github.com/hyperledger/fabric v2.0.1+incompatible

go 1.13
