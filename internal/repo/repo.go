package repo

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/ethereum/go-ethereum/event"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/spf13/viper"
)

type Repo struct {
	Config           *Config
	NetworkConfig    *NetworkConfig
	Key              *Key
	Certs            *libp2pcert.Certs
	ConfigChangeFeed event.Feed
}

func (r *Repo) SubscribeConfigChange(ch chan *Repo) event.Subscription {
	return r.ConfigChangeFeed.Subscribe(ch)
}

func Load(repoRoot string, passwd string, configPath string, networkPath string) (*Repo, error) {
	bViper := viper.New()
	nViper := viper.New()
	config, err := UnmarshalConfig(bViper, repoRoot, configPath)
	if err != nil {
		return nil, fmt.Errorf("unmarshal bitxhub config error: %w", err)
	}

	if err := checkConfig(config); err != nil {
		return nil, fmt.Errorf("check bitxhub config failed: %w", err)
	}

	var networkConfig *NetworkConfig
	if len(networkPath) == 0 {
		networkConfig, err = loadNetworkConfig(nViper, repoRoot, config.Genesis)
	} else {
		fileData, err := ioutil.ReadFile(networkPath)
		if err != nil {
			return nil, fmt.Errorf("read network config error: %w", err)
		}
		err = ioutil.WriteFile(filepath.Join(repoRoot, "network.toml"), fileData, 0644)
		if err != nil {
			return nil, fmt.Errorf("write network config failed: %w", err)
		}
		networkDir := filepath.Dir(networkPath)
		networkConfig, err = loadNetworkConfig(nViper, networkDir, config.Genesis)
	}
	if err != nil {
		return nil, fmt.Errorf("load network config: %w", err)
	}

	certs, err := libp2pcert.LoadCerts(repoRoot, config.NodeCertPath, config.AgencyCertPath, config.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("load certs failed: %w", err)
	}

	key, err := loadPrivKey(repoRoot, passwd)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	repo := &Repo{
		Config:        config,
		NetworkConfig: networkConfig,
		Key:           key,
		Certs:         certs,
	}

	// watch bitxhub.toml on changed
	WatchBitxhubConfig(bViper, &repo.ConfigChangeFeed)

	// watch network.toml on changed
	WatchNetworkConfig(nViper, &repo.ConfigChangeFeed)

	return repo, nil
}

func checkConfig(config *Config) error {
	// check genesis admin info
	hasSuperAdmin := false
	for _, admin := range config.Genesis.Admins {
		if admin.Weight == SuperAdminWeight {
			hasSuperAdmin = true
		} else if admin.Weight != NormalAdminWeight {
			return fmt.Errorf("Illegal admin weight in genesis config!")
		}
	}

	if !hasSuperAdmin {
		return fmt.Errorf("Set up at least one super administrator in genesis config!")
	}

	for _, s := range config.Genesis.Strategy {
		if err := CheckStrategyInfo(s.Typ, s.Module, s.ParticipateThreshold); err != nil {
			return err
		}
	}
	return nil
}

func CheckStrategyInfo(typ, module string, threshold float64) error {
	if CheckStrategyType(typ, threshold) != nil {
		return fmt.Errorf("illegal proposal strategy type:%s(threshold: %f)", typ, threshold)
	}
	if CheckManageModule(module) != nil {
		return fmt.Errorf("illegal proposal strategy module:%s", typ)
	}
	return nil
}

func CheckStrategyType(pst string, threshold float64) error {
	if pst != SuperMajorityApprove &&
		pst != SuperMajorityAgainst &&
		pst != SimpleMajority &&
		pst != ZeroPermission {
		return fmt.Errorf("illegal proposal strategy type")
	}

	if pst == ZeroPermission {
		if threshold != 0 {
			return fmt.Errorf("illegal ZeroPermission[participate_threshold]=%f", threshold)
		}
	} else {
		if threshold <= 0 || threshold > 1 {
			return fmt.Errorf("illegal SimpleMajority[participate_threshold]=%f", threshold)
		}
	}
	return nil
}

func CheckManageModule(moduleTyp string) error {
	if moduleTyp != AppchainMgr &&
		moduleTyp != RoleMgr &&
		moduleTyp != RuleMgr &&
		moduleTyp != DappMgr &&
		moduleTyp != ProposalStrategyMgr &&
		moduleTyp != NodeMgr &&
		moduleTyp != ServiceMgr {
		return fmt.Errorf("illegal manage module type")
	}
	return nil
}

func GetAPI(repoRoot string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join(repoRoot, APIName))
	if err != nil {
		return "", fmt.Errorf("read %s error: %w", filepath.Join(repoRoot, APIName), err)
	}

	return string(data), nil
}

func GetKeyPath(repoRoot string) string {
	return filepath.Join(repoRoot, KeyName)
}

func GetStoragePath(repoRoot string, subPath ...string) string {
	p := filepath.Join(repoRoot, "storage")
	for _, s := range subPath {
		p = filepath.Join(p, s)
	}

	return p
}
