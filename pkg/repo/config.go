package repo

import (
	"encoding/json"
	"os"
	"path"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/axiomesh/axiom-kit/fileutil"
)

type Duration time.Duration

func (d *Duration) MarshalText() (text []byte, err error) {
	return []byte(time.Duration(*d).String()), nil
}

func (d *Duration) UnmarshalText(b []byte) error {
	x, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = Duration(x)
	return nil
}

func StringToTimeDurationHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(Duration(5)) {
			return data, nil
		}

		d, err := time.ParseDuration(data.(string))
		if err != nil {
			return nil, err
		}
		return Duration(d), nil
	}
}

func (d *Duration) ToDuration() time.Duration {
	return time.Duration(*d)
}

func (d *Duration) String() string {
	return time.Duration(*d).String()
}

type Config struct {
	RepoRoot string `mapstructure:"-" toml:"-"`

	Port     Port     `mapstructure:"port" toml:"port"`
	JsonRPC  JsonRPC  `mapstructure:"jsonrpc" toml:"jsonrpc"`
	P2P      P2P      `mapstructure:"p2p" toml:"p2p"`
	Order    Order    `mapstructure:"order" toml:"order"`
	Ledger   Ledger   `mapstructure:"ledger" toml:"ledger"`
	Executor Executor `mapstructure:"executor" toml:"executor"`
	Genesis  Genesis  `mapstructure:"genesis" toml:"genesis"`
	PProf    PProf    `mapstructure:"pprof" toml:"pprof"`
	Monitor  Monitor  `mapstructure:"monitor" toml:"monitor"`
	Log      Log      `mapstructure:"log" toml:"log"`
}

type Port struct {
	JsonRpc   int64 `mapstructure:"jsonrpc" toml:"jsonrpc"`
	WebSocket int64 `mapstructure:"websocket" toml:"websocket"`
	PProf     int64 `mapstructure:"pprof" toml:"pprof"`
	Monitor   int64 `mapstructure:"monitor" toml:"monitor"`
}

type JsonRPC struct {
	GasCap     uint64   `mapstructure:"gas_cap" toml:"gas_cap"`
	EVMTimeout Duration `mapstructure:"evm_timeout" toml:"evm_timeout"`
	Limiter    JLimiter `mapstructure:"limiter" toml:"limiter"`
}

type P2PPipe struct {
	BroadcastType       string `mapstructure:"broadcast_type" toml:"broadcast_type"`
	ReceiveMsgCacheSize int    `mapstructure:"receive_msg_cache_size" toml:"receive_msg_cache_size"`
}

type P2P struct {
	Security    string   `mapstructure:"security" toml:"security"`
	SendTimeout Duration `mapstructure:"send_timeout" toml:"send_timeout"`
	ReadTimeout Duration `mapstructure:"read_timeout" toml:"read_timeout"`
	Ping        Ping     `mapstructure:"ping" toml:"ping"`
	Pipe        P2PPipe  `mapstructure:"pipe" toml:"pipe"`
}

type Monitor struct {
	Enable bool
}

type PProf struct {
	Enable   bool     `mapstructure:"enable" toml:"enbale"`
	PType    string   `mapstructure:"ptype" toml:"ptype"`
	Mode     string   `mapstructure:"mode" toml:"mode"`
	Duration Duration `mapstructure:"duration" toml:"duration"`
}

type JLimiter struct {
	Interval Duration `mapstructure:"interval" toml:"interval"`
	Quantum  int64    `mapstructure:"quantum" toml:"quantum"`
	Capacity int64    `mapstructure:"capacity" toml:"capacity"`
}

type Ping struct {
	Enable   bool     `mapstructure:"enable" toml:"enable"`
	Duration Duration `mapstructure:"duration" toml:"duration"`
}

type Log struct {
	Level        string    `mapstructure:"level" toml:"level"`
	Filename     string    `mapstructure:"filename" toml:"filename"`
	ReportCaller bool      `mapstructure:"report_caller" toml:"report_caller"`
	MaxAge       Duration  `mapstructure:"max_age" toml:"max_age"`
	RotationTime Duration  `mapstructure:"rotation_time" toml:"rotation_time"`
	Module       LogModule `mapstructure:"module" toml:"module"`
}

type LogModule struct {
	P2P       string `mapstructure:"p2p" toml:"p2p"`
	Consensus string `mapstructure:"consensus" toml:"consensus"`
	Executor  string `mapstructure:"executor" toml:"executor"`
	Router    string `mapstructure:"router" toml:"router"`
	API       string `mapstructure:"api" toml:"api"`
	CoreAPI   string `mapstructure:"coreapi" toml:"coreapi"`
	Storage   string `mapstructure:"storage" toml:"storage"`
	Profile   string `mapstructure:"profile" toml:"profile"`
	TSS       string `mapstructure:"tss" toml:"tss"`
	Finance   string `mapstructure:"finance" toml:"finance"`
}

type Genesis struct {
	ChainID       uint64    `mapstructure:"chainid" toml:"chainid"`
	GasLimit      uint64    `mapstructure:"gas_limit" toml:"gas_limit"`
	GasPrice      uint64    `mapstructure:"gas_price" toml:"gas_price"`
	MaxGasPrice   uint64    `mapstructure:"max_gas_price" toml:"max_gas_price"`
	MinGasPrice   uint64    `mapstructure:"min_gas_price" toml:"min_gas_price"`
	GasChangeRate float64   `mapstructure:"gas_change_rate" toml:"gas_change_rate"`
	Balance       string    `mapstructure:"balance" toml:"balance"`
	Admins        []*Admin  `mapstructure:"admins" toml:"admins"`
	Members       []*Member `mapstructure:"members" toml:"members"`
}

