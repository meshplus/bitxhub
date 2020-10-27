package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
)

type Config struct {
	RepoRoot string `json:"repo_root"`
	Title    string `json:"title"`
	Solo     bool   `json:"solo"`
	Port     `json:"port"`
	PProf    `json:"pprof"`
	Monitor  `json:"monitor"`
	Gateway  `json:"gateway"`
	Log      `json:"log"`
	Cert     `json:"cert"`
	Txpool   `json:"txpool"`
	Order    `json:"order"`
	Executor `json:"executor"`
	Security Security `toml:"security" json:"security"`
}

// Security are files used to setup connection with tls
type Security struct {
	EnableTLS     bool   `mapstructure:"enable_tls"`
	PemFilePath   string `mapstructure:"pem_file_path" json:"pem_file_path"`
	ServerKeyPath string `mapstructure:"server_key_path" json:"server_key_path"`
}

type Port struct {
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

type Gateway struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
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
}

type Genesis struct {
	Addresses []string `json:"addresses" toml:"addresses"`
}

type Cert struct {
	Verify bool `toml:"verify" json:"verify"`
}

type Txpool struct {
	BatchSize    int           `mapstructure:"batch_size" json:"batch_size"`
	BatchTimeout time.Duration `mapstructure:"batch_timeout" json:"batch_timeout"`
}

type Order struct {
	Plugin string `toml:"plugin" json:"plugin"`
}

type Executor struct {
	Type string `toml:"type" json:"type"`
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
		Cert: Cert{Verify: true},
		Txpool: Txpool{
			BatchSize:    500,
			BatchTimeout: 500 * time.Millisecond,
		},
		Order: Order{
			Plugin: "plugins/raft.so",
		},
		Executor: Executor{
			Type: "serial",
		},
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

func loadGenesis(repoRoot string) (*Genesis, error) {
	genesis := &Genesis{}
	if err := ReadConfig(filepath.Join(repoRoot, "genesis.json"), "json", genesis); err != nil {
		return nil, err
	}

	if len(genesis.Addresses) == 0 {
		return nil, fmt.Errorf("wrong genesis address number")
	}

	return genesis, nil
}
