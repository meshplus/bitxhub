module github.com/meshplus/bitxhub

require (
	github.com/Rican7/retry v0.1.0
	github.com/bitxhub/parallel-executor v0.0.0-20201022141235-a2d73478b5a0
	github.com/cbergoon/merkletree v0.2.0
	github.com/common-nighthawk/go-figure v0.0.0-20190529165535-67e0ed34491a
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/ethereum/go-ethereum v1.9.18
	github.com/gobuffalo/packd v1.0.0
	github.com/gobuffalo/packr v1.30.1
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.4.3
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/google/btree v1.0.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.13.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/libp2p/go-libp2p-core v0.5.6
	github.com/magiconair/properties v1.8.1
	github.com/meshplus/bitxhub-core v0.1.0-rc1.0.20201022032823-4591a8883995
	github.com/meshplus/bitxhub-kit v1.1.2-0.20201021105954-468d0a9d7957
	github.com/meshplus/bitxhub-model v1.1.2-0.20201021152621-0b3c17c54b23
	github.com/meshplus/go-lightp2p v0.0.0-20200817105923-6b3aee40fa54
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.2.2
	github.com/orcaman/concurrent-map v0.0.0-20190826125027-8c72a8bb44f6
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.0
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cast v1.3.0
	github.com/spf13/viper v1.6.1
	github.com/stretchr/testify v1.6.0
	github.com/syndtr/goleveldb v1.0.1-0.20190923125748-758128399b1d
	github.com/tidwall/gjson v1.3.5
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5
	github.com/urfave/cli v1.22.1
	github.com/wasmerio/go-ext-wasm v0.3.1
	github.com/willf/bitset v1.1.11 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	go.uber.org/atomic v1.6.0
	google.golang.org/grpc v1.27.1
)

replace github.com/golang/protobuf v1.4.2 => github.com/golang/protobuf v1.3.2

go 1.14