type Admin struct {
	Address string `mapstructure:"address" toml:"address"`
	Weight  uint64 `mapstructure:"weight" toml:"weight"`
}

type Txpool struct {
	BatchSize    int      `mapstructure:"batch_size" toml:"batch_size"`
	BatchTimeout Duration `mapstructure:"batch_timeout" toml:"batch_timeout"`
}

type Order struct {
	Type string `mapstructure:"type" toml:"type"`
}

type Ledger struct {
	Kv string `mapstructure:"kv" toml:"kv"`
}

type Executor struct {
}

type Member struct {
	NodeId string `mapstructure:"node_id" toml:"node_id"`
}

func (c *Config) Bytes() ([]byte, error) {
	ret, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func DefaultConfig(repoRoot string) *Config {
	return &Config{
		RepoRoot: repoRoot,
		Port: Port{
			JsonRpc:   8881,
			WebSocket: 9991,
			PProf:     53121,
			Monitor:   40011,
		},
		JsonRPC: JsonRPC{
			GasCap:     300000000,
			EVMTimeout: Duration(5 * time.Second),
			Limiter: JLimiter{
				Interval: 50,
				Quantum:  500,
				Capacity: 10000,
			},
		},
		P2P: P2P{
			Security:    P2PSecurityTLS,
			SendTimeout: Duration(5 * time.Second),
			ReadTimeout: Duration(5 * time.Second),
			Ping: Ping{
				Enable:   false,
				Duration: Duration(15 * time.Second),
			},
			Pipe: P2PPipe{
				BroadcastType:       P2PPipeBroadcastGossip,
				ReceiveMsgCacheSize: 100,
			},
		},
		Order: Order{
			Type: "rbft",
		},
		Ledger: Ledger{
			Kv: "leveldb",
		},
		Executor: Executor{},
		Genesis: Genesis{
			ChainID:       1356,
			GasLimit:      0x5f5e100,
			GasPrice:      5000000000000,
			MaxGasPrice:   10000000000000,
			MinGasPrice:   1000000000000,
			GasChangeRate: 0.125,
			Balance:       "1000000000000000000000000000",
			Admins: []*Admin{
				{
					Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
					Weight:  1,
				},
				{
					Address: "0x79a1215469FaB6f9c63c1816b45183AD3624bE34",
					Weight:  1,
				},
				{
					Address: "0x97c8B516D19edBf575D72a172Af7F418BE498C37",
					Weight:  1,
				},
				{
					Address: "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8",
					Weight:  1,
				},
			},
			Members: []*Member{
				{
					NodeId: "16Uiu2HAmJ38LwfY6pfgDWNvk3ypjcpEMSePNTE6Ma2NCLqjbZJSF",
				},
				{
					NodeId: "16Uiu2HAmRypzJbdbUNYsCV2VVgv9UryYS5d7wejTJXT73mNLJ8AK",
				},
				{
					NodeId: "16Uiu2HAmTwEET536QC9MZmYFp1NUshjRuaq5YSH1sLjW65WasvRk",
				},
				{
					NodeId: "16Uiu2HAmQBFTnRr84M3xNhi3EcWmgZnnBsDgewk4sNtpA3smBsHJ",
				},
			},
		},
		PProf: PProf{
			Enable:   true,
			PType:    PprofTypeHTTP,
			Mode:     PprofModeMem,
			Duration: Duration(30 * time.Second),
		},
		Monitor: Monitor{
			Enable: true,
		},
		Log: Log{
			Level:        "info",
			Filename:     "axiom.log",
			ReportCaller: false,
			MaxAge:       Duration(90 * 24 * time.Hour),
			RotationTime: Duration(24 * time.Hour),
			Module: LogModule{
				P2P:       "info",
				Consensus: "info",
				Executor:  "info",
				Router:    "info",
				API:       "info",
				CoreAPI:   "info",
				Storage:   "info",
				Profile:   "info",
				TSS:       "info",
				Finance:   "info",
			},
		},
	}
}

func LoadConfig(repoRoot string) (*Config, error) {
	cfg, err := func() (*Config, error) {
		rootPath, err := LoadRepoRootFromEnv(repoRoot)
		if err != nil {
			return nil, err
		}
		cfg := DefaultConfig(rootPath)

		cfgPath := path.Join(repoRoot, cfgFileName)
		existConfig := fileutil.Exist(cfgPath)
		if !existConfig {
			err := os.MkdirAll(rootPath, 0755)
			if err != nil {
				return nil, errors.Wrap(err, "failed to build default config")
			}

			if err := writeConfigWithEnv(cfgPath, cfg); err != nil {
				return nil, errors.Wrap(err, "failed to build default config")
			}
		} else {
			if err := CheckWritable(rootPath); err != nil {
				return nil, err
			}
			if err = readConfigFromFile(cfgPath, cfg); err != nil {
				return nil, err
			}
		}

		return cfg, nil
	}()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}
	return cfg, nil
}
