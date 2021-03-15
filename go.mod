module github.com/meshplus/bitxhub

require (
	github.com/Rican7/retry v0.1.0
	github.com/aristanetworks/goarista v0.0.0-20200310212843-2da4c1f5881b // indirect
	github.com/cbergoon/merkletree v0.2.0
	github.com/cheynewallace/tabby v1.1.1
	github.com/common-nighthawk/go-figure v0.0.0-20190529165535-67e0ed34491a
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/ethereum/go-ethereum v1.9.18
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/gobuffalo/packd v1.0.0
	github.com/gobuffalo/packr v1.30.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.5.0
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/google/btree v1.0.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/juju/ratelimit v1.0.1
	github.com/libp2p/go-libp2p-core v0.5.6
	github.com/magiconair/properties v1.8.4
	github.com/meshplus/bitxhub-core v0.1.0-rc1.0.20210330035001-b327cf056572
	github.com/meshplus/bitxhub-kit v1.1.2-0.20210112075018-319e668d6359
	github.com/meshplus/bitxhub-model v1.1.2-0.20210309053945-afaea82e9fe1
	github.com/meshplus/bitxid v0.0.0-20210331074321-62187b1145f5
	github.com/meshplus/did-registry v0.0.0-20210407092831-8da970934f93
	github.com/meshplus/bitxhub-model v1.1.2-0.20210409090411-de23bd385c5f
	github.com/meshplus/did-registry v0.0.0-20210412063954-3a6aab1c683e
	github.com/meshplus/go-libp2p-cert v0.0.0-20210125063330-7c25fd5b7a49
	github.com/meshplus/go-lightp2p v0.0.0-20210120082108-df5a536a6192
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/multiformats/go-multiaddr v0.3.0
	github.com/orcaman/concurrent-map v0.0.0-20190826125027-8c72a8bb44f6
	github.com/pelletier/go-toml v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.0
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/rs/cors v1.7.0
	github.com/spf13/afero v1.5.1 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cast v1.3.1
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/sykesm/zap-logfmt v0.0.4 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/syndtr/goleveldb v1.0.1-0.20190923125748-758128399b1d
	github.com/tidwall/gjson v1.6.8
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5
	github.com/urfave/cli v1.22.1
	github.com/wasmerio/go-ext-wasm v0.3.1
	github.com/willf/bitset v1.1.11 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	go.uber.org/atomic v1.7.0
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b // indirect
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4 // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/genproto v0.0.0-20201119123407-9b1e624d6bc4 // indirect
	google.golang.org/grpc v1.33.2
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/golang/protobuf => github.com/golang/protobuf v1.3.2

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200218151345-dad8c97a84f5

replace google.golang.org/grpc => google.golang.org/grpc v1.33.0

replace github.com/hyperledger/fabric => github.com/hyperledger/fabric v2.0.1+incompatible

go 1.13
