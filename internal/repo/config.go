package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const (
	// defaultPathName is the default config dir name
	defaultPathName = ".bitxhub"
	// defaultPathRoot is the path to the default config dir location.
	defaultPathRoot = "~/" + defaultPathName
	// envDir is the environment variable used to change the path root.
	envDir = "BITXHUB_PATH"
	// Config name
	configName = "bitxhub.toml"
	// key name
	KeyName = "key.json"
	// API name
	APIName = "api"
	// admin weight
	SuperAdminWeight  = 2
	NormalAdminWeight = 1
	// governance strategy default participate threshold
	DefaultParticipateThreshold = 0.75
	// bitxhub DID root prefix
	BitxhubRootPrefix = "did:bitxhub"
	//Passwd
	DefaultPasswd = "bitxhub"
)

type Config struct {
	RepoRoot string `json:"repo_root"`
	Title    string `json:"title"`
	Solo     bool   `json:"solo"`
	Port     `json:"port"`
	PProf    `json:"pprof"`
	Monitor  `json:"monitor"`
	Limiter  `json:"limiter"`
	Appchain `json:"appchain"`
	Gateway  `json:"gateway"`
	Ping     `json:"ping"`
	Log      `json:"log"`
	Cert     `json:"cert"`
	Txpool   `json:"txpool"`
	Order    `json:"order"`
	Executor `json:"executor"`
	Ledger   `json:"ledger"`
	Genesis  `json:"genesis"`
	Security Security `toml:"security" json:"security"`
	License  License  `toml:"license" json:"license"`
}

// Security are files used to setup connection with tls
type Security struct {
	EnableTLS     bool   `mapstructure:"enable_tls"`
	EnableAccess  bool   `mapstructure:"enable_access"`
	PemFilePath   string `mapstructure:"pem_file_path" json:"pem_file_path"`
	ServerKeyPath string `mapstructure:"server_key_path" json:"server_key_path"`
}

type Port struct {
	JsonRpc int64 `toml:"jsonrpc" json:"jsonrpc"`
	Grpc    int64 `toml:"grpc" json:"grpc"`
	Gateway int64 `toml:"gateway" json:"gateway"`
	PProf   int64 `toml:"pprof" json:"pprof"`
	Monitor int64 `toml:"monitor" json:"monitor"`
}

type Monitor struct {
	Enable bool
}

type PProf struct {
	Enable   bool          `toml:"enbale" json:"enable"`
	PType    string        `toml:"ptype" json:"ptype"`
	Mode     string        `toml:"mode" json:"mode"`
	Duration time.Duration `toml:"duration" json:"duration"`
}

type Limiter struct {
	Interval time.Duration `toml:"interval" json:"interval"`
	Quantum  int64         `toml:"quantum" json:"quantum"`
	Capacity int64         `toml:"capacity" json:"capacity"`
}

type Appchain struct {
	Enable        bool   `toml:"enable" json:"enable"`
	EthHeaderPath string `mapstructure:"eth_header_path"`
}

type Gateway struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type Ping struct {
	Enable   bool          `toml:"enable" json:"enable"`
	Duration time.Duration `toml:"duration" json:"duration"`
}

type Log struct {
	Level        string    `toml:"level" json:"level"`
	Dir          string    `toml:"dir" json:"dir"`
	Filename     string    `toml:"filename" json:"filename"`
	ReportCaller bool      `mapstructure:"report_caller" json:"report_caller"`
	Module       LogModule `toml:"module" json:"module"`
}

type LogModule struct {
	P2P       string `toml:"p2p" json:"p2p"`
	Consensus string `toml:"consensus" json:"consensus"`
	Executor  string `toml:"executor" json:"executor"`
	Router    string `toml:"router" json:"router"`
	API       string `toml:"api" json:"api"`
	CoreAPI   string `mapstructure:"coreapi" toml:"coreapi" json:"coreapi"`
	Storage   string `toml:"storage" json:"storage"`
	Profile   string `toml:"profile" json:"profile"`
}

