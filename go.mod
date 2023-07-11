module github.com/meshplus/bitxhub

go 1.14

replace (
	github.com/meshplus/bitxhub-core => github.com/TaiChiChain/bitxhub-core v1.3.1-0.20230625070128-e24beb3d491e
	github.com/meshplus/bitxhub-kit => github.com/TaiChiChain/bitxhub-kit v1.20.1-0.20230625064843-b7ae66571e52
	github.com/meshplus/bitxhub-model => github.com/TaiChiChain/bitxhub-model v1.20.2-0.20230625065636-b5b1ab540d61
	github.com/meshplus/eth-kit => github.com/TaiChiChain/eth-kit v0.0.0-20230707085233-79e2e9caa315
	github.com/meshplus/go-libp2p-cert => github.com/TaiChiChain/go-libp2p-cert v0.0.0-20230625062152-44e7041e4770
	github.com/meshplus/go-lightp2p => github.com/TaiChiChain/go-lightp2p v0.0.0-20230625064042-335b4532a750
	github.com/ultramesh/rbft => github.com/TaiChiChain/rbft v0.1.5-0.20230703101940-35c10f80be39
)

replace (
	github.com/agl/ed25519 => github.com/binance-chain/edwards25519 v0.0.0-20200305024217-f36fc4b53d43
	github.com/binance-chain/tss-lib => github.com/dawn-to-dusk/tss-lib v1.3.2-0.20220422023240-5ddc16a330ed
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.2
	github.com/hyperledger/fabric => github.com/hyperledger/fabric v2.0.1+incompatible
	github.com/karalabe/usb => github.com/karalabe/usb v0.0.2
	golang.org/x/net => golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200218151345-dad8c97a84f5
	google.golang.org/grpc => google.golang.org/grpc v1.33.0
)

require (
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/Rican7/retry v0.1.0
	github.com/binance-chain/tss-lib v1.3.3-0.20210411025750-fffb56b30511
	github.com/bytecodealliance/wasmtime-go v0.37.0
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
	github.com/gobuffalo/packr/v2 v2.5.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/google/btree v1.0.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/iancoleman/orderedmap v0.2.0
	github.com/juju/ratelimit v1.0.1
	github.com/libp2p/go-libp2p-core v0.5.6
	github.com/libp2p/go-libp2p-swarm v0.2.4
	github.com/looplab/fsm v0.2.0
	github.com/magiconair/properties v1.8.4
	github.com/meshplus/bitxhub-core v1.3.1-0.20220701090824-788a823e6049
	github.com/meshplus/bitxhub-kit v1.2.1-0.20220610092251-68e35ef37b7a
	github.com/meshplus/bitxhub-model v1.2.1-0.20220705090611-2a1e897ee1f7
	github.com/meshplus/eth-kit v0.0.0-20220628031226-4b30a994a2a6
	github.com/meshplus/go-libp2p-cert v0.0.0-20210125114242-7d9ed2eaaccd
	github.com/meshplus/go-lightp2p v0.0.0-20220117071358-c37ba4e6dcbc
	github.com/miguelmota/go-solidity-sha3 v0.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/orcaman/concurrent-map v0.0.0-20210501183033-44dafcb38ecc
	github.com/pelletier/go-toml v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.0
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cast v1.3.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/tidwall/gjson v1.6.8
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5
	github.com/ultramesh/rbft v0.0.0-00010101000000-000000000000
	github.com/urfave/cli v1.22.1
	go.uber.org/atomic v1.7.0
	go.uber.org/zap v1.19.0 // indirect
	google.golang.org/grpc v1.35.0
)
