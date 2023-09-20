package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/axiomesh/axiom"
	rbft "github.com/axiomesh/axiom-bft"
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

	Ulimit   uint64   `mapstructure:"ulimit" toml:"ulimit"`
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
	P2P       int64 `mapstructure:"p2p" toml:"p2p"`
	PProf     int64 `mapstructure:"pprof" toml:"pprof"`
	Monitor   int64 `mapstructure:"monitor" toml:"monitor"`
}

type JsonRPC struct {
	GasCap                       uint64   `mapstructure:"gas_cap" toml:"gas_cap"`
	EVMTimeout                   Duration `mapstructure:"evm_timeout" toml:"evm_timeout"`
	Limiter                      JLimiter `mapstructure:"limiter" toml:"limiter"`
	RejectTxsIfConsensusAbnormal bool     `mapstructure:"reject_txs_if_consensus_abnormal" toml:"reject_txs_if_consensus_abnormal"`
}

type P2PPipeGossipsub struct {
	SubBufferSize          int      `mapstructure:"sub_buffer_size" toml:"sub_buffer_size"`
	PeerOutboundBufferSize int      `mapstructure:"peer_outbound_buffer_size" toml:"peer_outbound_buffer_size"`
	ValidateBufferSize     int      `mapstructure:"validate_buffer_size" toml:"validate_buffer_size"`
	SeenMessagesTTL        Duration `mapstructure:"seen_messages_ttl" toml:"seen_messages_ttl"`
}

type P2PPipeSimpleBroadcast struct {
	WorkerCacheSize        int      `mapstructure:"worker_cache_size" toml:"worker_cache_size"`
	WorkerConcurrencyLimit int      `mapstructure:"worker_concurrency_limit" toml:"worker_concurrency_limit"`
	RetryNumber            int      `mapstructure:"retry_number" toml:"retry_number"`
	RetryBaseTime          Duration `mapstructure:"retry_base_time" toml:"retry_base_time"`
}

type P2PPipe struct {
	ReceiveMsgCacheSize int                    `mapstructure:"receive_msg_cache_size" toml:"receive_msg_cache_size"`
	BroadcastType       string                 `mapstructure:"broadcast_type" toml:"broadcast_type"`
	SimpleBroadcast     P2PPipeSimpleBroadcast `mapstructure:"simple_broadcast" toml:"simple_broadcast"`
	Gossipsub           P2PPipeGossipsub       `mapstructure:"gossipsub" toml:"gossipsub"`
}

type P2P struct {
	BootstrapNodeAddresses []string `mapstructure:"bootstrap_node_addresses" toml:"bootstrap_node_addresses"`
	Security               string   `mapstructure:"security" toml:"security"`
	SendTimeout            Duration `mapstructure:"send_timeout" toml:"send_timeout"`
	ReadTimeout            Duration `mapstructure:"read_timeout" toml:"read_timeout"`
	Ping                   Ping     `mapstructure:"ping" toml:"ping"`
	Pipe                   P2PPipe  `mapstructure:"pipe" toml:"pipe"`
}

type Monitor struct {
	Enable bool `mapstructure:"enable" toml:"enable"`
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
	Level          string `mapstructure:"level" toml:"level"`
	Filename       string `mapstructure:"filename" toml:"filename"`
	ReportCaller   bool   `mapstructure:"report_caller" toml:"report_caller"`
	EnableCompress bool   `mapstructure:"enable_compress" toml:"enable_compress"`
	EnableColor    bool   `mapstructure:"enable_color" toml:"enable_color"`

	// unit: day
	MaxAge uint `mapstructure:"max_age" toml:"max_age"`

	// unit: MB
	MaxSize uint `mapstructure:"max_size" toml:"max_size"`

	RotationTime Duration  `mapstructure:"rotation_time" toml:"rotation_time"`
	Module       LogModule `mapstructure:"module" toml:"module"`
}

