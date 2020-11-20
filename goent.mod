module github.com/meshplus/bitxhub

require (
	github.com/Rican7/retry v0.1.0
	github.com/aristanetworks/goarista v0.0.0-20200310212843-2da4c1f5881b // indirect
	github.com/bitxhub/did-method-registry v0.0.0-20201119132648-c3faf09b020b
	github.com/bitxhub/parallel-executor v0.0.0-20201027053703-4bec95aa1cda
	github.com/bitxhub/service-mng v0.0.0-20201119121619-60cdbb2396c0
	github.com/cbergoon/merkletree v0.2.0
	github.com/common-nighthawk/go-figure v0.0.0-20190529165535-67e0ed34491a
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/ethereum/go-ethereum v1.9.18
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/gobuffalo/packd v1.0.0
	github.com/gobuffalo/packr v1.30.1
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.4.3
	github.com/golang/snappy v0.0.2 // indirect
	github.com/google/btree v1.0.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/hyperledger/fabric v2.1.1+incompatible // indirect
	github.com/hyperledger/fabric-protos-go v0.0.0-20201028172056-a3136dde2354 // indirect
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible // indirect
	github.com/lestrrat-go/strftime v1.0.3 // indirect
	github.com/libp2p/go-libp2p-core v0.5.6
	github.com/magiconair/properties v1.8.4
	github.com/meshplus/bitxhub-core v0.1.0-rc1.0.20201119115312-979111085a2c
	github.com/meshplus/bitxhub-kit v1.1.2-0.20201027090548-41dfc41037af
	github.com/meshplus/bitxhub-model v1.1.2-0.20201118055706-510eb971b4c6
	github.com/meshplus/go-lightp2p v0.0.0-20201102131103-3fa9723c2c7c
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.2.2
	github.com/orcaman/concurrent-map v0.0.0-20190826125027-8c72a8bb44f6
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.0
	github.com/prometheus/tsdb v0.7.1
	github.com/rogpeppe/go-internal v1.5.2 // indirect
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cast v1.3.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.0
	github.com/sykesm/zap-logfmt v0.0.4 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20190923125748-758128399b1d
	github.com/tidwall/gjson v1.3.5
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5
	github.com/urfave/cli v1.22.1
	github.com/wasmerio/go-ext-wasm v0.3.1
	github.com/willf/bitset v1.1.11 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	go.uber.org/atomic v1.7.0
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/crypto v0.0.0-20201117144127-c1f2f97bffc9 // indirect
	golang.org/x/sys v0.0.0-20201119102817-f84b799fce68 // indirect
	google.golang.org/genproto v0.0.0-20201119123407-9b1e624d6bc4 // indirect
	google.golang.org/grpc v1.33.2
)

replace github.com/golang/protobuf v1.4.2 => github.com/golang/protobuf v1.3.2

replace github.com/hyperledger/fabric => github.com/hyperledger/fabric v2.0.1+incompatible

go 1.13
