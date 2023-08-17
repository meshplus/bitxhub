package repo

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	rbft "github.com/axiomesh/axiom-bft"
)

type Repo struct {
	Config      *Config
	OrderConfig *OrderConfig
	NodeKey     *ecdsa.PrivateKey
	P2PKey      libp2pcrypto.PrivKey
	P2PID       string

	// TODO: use another account
	NodeAddress string

	ConfigChangeFeed event.Feed

	// TODO: Move to epoch manager service
	// Track current epoch info, will be updated bt executor
	EpochInfo *rbft.EpochInfo
}

type signerOpts struct {
}

func (*signerOpts) HashFunc() crypto.Hash {
	return crypto.SHA3_256
}

var signOpt = &signerOpts{}

func (r *Repo) NodeKeySign(data []byte) ([]byte, error) {
	return r.NodeKey.Sign(rand.Reader, data, signOpt)
}

// TODO: need support? remove it
func (r *Repo) SubscribeConfigChange(ch chan *Repo) event.Subscription {
	return r.ConfigChangeFeed.Subscribe(ch)
}

func (r *Repo) Flush() error {
	if err := writeConfigWithEnv(path.Join(r.Config.RepoRoot, CfgFileName), r.Config); err != nil {
		return errors.Wrap(err, "failed to write config")
	}
	if err := writeConfigWithEnv(path.Join(r.Config.RepoRoot, orderCfgFileName), r.OrderConfig); err != nil {
		return errors.Wrap(err, "failed to write order config")
	}
	if err := WriteKey(path.Join(r.Config.RepoRoot, nodeKeyFileName), r.NodeKey); err != nil {
		return errors.Wrap(err, "failed to write node key")
	}
	return nil
}

func writeConfigWithEnv(cfgPath string, config any) error {
	if err := writeConfig(cfgPath, config); err != nil {
		return err
	}
	// write back environment variables first
	// TODO: wait viper support read from environment variables
	if err := readConfigFromFile(cfgPath, config); err != nil {
		return errors.Wrapf(err, "failed to read cfg from environment")
	}
	if err := writeConfig(cfgPath, config); err != nil {
		return err
	}
	return nil
}

func writeConfig(cfgPath string, config any) error {
	raw, err := MarshalConfig(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(cfgPath, []byte(raw), 0755); err != nil {
		return err
	}

	return nil
}

func MarshalConfig(config any) (string, error) {
	buf := bytes.NewBuffer([]byte{})
	e := toml.NewEncoder(buf)
	e.SetIndentTables(true)
	e.SetArraysMultiline(true)
	err := e.Encode(config)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func Default(repoRoot string) (*Repo, error) {
	return DefaultWithNodeIndex(repoRoot, 0)
}

func DefaultWithNodeIndex(repoRoot string, nodeIndex int) (*Repo, error) {
	var key *ecdsa.PrivateKey
	var err error
	if nodeIndex < 0 || nodeIndex > len(DefaultNodeKeys)-1 {
		key, err = GenerateKey()
		nodeIndex = 0
	} else {
		key, err = ParseKey([]byte(DefaultNodeKeys[nodeIndex]))
	}
	if err != nil {
		return nil, err
	}

	p2pKey, err := P2PKeyFromECDSAKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ecdsa key : %w", err)
	}
	addr := ethcrypto.PubkeyToAddress(key.PublicKey)
	id, err := KeyToNodeID(key)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig(repoRoot)
	cfg.Port.P2P = int64(4001 + nodeIndex)

	return &Repo{
		Config:      cfg,
		OrderConfig: DefaultOrderConfig(),
		NodeKey:     key,
		P2PKey:      p2pKey,
		P2PID:       id,
		NodeAddress: addr.String(),
		EpochInfo:   cfg.Genesis.EpochInfo,
	}, nil
}

// load config from the repo, which is automatically initialized when the repo is empty
func Load(repoRoot string) (*Repo, error) {
	repoRoot, err := LoadRepoRootFromEnv(repoRoot)
	if err != nil {
		return nil, err
	}

	key, err := LoadNodeKey(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load node key: %w", err)
	}
	p2pKey, err := P2PKeyFromECDSAKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ecdsa key : %w", err)
	}
	addr := ethcrypto.PubkeyToAddress(key.PublicKey)
	id, err := KeyToNodeID(key)
	if err != nil {
		return nil, err
	}

	cfg, err := LoadConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	orderCfg, err := LoadOrderConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	repo := &Repo{
		Config:           cfg,
		OrderConfig:      orderCfg,
		NodeKey:          key,
		P2PKey:           p2pKey,
		P2PID:            id,
		NodeAddress:      addr.String(),
		ConfigChangeFeed: event.Feed{},
		EpochInfo:        cfg.Genesis.EpochInfo,
	}

	return repo, nil
}

func GetStoragePath(repoRoot string, subPath ...string) string {
	p := filepath.Join(repoRoot, "storage")
	for _, s := range subPath {
		p = filepath.Join(p, s)
	}

	return p
}

func LoadRepoRootFromEnv(repoRoot string) (string, error) {
	if repoRoot != "" {
		return repoRoot, nil
	}
	repoRoot = os.Getenv(rootPathEnvVar)
	var err error
	if len(repoRoot) == 0 {
		repoRoot, err = homedir.Expand(defaultRepoRoot)
	}
	return repoRoot, err
}

func readConfigFromFile(cfgFilePath string, config any) error {
	vp := viper.New()
	vp.SetConfigFile(cfgFilePath)
	vp.SetConfigType("toml")
	return readConfig(vp, config)
}

func readConfig(vp *viper.Viper, config any) error {
	vp.AutomaticEnv()
	vp.SetEnvPrefix("AXIOM")
	replacer := strings.NewReplacer(".", "_")
	vp.SetEnvKeyReplacer(replacer)

	err := vp.ReadInConfig()
	if err != nil {
		return err
	}

	if err := vp.Unmarshal(config, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(";"),
	))); err != nil {
		return err
	}

	return nil
}

func WritePid(rootPath string) error {
	pid := os.Getpid()
	pidStr := strconv.Itoa(pid)
	if err := os.WriteFile(filepath.Join(rootPath, pidFileName), []byte(pidStr), 0755); err != nil {
		return errors.Wrap(err, "failed to write pid file")
	}
	return nil
}

func RemovePID(rootPath string) error {
	return os.Remove(filepath.Join(rootPath, pidFileName))
}

func WriteDebugInfo(rootPath string, debugInfo any) error {
	p := filepath.Join(rootPath, debugFileName)
	_ = os.Remove(p)

	raw, err := json.Marshal(debugInfo)
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, raw, 0755); err != nil {
		return errors.Wrap(err, "failed to write debug info file")
	}
	return nil
}

func CheckWritable(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		// dir exists, make sure we can write to it
		testfile := filepath.Join(dir, "test")
		fi, err := os.Create(testfile)
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("%s is not writeable by the current user", dir)
			}
			return fmt.Errorf("unexpected error while checking writeablility of repo root: %s", err)
		}
		fi.Close()
		return os.Remove(testfile)
	}

	if os.IsNotExist(err) {
		// dir doesn't exist, check that we can create it
		return os.Mkdir(dir, 0775)
	}

	if os.IsPermission(err) {
		return fmt.Errorf("cannot write to %s, incorrect permissions", err)
	}

	return err
}