type LogModule struct {
	P2P        string `mapstructure:"p2p" toml:"p2p"`
	Consensus  string `mapstructure:"consensus" toml:"consensus"`
	Executor   string `mapstructure:"executor" toml:"executor"`
	Governance string `mapstructure:"governance" toml:"governance"`
	Router     string `mapstructure:"router" toml:"router"`
	API        string `mapstructure:"api" toml:"api"`
	CoreAPI    string `mapstructure:"coreapi" toml:"coreapi"`
	Storage    string `mapstructure:"storage" toml:"storage"`
	Profile    string `mapstructure:"profile" toml:"profile"`
	TSS        string `mapstructure:"tss" toml:"tss"`
	Finance    string `mapstructure:"finance" toml:"finance"`
}

type Genesis struct {
	ChainID       uint64          `mapstructure:"chainid" toml:"chainid"`
	GasLimit      uint64          `mapstructure:"gas_limit" toml:"gas_limit"`
	GasPrice      uint64          `mapstructure:"gas_price" toml:"gas_price"`
	MaxGasPrice   uint64          `mapstructure:"max_gas_price" toml:"max_gas_price"`
	MinGasPrice   uint64          `mapstructure:"min_gas_price" toml:"min_gas_price"`
	GasChangeRate float64         `mapstructure:"gas_change_rate" toml:"gas_change_rate"`
	Balance       string          `mapstructure:"balance" toml:"balance"`
	Admins        []*Admin        `mapstructure:"admins" toml:"admins"`
	Accounts      []string        `mapstructure:"accounts" toml:"accounts"`
	EpochInfo     *rbft.EpochInfo `mapstructure:"epoch_info" toml:"epoch_info"`
}

type Admin struct {
	Address string `mapstructure:"address" toml:"address"`
	Weight  uint64 `mapstructure:"weight" toml:"weight"`
	Name    string `mapstructure:"name" toml:"name"`
}

type Order struct {
	Type string `mapstructure:"type" toml:"type"`
}

type Ledger struct {
	Kv string `mapstructure:"kv" toml:"kv"`
}

type Executor struct {
	Type string `mapstructure:"type" toml:"type"`
}

var SupportMultiNode = make(map[string]bool)
var registrationMutex sync.Mutex

func Register(orderType string, isSupported bool) {
	registrationMutex.Lock()
	defer registrationMutex.Unlock()
	SupportMultiNode[orderType] = isSupported
}

