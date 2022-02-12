module github.com/meshplus/bitxhub

go 1.14

require (
	github.com/Rican7/retry v0.1.0
	github.com/cbergoon/merkletree v0.2.0
	github.com/cheynewallace/tabby v1.1.1
	github.com/common-nighthawk/go-figure v0.0.0-20190529165535-67e0ed34491a
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/ethereum/go-ethereum v1.10.4
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gobuffalo/packd v1.0.0
	github.com/gobuffalo/packr v1.30.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/google/btree v1.0.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/hyperledger/fabric-amcl v0.0.0-20210603140002-2670f91851c8 // indirect
	github.com/juju/ratelimit v1.0.1
	github.com/libp2p/go-libp2p-core v0.5.6
	github.com/looplab/fsm v0.2.0
	github.com/magiconair/properties v1.8.4
	github.com/meshplus/bitxhub-core v1.3.1-0.20220129100150-aa3fd98079e4
	github.com/meshplus/bitxhub-kit v1.2.1-0.20210830031953-cf5f83f2e1dd
	github.com/meshplus/bitxhub-model v1.2.1-0.20210909055322-3025ed2ba82f
	github.com/meshplus/bitxid v0.0.0-20210412025850-e0eaf0f9063a
	github.com/meshplus/did-registry v0.0.0-20210413035015-509c6c3a0bae
	github.com/meshplus/eth-kit v0.0.0-20220105083141-5ec7a17a3dea
	github.com/meshplus/go-libp2p-cert v0.0.0-20210125114242-7d9ed2eaaccd
	github.com/meshplus/go-lightp2p v0.0.0-20210617153734-471d08b829f8
	github.com/miguelmota/go-solidity-sha3 v0.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.0
	github.com/orcaman/concurrent-map v0.0.0-20190826125027-8c72a8bb44f6
	github.com/pelletier/go-toml v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.0
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cast v1.3.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/tidwall/gjson v1.6.8
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5
	github.com/urfave/cli v1.22.1
	github.com/wasmerio/wasmer-go v1.0.4-0.20210601080313-1e68d3994c0b
	github.com/willf/bloom v2.0.3+incompatible
	go.uber.org/atomic v1.7.0
	google.golang.org/grpc v1.33.2
	honnef.co/go/tools v0.2.2 // indirect
)

replace golang.org/x/net => golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7

replace honnef.co/go/tools => honnef.co/go/tools v0.0.1-2019.2.3

replace github.com/golang/protobuf => github.com/golang/protobuf v1.3.2

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200218151345-dad8c97a84f5

replace google.golang.org/grpc => google.golang.org/grpc v1.33.0

replace github.com/hyperledger/fabric => github.com/hyperledger/fabric v2.0.1+incompatible
