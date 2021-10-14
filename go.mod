module github.com/meshplus/bitxhub

go 1.14

require (
	github.com/Rican7/retry v0.1.0
	github.com/cbergoon/merkletree v0.2.0
	github.com/cheynewallace/tabby v1.1.1
	github.com/common-nighthawk/go-figure v0.0.0-20190529165535-67e0ed34491a
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/ethereum/go-ethereum v1.10.8
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/gobuffalo/packd v1.0.0
	github.com/gobuffalo/packr v1.30.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/google/btree v1.0.0
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/iancoleman/orderedmap v0.2.0
	github.com/juju/ratelimit v1.0.1
	github.com/libp2p/go-libp2p-core v0.5.6
	github.com/looplab/fsm v0.2.0
	github.com/magiconair/properties v1.8.4
	github.com/meshplus/bitxhub-core v1.3.1-0.20211015044526-39e8bbfefa99
	github.com/meshplus/bitxhub-kit v1.2.1-0.20210901014031-ee93dd4f4ced
	github.com/meshplus/bitxhub-model v1.2.1-0.20211013025747-b1d8b99f2d1e
	github.com/meshplus/eth-kit v0.0.0-20210906064541-8dfea98dbf95
	github.com/meshplus/go-libp2p-cert v0.0.0-20210125114242-7d9ed2eaaccd
	github.com/meshplus/go-lightp2p v0.0.0-20210617153734-471d08b829f8
	github.com/meshplus/go-wasm-metering v0.0.0-20210913105809-61352edd2047
	github.com/miguelmota/go-solidity-sha3 v0.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.0
	github.com/orcaman/concurrent-map v0.0.0-20210501183033-44dafcb38ecc
	github.com/pelletier/go-toml v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.0
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cast v1.3.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/tidwall/gjson v1.6.8
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5
	github.com/urfave/cli v1.22.1
	github.com/wasmerio/wasmer-go v1.0.4
	github.com/willf/bitset v1.1.11 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	go.uber.org/atomic v1.7.0
	google.golang.org/grpc v1.33.2
)

replace github.com/golang/protobuf => github.com/golang/protobuf v1.3.2

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200218151345-dad8c97a84f5

replace google.golang.org/grpc => google.golang.org/grpc v1.33.0

replace github.com/hyperledger/fabric => github.com/hyperledger/fabric v2.0.1+incompatible

replace github.com/wasmerio/wasmer-go v1.0.4 => github.com/meshplus/wasmer-go v1.0.5-0.20210817103436-19ec68f8bfe2