func (c *Config) Bytes() ([]byte, error) {
	ret, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func GenesisEpochInfo(epochEnable bool) *rbft.EpochInfo {
	var epochPeriod uint64 = 10000000
	var checkpointPeriod uint64 = 10
	var highWatermarkCheckpointPeriod uint64 = 4
	var proposerElectionType = rbft.ProposerElectionTypeRotating
	if epochEnable {
		epochPeriod = 100
		checkpointPeriod = 1
		highWatermarkCheckpointPeriod = 40
		proposerElectionType = rbft.ProposerElectionTypeWRF
	}

	return &rbft.EpochInfo{
		Version:     1,
		Epoch:       1,
		EpochPeriod: epochPeriod,
		StartBlock:  1,
		P2PBootstrapNodeAddresses: lo.Map(defaultNodeIDs, func(item string, idx int) string {
			return fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", 4001+idx, item)
		}),
		ConsensusParams: &rbft.ConsensusParams{
			CheckpointPeriod:              checkpointPeriod,
			HighWatermarkCheckpointPeriod: highWatermarkCheckpointPeriod,
			MaxValidatorNum:               20,
			BlockMaxTxNum:                 500,
			EnableTimedGenEmptyBlock:      false,
			NotActiveWeight:               1,
			ExcludeView:                   100,
			ProposerElectionType:          proposerElectionType,
		},
		CandidateSet: []*rbft.NodeInfo{},
		ValidatorSet: lo.Map(DefaultNodeAddrs, func(item string, idx int) *rbft.NodeInfo {
			return &rbft.NodeInfo{
				ID:                   uint64(idx + 1),
				AccountAddress:       DefaultNodeAddrs[idx],
				P2PNodeID:            defaultNodeIDs[idx],
				ConsensusVotingPower: 1000,
			}
		}),
	}
}

func DefaultConfig(repoRoot string, epochEnable bool) *Config {
	if axiom.Net == AriesTestnetName {
		return AriesConfig(repoRoot)
	}
	return &Config{
		RepoRoot: repoRoot,
		Ulimit:   65535,
		Port: Port{
			JsonRpc:   8881,
			WebSocket: 9991,
			P2P:       4001,
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
			RejectTxsIfConsensusAbnormal: false,
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
				ReceiveMsgCacheSize: 1024,
				BroadcastType:       P2PPipeBroadcastGossip,
				SimpleBroadcast: P2PPipeSimpleBroadcast{
					WorkerCacheSize:        1024,
					WorkerConcurrencyLimit: 20,
					RetryNumber:            5,
					RetryBaseTime:          Duration(100 * time.Millisecond),
				},
				Gossipsub: P2PPipeGossipsub{
					SubBufferSize:          1024,
					PeerOutboundBufferSize: 1024,
					ValidateBufferSize:     1024,
					SeenMessagesTTL:        Duration(120 * time.Second),
				},
			},
		},
		Order: Order{
			Type: OrderTypeRbft,
		},
		Ledger: Ledger{
			Kv: KVStorageTypePebble,
		},
		Executor: Executor{
			Type: ExecTypeNative,
		},
		Genesis: Genesis{
			ChainID:       1356,
			GasLimit:      0x5f5e100,
			GasPrice:      5000000000000,
			MaxGasPrice:   10000000000000,
			MinGasPrice:   1000000000000,
			GasChangeRate: 0.125,
			Balance:       "1000000000000000000000000000",
			Admins: lo.Map(DefaultNodeAddrs, func(item string, idx int) *Admin {
				return &Admin{
					Address: item,
					Weight:  1,
					Name:    DefaultNodeNames[idx],
				}
			}),
			Accounts: []string{
				"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
				"0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
				"0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC",
				"0x90F79bf6EB2c4f870365E785982E1f101E93b906",
				"0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65",
				"0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc",
				"0x976EA74026E726554dB657fA54763abd0C3a0aa9",
				"0x14dC79964da2C08b23698B3D3cc7Ca32193d9955",
				"0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f",
				"0xa0Ee7A142d267C1f36714E4a8F75612F20a79720",
				"0xBcd4042DE499D14e55001CcbB24a551F3b954096",
				"0x71bE63f3384f5fb98995898A86B02Fb2426c5788",
				"0xFABB0ac9d68B0B445fB7357272Ff202C5651694a",
				"0x1CBd3b2770909D4e10f157cABC84C7264073C9Ec",
				"0xdF3e18d64BC6A983f673Ab319CCaE4f1a57C7097",
				"0xcd3B766CCDd6AE721141F452C550Ca635964ce71",
				"0x2546BcD3c84621e976D8185a91A922aE77ECEc30",
				"0xbDA5747bFD65F08deb54cb465eB87D40e51B197E",
				"0xdD2FD4581271e230360230F9337D5c0430Bf44C0",
				"0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199",
			},
			EpochInfo: GenesisEpochInfo(epochEnable),
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
			Level:          "info",
			Filename:       "axiom",
			ReportCaller:   false,
			EnableCompress: false,
			EnableColor:    true,
			MaxAge:         30,
			MaxSize:        128,
			RotationTime:   Duration(24 * time.Hour),
			Module: LogModule{
				P2P:        "info",
				Consensus:  "info",
				Executor:   "info",
				Governance: "info",
				Router:     "info",
				API:        "info",
				CoreAPI:    "info",
				Storage:    "info",
				Profile:    "info",
				TSS:        "info",
				Finance:    "info",
			},
		},
	}
}

func LoadConfig(repoRoot string) (*Config, error) {
	cfg, err := func() (*Config, error) {
		cfg := DefaultConfig(repoRoot, false)
		cfgPath := path.Join(repoRoot, CfgFileName)
		existConfig := fileutil.Exist(cfgPath)
		if !existConfig {
			err := os.MkdirAll(repoRoot, 0755)
			if err != nil {
				return nil, errors.Wrap(err, "failed to build default config")
			}

			if err := writeConfigWithEnv(cfgPath, cfg); err != nil {
				return nil, errors.Wrap(err, "failed to build default config")
			}
		} else {
			if err := CheckWritable(repoRoot); err != nil {
				return nil, err
			}
			if err := readConfigFromFile(cfgPath, cfg); err != nil {
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