type Genesis struct {
	ChainID  uint64            `json:"chainid" toml:"chainid"`
	GasLimit uint64            `mapstructure:"gas_limit" json:"gas_limit" toml:"gas_limit"`
	Balance  string            `json:"balance" toml:"balance"`
	Admins   []*Admin          `json:"admins" toml:"admins"`
	Strategy map[string]string `json:"strategy" toml:"strategy"`
	Dider    string            `json:"dider" toml:"dider"`
}

type Admin struct {
	Address string `json:"address" toml:"address"`
	Weight  uint64 `json:"weight" toml:"weight"`
}

type Cert struct {
	Verify         bool   `toml:"verify" json:"verify"`
	NodeCertPath   string `mapstructure:"node_cert_path" json:"node_cert_path"`
	AgencyCertPath string `mapstructure:"agency_cert_path" json:"agency_cert_path"`
	CACertPath     string `mapstructure:"ca_cert_path" json:"ca_cert_path"`
}

type Txpool struct {
	BatchSize    int           `mapstructure:"batch_size" json:"batch_size"`
	BatchTimeout time.Duration `mapstructure:"batch_timeout" json:"batch_timeout"`
}

type Order struct {
	Type string `toml:"type" json:"type"`
}

type Executor struct {
	Type string `toml:"type" json:"type"`
}

type Ledger struct {
	Type string `toml:"type" json:"type"`
}

type License struct {
	Key      string `toml:"key" json:"key"`
	Verifier string `toml:"verifier" json:"verifier"`
}

func (c *Config) Bytes() ([]byte, error) {
	ret, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func DefaultConfig() (*Config, error) {
	return &Config{
		Title: "BitXHub configuration file",
		Solo:  false,
		Port: Port{
			Grpc:    60011,
			Gateway: 9091,
			PProf:   53121,
			Monitor: 40011,
		},
		PProf:   PProf{Enable: false},
		Ping:    Ping{Enable: false},
		Gateway: Gateway{AllowedOrigins: []string{"*"}},
		Log: Log{
			Level:    "info",
			Dir:      "logs",
			Filename: "bitxhub.log",
			Module: LogModule{
				P2P:       "info",
				Consensus: "debug",
				Executor:  "info",
				Router:    "info",
				API:       "info",
				CoreAPI:   "info",
			},
		},
		Cert: Cert{
			Verify:         true,
			NodeCertPath:   "certs/node.cert",
			AgencyCertPath: "certs/agency.cert",
			CACertPath:     "certs/ca.cert",
		},
		Txpool: Txpool{
			BatchSize:    500,
			BatchTimeout: 500 * time.Millisecond,
		},
		Order: Order{
			Type: "raft",
		},
		Executor: Executor{
			Type: "serial",
		},
		Genesis: Genesis{
			ChainID:  1,
			GasLimit: 0x5f5e100,
			Balance:  "100000000000000000000000000000000000",
		},
		Ledger: Ledger{Type: "complex"},
	}, nil
}

func UnmarshalConfig(repoRoot string) (*Config, error) {
	viper.SetConfigFile(filepath.Join(repoRoot, configName))
	viper.SetConfigType("toml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("BITXHUB")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	config, err := DefaultConfig()
	if err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	config.RepoRoot = repoRoot
	return config, nil
}

func WatchConfig(feed *event.Feed) {
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("config file changed: ", in.String())

		config, err := DefaultConfig()
		if err != nil {
			fmt.Println("get default config: ", err)
			return
		}

		if err := viper.Unmarshal(config); err != nil {
			fmt.Println("unmarshal config: ", err)
			return
		}

		feed.Send(config)
	})
}

func ReadConfig(path, configType string, config interface{}) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType(configType)
	if err := v.ReadInConfig(); err != nil {
		return err
	}

	if err := v.Unmarshal(config); err != nil {
		return err
	}

	return nil
}

func PathRoot() (string, error) {
	dir := os.Getenv(envDir)
	var err error
	if len(dir) == 0 {
		dir, err = homedir.Expand(defaultPathRoot)
	}
	return dir, err
}

func PathRootWithDefault(path string) (string, error) {
	if len(path) == 0 {
		return PathRoot()
	}

	return path, nil
}
